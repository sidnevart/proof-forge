-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS community_spaces (
    id            BIGSERIAL PRIMARY KEY,
    workspace_id  BIGINT REFERENCES workspaces(id) ON DELETE SET NULL,
    owner_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name          TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 80),
    slug          TEXT NOT NULL CHECK (slug ~ '^[a-z0-9][a-z0-9\-]{1,48}[a-z0-9]$'),
    description   TEXT NOT NULL DEFAULT '',
    invite_code   TEXT NOT NULL UNIQUE,
    is_public     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT community_spaces_slug_unique UNIQUE (slug)
);

CREATE TABLE IF NOT EXISTS community_memberships (
    id                 BIGSERIAL PRIMARY KEY,
    community_space_id BIGINT NOT NULL REFERENCES community_spaces(id) ON DELETE CASCADE,
    user_id            BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role               TEXT NOT NULL DEFAULT 'member'
                           CHECK (role IN ('community_leader', 'member')),
    status             TEXT NOT NULL DEFAULT 'active'
                           CHECK (status IN ('active', 'left', 'removed')),
    joined_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at            TIMESTAMPTZ,

    CONSTRAINT community_memberships_unique UNIQUE (community_space_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_community_spaces_workspace ON community_spaces(workspace_id)
    WHERE workspace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_community_spaces_owner ON community_spaces(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_community_memberships_user ON community_memberships(user_id, status);
CREATE INDEX IF NOT EXISTS idx_community_memberships_space_role ON community_memberships(community_space_id, role)
    WHERE status = 'active';

-- One active leader per community space — matches the teams pattern.
CREATE UNIQUE INDEX IF NOT EXISTS uq_community_one_leader ON community_memberships(community_space_id)
    WHERE role = 'community_leader' AND status = 'active';

-- Link circles to teamspace (teams) or community space (XOR, both nullable).
-- Business rule: a circle belongs to at most one parent space.
ALTER TABLE circles
    ADD COLUMN IF NOT EXISTS team_id           BIGINT REFERENCES teams(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS community_space_id BIGINT REFERENCES community_spaces(id) ON DELETE SET NULL;

-- XOR constraint: circle cannot belong to both team and community space.
ALTER TABLE circles
    ADD CONSTRAINT circles_parent_xor
    CHECK (NOT (team_id IS NOT NULL AND community_space_id IS NOT NULL));

CREATE INDEX IF NOT EXISTS idx_circles_team ON circles(team_id)
    WHERE team_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_circles_community_space ON circles(community_space_id)
    WHERE community_space_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_circles_community_space;
DROP INDEX IF EXISTS idx_circles_team;
ALTER TABLE circles DROP CONSTRAINT IF EXISTS circles_parent_xor;
ALTER TABLE circles
    DROP COLUMN IF EXISTS community_space_id,
    DROP COLUMN IF EXISTS team_id;

DROP INDEX IF EXISTS uq_community_one_leader;
DROP INDEX IF EXISTS idx_community_memberships_space_role;
DROP INDEX IF EXISTS idx_community_memberships_user;
DROP INDEX IF EXISTS idx_community_spaces_owner;
DROP INDEX IF EXISTS idx_community_spaces_workspace;

DROP TABLE IF EXISTS community_memberships;
DROP TABLE IF EXISTS community_spaces;

-- +goose StatementEnd
