package migrations_test

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	appmigrations "github.com/sidnevart/proof-forge/backend/migrations"
)

func TestMigrationRoundTrip_UpDownUp(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("MIGRATION_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("MIGRATION_DATABASE_URL is not set; skipping destructive migration round-trip")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open migration db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	goose.SetBaseFS(appmigrations.Files)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}

	if err := goose.UpContext(ctx, db, "."); err != nil {
		t.Fatalf("goose up: %v", err)
	}
	if err := goose.DownToContext(ctx, db, ".", 0); err != nil {
		t.Fatalf("goose down to 0: %v", err)
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		t.Fatalf("goose up after down: %v", err)
	}
}
