package config

import (
	"testing"
	"time"

	"github.com/sidnevart/proof-forge/backend/testutil"
)

func TestLoadSuccess(t *testing.T) {
	t.Setenv("APP_NAME", "ProofForge")
	t.Setenv("APP_ENV", "test")
	t.Setenv("WEB_ORIGIN", "http://localhost:3000")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/proofforge?sslmode=disable")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("DB_MAX_CONNS", "12")
	t.Setenv("WORKER_RECAP_SWEEP_INTERVAL", "2m")
	t.Setenv("HTTP_READ_TIMEOUT", "20s")
	t.Setenv("SESSION_COOKIE_NAME", "proof_session")
	t.Setenv("SESSION_TTL", "24h")
	t.Setenv("INVITE_TTL", "72h")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTP.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.HTTP.Port)
	}
	if cfg.DB.MaxConns != 12 {
		t.Fatalf("expected max conns 12, got %d", cfg.DB.MaxConns)
	}
	if cfg.Worker.RecapSweepInterval != 2*time.Minute {
		t.Fatalf("expected worker interval 2m, got %s", cfg.Worker.RecapSweepInterval)
	}
	if cfg.HTTP.ReadTimeout != 20*time.Second {
		t.Fatalf("expected read timeout 20s, got %s", cfg.HTTP.ReadTimeout)
	}
	if cfg.Session.CookieName != "proof_session" {
		t.Fatalf("expected cookie name proof_session, got %s", cfg.Session.CookieName)
	}
	if cfg.Invite.TTL != 72*time.Hour {
		t.Fatalf("expected invite ttl 72h, got %s", cfg.Invite.TTL)
	}
}

func TestLoadMissingDatabaseURL(t *testing.T) {
	testutil.ClearEnv(t, "DATABASE_URL")

	_, err := Load()
	if err == nil {
		t.Fatal("expected validation error for missing DATABASE_URL")
	}
}
