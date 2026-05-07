package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sidnevart/proof-forge/backend/internal/ai"
	"github.com/sidnevart/proof-forge/backend/internal/analytics"
	"github.com/sidnevart/proof-forge/backend/internal/checkins"
	"github.com/sidnevart/proof-forge/backend/internal/circles"
	"github.com/sidnevart/proof-forge/backend/internal/dailylog"
	"github.com/sidnevart/proof-forge/backend/internal/goals"
	"github.com/sidnevart/proof-forge/backend/internal/inspiration"
	"github.com/sidnevart/proof-forge/backend/internal/notifications"
	"github.com/sidnevart/proof-forge/backend/internal/personalization"
	platformconfig "github.com/sidnevart/proof-forge/backend/internal/platform/config"
	"github.com/sidnevart/proof-forge/backend/internal/platform/email"
	"github.com/sidnevart/proof-forge/backend/internal/platform/httpx"
	platformlogger "github.com/sidnevart/proof-forge/backend/internal/platform/logger"
	"github.com/sidnevart/proof-forge/backend/internal/platform/postgres"
	"github.com/sidnevart/proof-forge/backend/internal/platform/readiness"
	"github.com/sidnevart/proof-forge/backend/internal/recaps"
	"github.com/sidnevart/proof-forge/backend/internal/teamproof"
	"github.com/sidnevart/proof-forge/backend/internal/teams"
	"github.com/sidnevart/proof-forge/backend/internal/telegram"
	"github.com/sidnevart/proof-forge/backend/internal/telegram/bot"
	"github.com/sidnevart/proof-forge/backend/internal/telegram/callbacks"
	"github.com/sidnevart/proof-forge/backend/internal/telegram/commands"
	tgdailylog "github.com/sidnevart/proof-forge/backend/internal/telegram/dailylog"
	"github.com/sidnevart/proof-forge/backend/internal/users"
)

