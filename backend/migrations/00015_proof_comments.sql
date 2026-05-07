-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS proof_comments (
    id BIGSERIAL PRIMARY KEY,
    proof_id BIGINT NOT NULL REFERENCES check_ins(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text_content TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_proof_comments_proof
    ON proof_comments(proof_id, created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_proof_comments_proof;
DROP TABLE IF EXISTS proof_comments;
-- +goose StatementEnd
