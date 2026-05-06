-- +goose Up

-- Add daily window settings to circles (Wave 1A)
ALTER TABLE circles
    ADD COLUMN IF NOT EXISTS daily_window_tz TEXT NOT NULL DEFAULT 'UTC',
    ADD COLUMN IF NOT EXISTS daily_cutoff TEXT NOT NULL DEFAULT '23:59';

-- Circle events for Пульс feed (Wave 1B)
CREATE TABLE IF NOT EXISTS circle_events (
    id BIGSERIAL PRIMARY KEY,
    circle_id BIGINT NOT NULL REFERENCES circles(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_circle_events_circle_created ON circle_events(circle_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_circle_events_kind ON circle_events(kind);

-- +goose Down
DROP INDEX IF EXISTS idx_circle_events_kind;
DROP INDEX IF EXISTS idx_circle_events_circle_created;
DROP TABLE IF EXISTS circle_events;

ALTER TABLE circles
    DROP COLUMN IF EXISTS daily_window_tz,
    DROP COLUMN IF EXISTS daily_cutoff;
