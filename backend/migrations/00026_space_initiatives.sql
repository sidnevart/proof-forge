-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS initiatives (
    id                  BIGSERIAL PRIMARY KEY,
    space_type          TEXT NOT NULL CHECK (space_type IN ('teamspace', 'community')),
    teamspace_id        BIGINT REFERENCES teams(id) ON DELETE CASCADE,
    community_space_id  BIGINT REFERENCES community_spaces(id) ON DELETE CASCADE,
    creator_id          BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title               TEXT NOT NULL CHECK (length(trim(title)) BETWEEN 3 AND 120),
    description         TEXT NOT NULL DEFAULT '',
    proof_criteria      TEXT NOT NULL CHECK (length(trim(proof_criteria)) >= 10),
    status              TEXT NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active', 'archived')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- XOR: initiative belongs to exactly one space type
    CONSTRAINT initiatives_space_xor CHECK (
        (teamspace_id IS NOT NULL AND community_space_id IS NULL) OR
        (teamspace_id IS NULL     AND community_space_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_initiatives_teamspace
    ON initiatives(teamspace_id) WHERE teamspace_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_initiatives_community
    ON initiatives(community_space_id) WHERE community_space_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_initiatives_creator
    ON initiatives(creator_id);

-- Add initiative_id to goals: NULL for personal goals, non-NULL for initiative goals.
ALTER TABLE goals
    ADD COLUMN IF NOT EXISTS initiative_id BIGINT REFERENCES initiatives(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_goals_initiative
    ON goals(initiative_id) WHERE initiative_id IS NOT NULL;

-- Initiative goals have no buddy and no circle: relax the NOT NULL constraints
-- that were added for personal goals. Service-layer validation still enforces
-- buddy/circle presence for personal goals.
ALTER TABLE goals ALTER COLUMN buddy_user_id DROP NOT NULL;
ALTER TABLE goals ALTER COLUMN circle_id DROP NOT NULL;

-- Unique partial index: one goal per user per initiative. Drives ON CONFLICT DO NOTHING
-- in the idempotent JoinOrGet query.
CREATE UNIQUE INDEX IF NOT EXISTS idx_goals_user_initiative
    ON goals(owner_user_id, initiative_id) WHERE initiative_id IS NOT NULL;

-- Peer approvals for initiative proofs (replaces the buddy model for initiatives).
CREATE TABLE IF NOT EXISTS initiative_approvals (
    id           BIGSERIAL PRIMARY KEY,
    checkin_id   BIGINT NOT NULL REFERENCES check_ins(id) ON DELETE CASCADE,
    approver_id  BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    comment      TEXT NOT NULL DEFAULT '',
    approved_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One approval per proof: first approver wins.
    CONSTRAINT initiative_approvals_one_per_checkin UNIQUE (checkin_id)
);

CREATE INDEX IF NOT EXISTS idx_initiative_approvals_approver
    ON initiative_approvals(approver_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_initiative_approvals_approver;
DROP TABLE IF EXISTS initiative_approvals;

DROP INDEX IF EXISTS idx_goals_user_initiative;
ALTER TABLE goals ALTER COLUMN circle_id SET NOT NULL;
ALTER TABLE goals ALTER COLUMN buddy_user_id SET NOT NULL;
DROP INDEX IF EXISTS idx_goals_initiative;
ALTER TABLE goals DROP COLUMN IF EXISTS initiative_id;

DROP INDEX IF EXISTS idx_initiatives_creator;
DROP INDEX IF EXISTS idx_initiatives_community;
DROP INDEX IF EXISTS idx_initiatives_teamspace;
DROP TABLE IF EXISTS initiatives;

-- +goose StatementEnd
