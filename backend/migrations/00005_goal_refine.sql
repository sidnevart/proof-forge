-- +goose Up
ALTER TABLE goals
    ADD COLUMN IF NOT EXISTS proof_examples TEXT,
    ADD COLUMN IF NOT EXISTS category TEXT;

CREATE TABLE IF NOT EXISTS goal_refine_cache (
    hash TEXT PRIMARY KEY,
    response JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS goal_refine_requests (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    draft_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_goal_refine_requests_user_created
    ON goal_refine_requests(user_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_goal_refine_requests_user_created;
DROP TABLE IF EXISTS goal_refine_requests;
DROP TABLE IF EXISTS goal_refine_cache;

ALTER TABLE goals
    DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS proof_examples;
