package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/sidnevart/proof-forge/backend/internal/platform/config"
)

func ParsePoolConfig(cfg config.DBConfig) (*pgxpool.Config, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse pgx pool config: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime

	return poolCfg, nil
}

func Open(ctx context.Context, cfg config.DBConfig) (*pgxpool.Pool, error) {
	poolCfg, err := ParsePoolConfig(cfg)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("open pgx pool: %w", err)
	}

	checkCtx, cancel := context.WithTimeout(ctx, cfg.HealthcheckTimeout)
	defer cancel()
	if err := pool.Ping(checkCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping pgx pool: %w", err)
	}

	return pool, nil
}

func OpenStdlib(cfg config.DBConfig) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("open stdlib db: %w", err)
	}

	db.SetMaxOpenConns(int(cfg.MaxConns))
	db.SetMaxIdleConns(int(cfg.MinConns))
	db.SetConnMaxLifetime(cfg.MaxConnLifetime)
	db.SetConnMaxIdleTime(cfg.MaxConnIdleTime)

	return db, nil
}
