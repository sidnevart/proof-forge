package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	platformconfig "github.com/sidnevart/proof-forge/backend/internal/platform/config"
	platformlogger "github.com/sidnevart/proof-forge/backend/internal/platform/logger"
	"github.com/sidnevart/proof-forge/backend/internal/platform/postgres"
	"github.com/sidnevart/proof-forge/backend/internal/recaps"
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
