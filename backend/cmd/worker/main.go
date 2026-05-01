package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/sidnevart/proof-forge/backend/internal/platform/app"
	"github.com/sidnevart/proof-forge/backend/internal/platform/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.RunWorker(ctx, cfg); err != nil {
		log.Fatalf("run worker: %v", err)
	}
}
