-- +goose Up
CREATE TABLE user_achievements (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT NOT NULL REFERENCES users(id),
    achievement_id TEXT NOT NULL,
    unlocked_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged   BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE(user_id, achievement_id)
);

CREATE INDEX user_achievements_user_idx ON user_achievements(user_id, unlocked_at DESC);

-- +goose Down
DROP TABLE IF EXISTS user_achievements;
