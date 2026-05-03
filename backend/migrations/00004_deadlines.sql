-- +goose Up
ALTER TABLE goals     ADD COLUMN deadline_at DATE;
ALTER TABLE check_ins ADD COLUMN deadline_at DATE;

-- +goose Down
ALTER TABLE check_ins DROP COLUMN deadline_at;
ALTER TABLE goals     DROP COLUMN deadline_at;
