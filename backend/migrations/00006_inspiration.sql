-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;

ALTER TABLE goals
    ADD COLUMN IF NOT EXISTS is_public_template BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS is_hidden          BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS embedding          vector(1536);

ALTER TABLE check_ins
    ADD COLUMN IF NOT EXISTS is_public_example      BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS is_hidden              BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS public_attachment_ids  BIGINT[] NOT NULL DEFAULT '{}';

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS public_alias         TEXT,
    ADD COLUMN IF NOT EXISTS is_anonymous_public  BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS share_default        BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_goals_public_template
    ON goals (is_public_template, is_hidden)
    WHERE is_public_template = TRUE AND is_hidden = FALSE;

CREATE INDEX IF NOT EXISTS idx_checkins_public_example
    ON check_ins (is_public_example, is_hidden)
    WHERE is_public_example = TRUE AND is_hidden = FALSE;

-- ivfflat indexes created only once there is enough data; add them via separate migration in prod
-- CREATE INDEX goals_pub_embedding ON goals USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100)
--     WHERE is_public_template = TRUE AND is_hidden = FALSE;
-- CREATE INDEX checkins_pub_embedding ON check_ins USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100)
--     WHERE is_public_example = TRUE AND is_hidden = FALSE;

CREATE TABLE IF NOT EXISTS inspiration_reports (
    id          BIGSERIAL PRIMARY KEY,
    reporter_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_kind TEXT    NOT NULL CHECK (target_kind IN ('goal','checkin')),
    target_id   BIGINT  NOT NULL,
    reason      TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE IF EXISTS inspiration_reports;

ALTER TABLE users
    DROP COLUMN IF EXISTS share_default,
    DROP COLUMN IF EXISTS is_anonymous_public,
    DROP COLUMN IF EXISTS public_alias;

ALTER TABLE check_ins
    DROP COLUMN IF EXISTS public_attachment_ids,
    DROP COLUMN IF EXISTS is_hidden,
    DROP COLUMN IF EXISTS is_public_example;

ALTER TABLE goals
    DROP COLUMN IF EXISTS embedding,
    DROP COLUMN IF EXISTS is_hidden,
    DROP COLUMN IF EXISTS is_public_template;

DROP EXTENSION IF EXISTS vector;
