package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sidnevart/proof-forge/backend/internal/checkins"
	"github.com/sidnevart/proof-forge/backend/internal/goals"
	platformconfig "github.com/sidnevart/proof-forge/backend/internal/platform/config"
	"github.com/sidnevart/proof-forge/backend/internal/platform/email"
	"github.com/sidnevart/proof-forge/backend/internal/platform/httpx"
	platformlogger "github.com/sidnevart/proof-forge/backend/internal/platform/logger"
	"github.com/sidnevart/proof-forge/backend/internal/platform/postgres"
	"github.com/sidnevart/proof-forge/backend/internal/platform/readiness"
	"github.com/sidnevart/proof-forge/backend/internal/recaps"
	"github.com/sidnevart/proof-forge/backend/internal/stakes"
	"github.com/sidnevart/proof-forge/backend/internal/telegram"
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
	usersHandler := users.NewHandler(
		platformlogger.WithComponent(log, "users"),
		users.NewService(usersRepo, usersRepo, cfg.Session.TTL),
		cfg.Session.CookieName,
		cfg.App.Env == "production",
	)
	var emailSender email.Sender
	if cfg.SMTP.Enabled {
		emailSender = email.NewSMTPSender(cfg.SMTP)
	} else {
		emailSender = email.NoopSender{}
	}

	goalsHandler := goals.NewHandler(
		platformlogger.WithComponent(log, "goals"),
		goals.NewService(goals.NewPostgresRepository(pool), emailSender, cfg.App.WebOrigin, platformlogger.WithComponent(log, "goals"), cfg.Invite.TTL),
	)

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
	checkinsHandler := checkins.NewHandler(
		platformlogger.WithComponent(log, "checkins"),
		checkins.NewService(checkins.NewPostgresRepository(pool), objStorage),
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

	stakesHandler := stakes.NewHandler(
		platformlogger.WithComponent(log, "stakes"),
		stakes.NewService(stakes.NewPostgresRepository(pool)),
	)

	if cfg.Telegram.Enabled {
		telegramHandler := telegram.NewHandler(
			cfg.Telegram.WebhookSecret,
			platformlogger.WithComponent(log, "telegram"),
		)
		telegramHandler.RegisterRoutes(router)
	}

	router.Route("/v1", func(r chi.Router) {
		usersHandler.RegisterPublicRoutes(r)
		goalsHandler.RegisterPublicRoutes(r)
		r.Group(func(r chi.Router) {
			r.Use(usersHandler.AuthMiddleware)
			usersHandler.RegisterProtectedRoutes(r)
			goalsHandler.RegisterRoutes(r)
			checkinsHandler.RegisterRoutes(r)
			recapsHandler.RegisterRoutes(r)
			stakesHandler.RegisterRoutes(r)
		})
	})
}
