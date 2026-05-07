-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS daily_log_entries (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    log_date DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'logged'
        CHECK (status IN ('logged', 'skipped', 'frozen', 'missed')),
    text_content TEXT NOT NULL DEFAULT '',
    has_artifact BOOLEAN NOT NULL DEFAULT FALSE,
    external_url TEXT,
    storage_key TEXT,
    mime_type TEXT,
    file_size_bytes BIGINT,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    overwritten_at TIMESTAMPTZ,
    consumed_in_check_in_id BIGINT REFERENCES check_ins(id) ON DELETE SET NULL,
    consumed_at TIMESTAMPTZ,
    UNIQUE (user_id, team_id, log_date)
);

CREATE INDEX IF NOT EXISTS idx_daily_log_team_user_date
    ON daily_log_entries(team_id, user_id, log_date DESC);

CREATE INDEX IF NOT EXISTS idx_daily_log_unconsumed_week
    ON daily_log_entries(team_id, user_id, log_date)
    WHERE consumed_in_check_in_id IS NULL AND status = 'logged';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_daily_log_unconsumed_week;
DROP INDEX IF EXISTS idx_daily_log_team_user_date;
DROP TABLE IF EXISTS daily_log_entries;
-- +goose StatementEnd
