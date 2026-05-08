-- +goose Up
ALTER TABLE circle_invitations
    ADD COLUMN IF NOT EXISTS message TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE circle_invitations
SET updated_at = COALESCE(responded_at, created_at, NOW())
WHERE updated_at IS NULL;

-- +goose Down
ALTER TABLE circle_invitations
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS message;
