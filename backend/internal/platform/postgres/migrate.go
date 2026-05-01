package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/pressly/goose/v3"

	appmigrations "github.com/sidnevart/proof-forge/backend/migrations"
)

func Up(ctx context.Context, db *sql.DB, log *slog.Logger) error {
	goose.SetBaseFS(appmigrations.Files)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if log != nil {
		log.Info("running database migrations")
	}

	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}
