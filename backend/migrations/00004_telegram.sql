-- +goose Up

CREATE TABLE IF NOT EXISTS telegram_links (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    telegram_chat_id BIGINT NOT NULL UNIQUE,
    telegram_username TEXT,
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS telegram_link_pending (
    token TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS notifications_log (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    dedup_key TEXT UNIQUE,
    payload JSONB NOT NULL DEFAULT '{}',
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL DEFAULT 'sent'
);

CREATE TABLE IF NOT EXISTS domain_events (
    id BIGSERIAL PRIMARY KEY,
    kind TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telegram_links_chat_id ON telegram_links(telegram_chat_id);
CREATE INDEX IF NOT EXISTS idx_notifications_log_user_day ON notifications_log(user_id, sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_domain_events_unprocessed ON domain_events(created_at)
    WHERE processed_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_domain_events_unprocessed;
DROP INDEX IF EXISTS idx_notifications_log_user_day;
DROP INDEX IF EXISTS idx_telegram_links_chat_id;
DROP TABLE IF EXISTS domain_events;
DROP TABLE IF EXISTS notifications_log;
DROP TABLE IF EXISTS telegram_link_pending;
DROP TABLE IF EXISTS telegram_links;
