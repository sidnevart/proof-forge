-- +goose Up
-- +goose StatementBegin

ALTER TABLE users
  ADD COLUMN is_platform_admin BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX users_platform_admin_idx ON users(id) WHERE is_platform_admin = TRUE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS users_platform_admin_idx;
ALTER TABLE users DROP COLUMN IF EXISTS is_platform_admin;

-- +goose StatementEnd
