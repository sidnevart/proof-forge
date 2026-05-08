-- +goose Up
CREATE TABLE proof_likes (
    id          BIGSERIAL PRIMARY KEY,
    check_in_id BIGINT NOT NULL REFERENCES check_ins(id) ON DELETE CASCADE,
    user_id     BIGINT NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(check_in_id, user_id)
);

CREATE INDEX proof_likes_checkin_idx ON proof_likes(check_in_id);

-- +goose Down
DROP TABLE IF EXISTS proof_likes;
