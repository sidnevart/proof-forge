-- +goose Up
CREATE TABLE milestones (
    id                   BIGSERIAL PRIMARY KEY,
    goal_id              BIGINT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    title                TEXT NOT NULL,
    description          TEXT NOT NULL DEFAULT '',
    status               TEXT NOT NULL DEFAULT 'pending',
    sort_order           INTEGER NOT NULL DEFAULT 0,
    completed_at         TIMESTAMPTZ,
    completed_by_user_id BIGINT REFERENCES users(id),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_milestones_goal_id ON milestones(goal_id, sort_order);

-- +goose Down
DROP TABLE IF EXISTS milestones;
