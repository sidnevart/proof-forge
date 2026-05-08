-- +goose Up
-- +goose StatementBegin
ALTER TABLE analytics_events
    ADD COLUMN IF NOT EXISTS workspace_id BIGINT;

CREATE INDEX IF NOT EXISTS idx_analytics_workspace_event_ts
    ON analytics_events(workspace_id, event_name, ts DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_analytics_workspace_event_ts;
ALTER TABLE analytics_events DROP COLUMN IF EXISTS workspace_id;
-- +goose StatementEnd
