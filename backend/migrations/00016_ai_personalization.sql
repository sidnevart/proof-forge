-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS ai_personalization_cache (
    cache_key TEXT PRIMARY KEY,
    feature TEXT NOT NULL,
    payload JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_cache_expires
    ON ai_personalization_cache(expires_at);

CREATE TABLE IF NOT EXISTS ai_daily_budget (
    budget_date DATE PRIMARY KEY,
    tokens_used BIGINT NOT NULL DEFAULT 0,
    requests_count INTEGER NOT NULL DEFAULT 0,
    fallback_count INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai_circuit_breakers (
    feature TEXT PRIMARY KEY,
    state TEXT NOT NULL CHECK (state IN ('closed', 'open')),
    opened_until TIMESTAMPTZ,
    reason TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ai_circuit_breakers;
DROP TABLE IF EXISTS ai_daily_budget;
DROP INDEX IF EXISTS idx_ai_cache_expires;
DROP TABLE IF EXISTS ai_personalization_cache;
-- +goose StatementEnd
