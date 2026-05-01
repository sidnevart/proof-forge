-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    avatar_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS goals (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES users(id),
    buddy_user_id BIGINT NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    current_progress_health TEXT NOT NULL DEFAULT 'unknown',
    current_streak_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS pacts (
    id BIGSERIAL PRIMARY KEY,
    goal_id BIGINT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    owner_user_id BIGINT NOT NULL REFERENCES users(id),
    buddy_user_id BIGINT NOT NULL REFERENCES users(id),
    status TEXT NOT NULL,
    accepted_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS invites (
    id BIGSERIAL PRIMARY KEY,
    goal_id BIGINT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    pact_id BIGINT NOT NULL REFERENCES pacts(id) ON DELETE CASCADE,
    inviter_user_id BIGINT NOT NULL REFERENCES users(id),
    invitee_user_id BIGINT NOT NULL REFERENCES users(id),
    token_hash TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS check_ins (
    id BIGSERIAL PRIMARY KEY,
    goal_id BIGINT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    owner_user_id BIGINT NOT NULL REFERENCES users(id),
    status TEXT NOT NULL,
    submitted_at TIMESTAMPTZ,
    approved_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ,
    changes_requested_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS evidence_items (
    id BIGSERIAL PRIMARY KEY,
    check_in_id BIGINT NOT NULL REFERENCES check_ins(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    text_content TEXT,
    external_url TEXT,
    storage_key TEXT,
    mime_type TEXT,
    file_size_bytes BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS check_in_reviews (
    id BIGSERIAL PRIMARY KEY,
    check_in_id BIGINT NOT NULL REFERENCES check_ins(id) ON DELETE CASCADE,
    reviewer_user_id BIGINT NOT NULL REFERENCES users(id),
    decision TEXT NOT NULL,
    comment TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS weekly_recaps (
    id BIGSERIAL PRIMARY KEY,
    goal_id BIGINT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    owner_user_id BIGINT NOT NULL REFERENCES users(id),
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL,
    summary_text TEXT NOT NULL DEFAULT '',
    model_name TEXT,
    generated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_goals_owner_status ON goals(owner_user_id, status);
CREATE INDEX IF NOT EXISTS idx_goals_buddy_status ON goals(buddy_user_id, status);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_expires ON user_sessions(user_id, expires_at DESC);
CREATE INDEX IF NOT EXISTS idx_pacts_goal_id ON pacts(goal_id);
CREATE INDEX IF NOT EXISTS idx_pacts_owner_buddy_status ON pacts(owner_user_id, buddy_user_id, status);
CREATE INDEX IF NOT EXISTS idx_invites_invitee_status ON invites(invitee_user_id, status);
CREATE INDEX IF NOT EXISTS idx_invites_goal_status ON invites(goal_id, status);
CREATE INDEX IF NOT EXISTS idx_check_ins_goal_created_at ON check_ins(goal_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_check_ins_owner_status ON check_ins(owner_user_id, status);
CREATE INDEX IF NOT EXISTS idx_check_ins_goal_status ON check_ins(goal_id, status);
CREATE INDEX IF NOT EXISTS idx_evidence_items_check_in_id ON evidence_items(check_in_id);
CREATE INDEX IF NOT EXISTS idx_check_in_reviews_check_in_created_at ON check_in_reviews(check_in_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_weekly_recaps_goal_period ON weekly_recaps(goal_id, period_start, period_end);
CREATE INDEX IF NOT EXISTS idx_weekly_recaps_owner_period ON weekly_recaps(owner_user_id, period_start DESC);

-- +goose Down
DROP TABLE IF EXISTS weekly_recaps;
DROP TABLE IF EXISTS check_in_reviews;
DROP TABLE IF EXISTS evidence_items;
DROP TABLE IF EXISTS check_ins;
DROP TABLE IF EXISTS invites;
DROP TABLE IF EXISTS pacts;
DROP TABLE IF EXISTS goals;
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS users;
