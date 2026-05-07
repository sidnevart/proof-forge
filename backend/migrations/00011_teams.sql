-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS teams (
    id BIGSERIAL PRIMARY KEY,
    lead_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 80),
    invite_code TEXT NOT NULL UNIQUE,
    member_limit INTEGER NOT NULL DEFAULT 25 CHECK (member_limit BETWEEN 2 AND 100),
    ai_mode TEXT NOT NULL DEFAULT 'metadata-only'
        CHECK (ai_mode IN ('off', 'metadata-only', 'full')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS team_memberships (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('lead', 'trusted_approver', 'member')),
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'left', 'removed')),
    ai_consent BOOLEAN NOT NULL DEFAULT FALSE,
    timezone TEXT NOT NULL DEFAULT 'Europe/Moscow',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    UNIQUE (team_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_teams_lead ON teams(lead_user_id) WHERE archived_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_team_memberships_user ON team_memberships(user_id, status);
CREATE INDEX IF NOT EXISTS idx_team_memberships_team_role ON team_memberships(team_id, role) WHERE status = 'active';

-- Ровно один активный lead на команду — partial unique index.
-- Защищает от двух одновременных lead-membership через uniqueness, а не trigger.
CREATE UNIQUE INDEX IF NOT EXISTS uq_team_one_lead ON team_memberships(team_id)
    WHERE role = 'lead' AND status = 'active';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS uq_team_one_lead;
DROP INDEX IF EXISTS idx_team_memberships_team_role;
DROP INDEX IF EXISTS idx_team_memberships_user;
DROP INDEX IF EXISTS idx_teams_lead;
DROP TABLE IF EXISTS team_memberships;
DROP TABLE IF EXISTS teams;
-- +goose StatementEnd
