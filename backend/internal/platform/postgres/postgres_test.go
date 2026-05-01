package postgres

import (
	"testing"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/platform/config"
)

func TestParsePoolConfig(t *testing.T) {
	cfg := config.DBConfig{
		URL:                "postgres://user:pass@localhost:5432/proofforge?sslmode=disable",
		MaxConns:           12,
		MinConns:           2,
		MaxConnLifetime:    time.Hour,
		MaxConnIdleTime:    10 * time.Minute,
		HealthcheckTimeout: time.Second,
	}

	poolCfg, err := ParsePoolConfig(cfg)
	if err != nil {
		t.Fatalf("ParsePoolConfig() error = %v", err)
	}

	if poolCfg.MaxConns != 12 {
		t.Fatalf("expected MaxConns 12, got %d", poolCfg.MaxConns)
	}
	if poolCfg.MinConns != 2 {
		t.Fatalf("expected MinConns 2, got %d", poolCfg.MinConns)
	}
}
