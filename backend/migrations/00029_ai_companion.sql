-- +goose Up

-- Track which triggers have already fired for idempotency
CREATE TABLE ai_companion_fired (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     BIGINT REFERENCES users(id) ON DELETE CASCADE,
    feature     TEXT NOT NULL,
    trigger_id  TEXT NOT NULL,
    fired_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, trigger_id)
);
CREATE INDEX idx_ai_companion_fired_user ON ai_companion_fired(user_id, fired_at DESC);
CREATE INDEX idx_ai_companion_fired_feature ON ai_companion_fired(feature, fired_at DESC);

-- AI-assembled proof drafts awaiting user action
CREATE TABLE ai_proof_drafts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_id     BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    goal_id     BIGINT REFERENCES goals(id) ON DELETE SET NULL,
    note_ids    BIGINT[] NOT NULL DEFAULT '{}',
    rationale   TEXT,
    confidence  TEXT CHECK (confidence IN ('low','medium','high')),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    consumed_at TIMESTAMPTZ
);
CREATE INDEX idx_ai_proof_drafts_user ON ai_proof_drafts(user_id, consumed_at NULLS FIRST);
CREATE INDEX idx_ai_proof_drafts_team ON ai_proof_drafts(team_id, created_at DESC);

-- Cache for AI companion outputs (briefs, recaps)
CREATE TABLE ai_companion_cache (
    cache_key   TEXT PRIMARY KEY,
    feature     TEXT NOT NULL,
    payload     JSONB NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_ai_companion_cache_expires ON ai_companion_cache(expires_at);

-- In-app notifications for AI insights
CREATE TABLE ai_notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    feature     TEXT NOT NULL,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL,
    actions     JSONB NOT NULL DEFAULT '[]',
    dismissed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_ai_notifications_user ON ai_notifications(user_id, created_at DESC);
CREATE INDEX idx_ai_notifications_active ON ai_notifications(user_id, dismissed_at NULLS FIRST);

-- +goose Down

DROP TABLE IF EXISTS ai_notifications;
DROP TABLE IF EXISTS ai_companion_cache;
DROP TABLE IF EXISTS ai_proof_drafts;
DROP TABLE IF EXISTS ai_companion_fired;