func RunAPI(ctx context.Context, cfg platformconfig.Config) error {
	log := platformlogger.WithComponent(platformlogger.New(cfg.Log), "api")

	if cfg.DB.RunMigrations {
		sqlDB, err := postgres.OpenStdlib(cfg.DB)
		if err != nil {
			return err
		}
		defer sqlDB.Close()

		if err := postgres.Up(ctx, sqlDB, log); err != nil {
			return err
		}
	}

	pool, err := postgres.Open(ctx, cfg.DB)
	if err != nil {
		return err
	}
	defer pool.Close()

	readinessService := readiness.NewService(pool, cfg.DB.HealthcheckTimeout)
	router := httpx.NewRouter(log, readinessService, cfg.App.WebOrigin)
	registerAPIRoutes(router, log, pool, cfg)

	server := &http.Server{
		Addr:              cfg.HTTP.Address(),
		Handler:           router,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("api server starting", "addr", cfg.HTTP.Address())
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		log.Info("api server shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown api server: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}

func registerAPIRoutes(router *chi.Mux, log *slog.Logger, pool *pgxpool.Pool, cfg platformconfig.Config) {
	usersRepo := users.NewPostgresRepository(pool)
	usersSvc := users.NewService(
		usersRepo, usersRepo, cfg.Session.TTL,
		users.WithRefreshRepo(usersRepo),
		users.WithRefreshTTL(cfg.Session.RefreshTTL),
	)
	usersHandler := users.NewHandlerWithConfig(users.HandlerConfig{
		Log:               platformlogger.WithComponent(log, "users"),
		Service:           usersSvc,
		CookieName:        cfg.Session.CookieName,
		RefreshCookieName: cfg.Session.RefreshCookieName,
		CookieDomain:      cfg.Session.CookieDomain,
		SecureCookie:      cfg.App.Env == "production",
	})

	var emailSender email.Sender
	if cfg.SMTP.Enabled {
		emailSender = email.NewSMTPSender(cfg.SMTP)
	} else {
		emailSender = email.NoopSender{}
	}

	var goalRefineProvider ai.RefineProvider
	if cfg.AI.Enabled {
		goalRefineProvider = ai.NewOpenAIRefineProvider(cfg.AI.BaseURL, cfg.AI.APIKey, cfg.AI.Model)
	} else {
		goalRefineProvider = ai.NewFakeGoalRefineProvider()
	}
	notifRepo := notifications.NewPostgresRepository(pool)

	circlesService := circles.NewService(circles.NewPostgresRepository(pool))
	goalsHandler := goals.NewHandler(
		platformlogger.WithComponent(log, "goals"),
		goals.NewService(
			goals.NewPostgresRepository(pool),
			emailSender,
			cfg.App.WebOrigin,
			platformlogger.WithComponent(log, "goals"),
			cfg.Invite.TTL,
			goals.WithRefineProvider(goalRefineProvider),
			goals.WithNotifEmitter(notifRepo),
			goals.WithCircleLister(circlesService),
		),
	)
	circlesHandler := circles.NewHandler(circlesService)

	teamsService := teams.NewService(teams.NewPostgresRepository(pool))
	teamsHandler := teams.NewHandler(teamsService)

	analyticsRecorder := analytics.NewPostgresRecorder(pool)
	analyticsHandler := analytics.NewHandler(analyticsRecorder)

	dailylogService := dailylog.NewService(dailylog.NewPostgresRepository(pool), dailylog.WithRecorder(analyticsRecorder))
	dailylogHandler := dailylog.NewHandler(dailylogService)

	teamproofService := teamproof.NewService(teamproof.NewPostgresRepository(pool), teamproof.WithRecorder(analyticsRecorder))
	teamproofHandler := teamproof.NewHandler(teamproofService)

	var llmProvider *personalization.LLMProvider
	if cfg.AI.Enabled {
		llmProvider = personalization.NewLLMProvider(cfg.AI.BaseURL, cfg.AI.APIKey, cfg.AI.Model)
	}
	persService := personalization.NewService(
		cfg.AI.Enabled,
		personalization.NewPostgresCircuitBreakerStore(pool),
		personalization.NewPostgresBudgetStore(pool),
		llmProvider,
		personalization.WithRecorder(analyticsRecorder),
	)
	persHandler := personalization.NewHandler(persService, pool, platformlogger.WithComponent(log, "personalization"))

	var objStorage checkins.Storage
	if cfg.Storage.Enabled {
		objStorage = checkins.NewS3Storage(checkins.S3Config{
			Endpoint:        cfg.Storage.Endpoint,
			Region:          cfg.Storage.Region,
			Bucket:          cfg.Storage.Bucket,
			AccessKeyID:     cfg.Storage.AccessKeyID,
			SecretAccessKey: cfg.Storage.SecretAccessKey,
			UsePathStyle:    cfg.Storage.UsePathStyle,
		})
	} else {
		objStorage = checkins.NoopStorage{}
	}

	checkinsSvc := checkins.NewService(checkins.NewPostgresRepository(pool), objStorage, notifRepo).
		WithMembershipChecker(circlesService)
	checkinsHandler := checkins.NewHandler(
		platformlogger.WithComponent(log, "checkins"),
		checkinsSvc,
	)

	var aiProvider recaps.AIProvider
	if cfg.AI.Enabled {
		aiProvider = recaps.NewOpenAIProvider(cfg.AI.BaseURL, cfg.AI.APIKey, cfg.AI.Model)
	} else {
		aiProvider = recaps.NoopProvider{}
	}
	recapsHandler := recaps.NewHandler(
		platformlogger.WithComponent(log, "recaps"),
		recaps.NewService(recaps.NewPostgresRepository(pool), aiProvider, platformlogger.WithComponent(log, "recaps")),
	)

	if cfg.Telegram.Enabled {
		var sender bot.Sender
		sender = bot.New(cfg.Telegram.BotToken)

		tgRepo := telegram.NewRepository(pool)
		notifDispatcher := notifications.NewDispatcher(sender, notifRepo, platformlogger.WithComponent(log, "notifications"))

		startCmd := commands.NewStartHandler(tgRepo, sender, platformlogger.WithComponent(log, "telegram.start"))
		cmdHandler := commands.NewCommandHandler(
			tgRepo, notifRepo, sender, cfg.App.WebOrigin,
			platformlogger.WithComponent(log, "telegram.commands"),
		)
		reviewCb := callbacks.NewReviewHandler(checkinsSvc, tgRepo, sender, platformlogger.WithComponent(log, "telegram.review"))

		dlCallback := tgdailylog.NewCallbackHandler(dailylogService, tgRepo, sender, platformlogger.WithComponent(log, "telegram.dailylog"))
		dlMsgHandler := tgdailylog.NewMessageHandler(dailylogService, tgRepo, sender, platformlogger.WithComponent(log, "telegram.dailylog"))

		telegramHandler := telegram.NewHandler(telegram.HandlerConfig{
			Secret:             cfg.Telegram.WebhookSecret,
			StartHandler:       startCmd,
			CommandHandler:     cmdHandler,
			ReviewCallback:     reviewCb,
			DailyLogCallback:   dlCallback,
			DailyLogMsgHandler: dlMsgHandler,
			Log:                platformlogger.WithComponent(log, "telegram"),
		})
		telegramHandler.RegisterRoutes(router)

		_ = notifDispatcher // used by worker; here for compile-time wiring check
	}

	inspRepo := inspiration.NewPostgresRepository(pool)
	inspSvc := inspiration.NewService(inspRepo)
	inspHandler := inspiration.NewHandler(platformlogger.WithComponent(log, "inspiration"), inspSvc)

	router.Route("/v1", func(r chi.Router) {
		usersHandler.RegisterPublicRoutes(r)
		goalsHandler.RegisterPublicRoutes(r)
		inspHandler.RegisterPublicRoutes(r)
		r.Group(func(r chi.Router) {
			r.Use(usersHandler.AuthMiddleware)
			usersHandler.RegisterProtectedRoutes(r)
			circlesHandler.RegisterRoutes(r)
			teamsHandler.RegisterRoutes(r)
			dailylogHandler.RegisterRoutes(r)
			teamproofHandler.RegisterRoutes(r)
			persHandler.RegisterRoutes(r)
			goalsHandler.RegisterRoutes(r)
			checkinsHandler.RegisterRoutes(r)
			recapsHandler.RegisterRoutes(r)
			inspHandler.RegisterRoutes(r)
			analyticsHandler.RegisterRoutes(r)

			// Telegram link token endpoint.
			if cfg.Telegram.Enabled {
				tgRepo := telegram.NewRepository(pool)
				botUsername := cfg.Telegram.BotUsername
				r.Post("/telegram/link-token", makeLinkTokenHandler(tgRepo, botUsername, log))
			}
		})
	})
}

func makeLinkTokenHandler(repo *telegram.Repository, botUsername string, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := users.CurrentUser(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		token, err := commands.GenerateToken(r.Context(), repo, actor.ID)
		if err != nil {
			log.Error("link token: generate", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		deeplink := fmt.Sprintf("https://t.me/%s?start=link_%s", botUsername, token)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"token":    token,
			"deeplink": deeplink,
		})
	}
}
