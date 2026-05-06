-- +goose Up
CREATE TABLE IF NOT EXISTS circles (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    invite_code TEXT NOT NULL UNIQUE,
    member_limit INTEGER NOT NULL DEFAULT 8,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS circle_memberships (
    id BIGSERIAL PRIMARY KEY,
    circle_id BIGINT NOT NULL REFERENCES circles(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'active',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (circle_id, user_id)
);

CREATE TABLE IF NOT EXISTS circle_seasons (
    id BIGSERIAL PRIMARY KEY,
    circle_id BIGINT NOT NULL REFERENCES circles(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE goals
    ADD COLUMN IF NOT EXISTS circle_id BIGINT REFERENCES circles(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_circles_owner_created ON circles(owner_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_circle_memberships_user_circle ON circle_memberships(user_id, circle_id);
CREATE INDEX IF NOT EXISTS idx_circle_memberships_circle_status ON circle_memberships(circle_id, status);
CREATE INDEX IF NOT EXISTS idx_circle_seasons_circle_status ON circle_seasons(circle_id, status);
CREATE INDEX IF NOT EXISTS idx_goals_circle_owner ON goals(circle_id, owner_user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_goals_circle_owner;
DROP INDEX IF EXISTS idx_circle_seasons_circle_status;
DROP INDEX IF EXISTS idx_circle_memberships_circle_status;
DROP INDEX IF EXISTS idx_circle_memberships_user_circle;
DROP INDEX IF EXISTS idx_circles_owner_created;

ALTER TABLE goals DROP COLUMN IF EXISTS circle_id;
DROP TABLE IF EXISTS circle_seasons;
DROP TABLE IF EXISTS circle_memberships;
DROP TABLE IF EXISTS circles;
