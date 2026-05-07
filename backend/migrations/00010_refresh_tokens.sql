-- 00010_refresh_tokens.sql — refresh-token rotation chain.
--
-- The previous auth model issued a single 30-day opaque session and never
-- extended it: an active user got hard-evicted on day 30, and a transient
-- network blip in the dashboard fetch was enough to make the SPA *think* the
-- user was logged out. We now move to access + refresh:
--
--   * access  — short-lived (~15m), still stored in `user_sessions` (the same
--                token-hash table). The middleware keeps validating it
--                untouched.
--   * refresh — long-lived (~30d), stored here. Each /v1/auth/refresh call
--                rotates the token: old row marked revoked, new row chained
--                via parent_id. Re-using a revoked-but-rotated row is a
--                stolen-token signal — we revoke the entire chain.
--
-- This change adds the new table only; `user_sessions` is left untouched so
-- already-logged-in users keep working until their cookie expires.

-- +goose Up
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    -- Pointer to the previous row in the rotation chain. When a refresh
    -- request rotates a token, we set parent_id on the new row so a reuse
    -- detection sweep can walk the chain.
    parent_id   BIGINT REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_idx ON refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS refresh_tokens_parent_id_idx ON refresh_tokens (parent_id);

-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;
