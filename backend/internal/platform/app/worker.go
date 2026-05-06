package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/ai"
	"github.com/sidnevart/proof-forge/backend/internal/inspiration"
	"github.com/sidnevart/proof-forge/backend/internal/notifications"
	platformconfig "github.com/sidnevart/proof-forge/backend/internal/platform/config"
	platformlogger "github.com/sidnevart/proof-forge/backend/internal/platform/logger"
	"github.com/sidnevart/proof-forge/backend/internal/platform/postgres"
	"github.com/sidnevart/proof-forge/backend/internal/recaps"
	"github.com/sidnevart/proof-forge/backend/internal/telegram/bot"
)

func RunWorker(ctx context.Context, cfg platformconfig.Config) error {
	log := platformlogger.WithComponent(platformlogger.New(cfg.Log), "worker")

	pool, err := postgres.Open(ctx, cfg.DB)
	if err != nil {
		return err
	}
	defer pool.Close()

	var aiProvider recaps.AIProvider
	if cfg.AI.Enabled {
		aiProvider = recaps.NewOpenAIProvider(cfg.AI.BaseURL, cfg.AI.APIKey, cfg.AI.Model)
	} else {
		aiProvider = recaps.NoopProvider{}
	}

	recapsSvc := recaps.NewService(
		recaps.NewPostgresRepository(pool),
		aiProvider,
		platformlogger.WithComponent(log, "recaps"),
	)

	// Embeddings worker — always run when AI is enabled.
	if cfg.AI.Enabled {
		embedProvider := ai.NewOpenAIEmbedProvider(cfg.AI.BaseURL, cfg.AI.APIKey)
		inspRepo := inspiration.NewPostgresRepository(pool)
		embWorker := inspiration.NewEmbeddingsWorker(inspRepo, embedProvider, platformlogger.WithComponent(log, "embeddings"))
		go embWorker.Run(ctx)
		log.Info("embeddings worker started")
	}

	// Set up notification workers if Telegram is enabled.
	if cfg.Telegram.Enabled {
		var sender bot.Sender = bot.New(cfg.Telegram.BotToken)

		notifRepo := notifications.NewPostgresRepository(pool)
		dispatcher := notifications.NewDispatcher(sender, notifRepo, platformlogger.WithComponent(log, "notifications"))

		digestWorker := notifications.NewDigestWorker(notifRepo, dispatcher, platformlogger.WithComponent(log, "digest"))
		nudgeEngine := notifications.NewNudgeEngine(notifRepo, dispatcher, platformlogger.WithComponent(log, "nudge"))

		go digestWorker.Run(ctx)
		go nudgeEngine.Run(ctx)

		log.Info("notification workers started")
	}

	log.Info("worker started", "recap_sweep_interval", cfg.Worker.RecapSweepInterval.String())

	ticker := time.NewTicker(cfg.Worker.RecapSweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("worker shutting down")
			return nil
		case <-ticker.C:
			if err := runRecapSweep(ctx, log, recapsSvc); err != nil {
				return err
			}
		}
	}
}

func runRecapSweep(ctx context.Context, log *slog.Logger, svc *recaps.Service) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("worker context canceled: %w", ctx.Err())
	default:
	}

	log.Info("worker tick", "job", "weekly_recap_sweep")
	if err := svc.SweepAndGenerate(ctx, time.Now()); err != nil {
		log.Error("recap sweep failed", "err", err)
	}
	return nil
}
