-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS analytics_events (
    id BIGSERIAL,
    ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_name TEXT NOT NULL,
    user_id BIGINT,
    team_id BIGINT,
    goal_id BIGINT,
    proof_id BIGINT,
    source TEXT NOT NULL CHECK (source IN ('web', 'telegram_bot', 'cron', 'system')),
    properties JSONB NOT NULL DEFAULT '{}',
    PRIMARY KEY (id, ts)
) PARTITION BY RANGE (ts);

-- Create initial monthly partition.
CREATE TABLE IF NOT EXISTS analytics_events_2026_05
    PARTITION OF analytics_events
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE INDEX IF NOT EXISTS idx_analytics_team_event_ts
    ON analytics_events(team_id, event_name, ts DESC);

CREATE INDEX IF NOT EXISTS idx_analytics_event_ts
    ON analytics_events(event_name, ts DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_analytics_event_ts;
DROP INDEX IF EXISTS idx_analytics_team_event_ts;
DROP TABLE IF EXISTS analytics_events_2026_05;
DROP TABLE IF EXISTS analytics_events;
-- +goose StatementEnd
