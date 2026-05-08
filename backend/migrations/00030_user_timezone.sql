-- +goose Up

-- Add timezone column to users table for local-time AI triggers.
ALTER TABLE users ADD COLUMN IF NOT EXISTS timezone TEXT DEFAULT 'UTC';

-- +goose Down

ALTER TABLE users DROP COLUMN IF EXISTS timezone;
