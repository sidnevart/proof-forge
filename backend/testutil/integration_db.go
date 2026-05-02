package testutil

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	appmigrations "github.com/sidnevart/proof-forge/backend/migrations"
)

func OpenIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open stdlib test db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	goose.SetBaseFS(appmigrations.Files)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if logger != nil {
		logger.Info("running integration test migrations")
	}
	if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	poolCfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse test database url: %v", err)
	}
	poolCfg.MaxConns = 4
	poolCfg.MinConns = 1
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 10 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		t.Fatalf("open test pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping test pool: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		TRUNCATE TABLE
			milestones,
			stake_forfeitures,
			stakes,
			weekly_recaps,
			check_in_reviews,
			evidence_items,
			check_ins,
			invites,
			pacts,
			goals,
			user_sessions,
			users
		RESTART IDENTITY CASCADE
	`); err != nil {
		pool.Close()
		t.Fatalf("truncate test data: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}
