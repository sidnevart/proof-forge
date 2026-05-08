-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS proof_contracts (
    id             BIGSERIAL PRIMARY KEY,
    goal_id        BIGINT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    user_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    buddy_user_id  BIGINT REFERENCES users(id) ON DELETE SET NULL,
    what_to_prove  TEXT NOT NULL CHECK (length(trim(what_to_prove)) > 0),
    how_to_prove   TEXT NOT NULL DEFAULT '',
    due_at         TIMESTAMPTZ NOT NULL,
    status         TEXT NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'active', 'fulfilled', 'broken', 'cancelled')),
    fulfilled_at   TIMESTAMPTZ,
    broken_at      TIMESTAMPTZ,
    cancelled_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_proof_contracts_goal ON proof_contracts(goal_id, status);
CREATE INDEX IF NOT EXISTS idx_proof_contracts_user_status ON proof_contracts(user_id, status);
CREATE INDEX IF NOT EXISTS idx_proof_contracts_due ON proof_contracts(due_at, status)
    WHERE status IN ('pending', 'active');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_proof_contracts_due;
DROP INDEX IF EXISTS idx_proof_contracts_user_status;
DROP INDEX IF EXISTS idx_proof_contracts_goal;
DROP TABLE IF EXISTS proof_contracts;

-- +goose StatementEnd
