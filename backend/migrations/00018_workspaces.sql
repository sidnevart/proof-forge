-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS workspaces (
    id          BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name        TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 80),
    slug        TEXT NOT NULL CHECK (slug ~ '^[a-z0-9][a-z0-9\-]{1,48}[a-z0-9]$'),
    type        TEXT NOT NULL CHECK (type IN ('organization', 'community')),
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    frozen_at   TIMESTAMPTZ,
    frozen_reason TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT workspaces_slug_unique UNIQUE (slug)
);

CREATE INDEX IF NOT EXISTS idx_workspaces_owner ON workspaces(owner_user_id) WHERE is_active = TRUE;

-- Link existing teams to workspaces (nullable for backward compat).
ALTER TABLE teams
    ADD COLUMN IF NOT EXISTS workspace_id BIGINT REFERENCES workspaces(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_teams_workspace ON teams(workspace_id) WHERE workspace_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_teams_workspace;
ALTER TABLE teams DROP COLUMN IF EXISTS workspace_id;
DROP INDEX IF EXISTS idx_workspaces_owner;
DROP TABLE IF EXISTS workspaces;

-- +goose StatementEnd
