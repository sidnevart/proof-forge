-- +goose Up
CREATE TABLE stakes (
    id            BIGSERIAL PRIMARY KEY,
    goal_id       BIGINT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    owner_user_id BIGINT NOT NULL REFERENCES users(id),
    description   TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'active',
    forfeited_at  TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    cancelled_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stakes_goal_id ON stakes(goal_id);
CREATE INDEX idx_stakes_owner_status ON stakes(owner_user_id, status);

CREATE TABLE stake_forfeitures (
    id                  BIGSERIAL PRIMARY KEY,
    stake_id            BIGINT NOT NULL REFERENCES stakes(id) ON DELETE CASCADE,
    declared_by_user_id BIGINT NOT NULL REFERENCES users(id),
    reason              TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stake_forfeitures_stake_id ON stake_forfeitures(stake_id);

-- +goose Down
DROP TABLE IF EXISTS stake_forfeitures;
DROP TABLE IF EXISTS stakes;
