-- +goose Up
-- +goose StatementBegin

ALTER TABLE goals
    ADD COLUMN IF NOT EXISTS team_id BIGINT REFERENCES teams(id) ON DELETE SET NULL;

ALTER TABLE goals
    ALTER COLUMN circle_id DROP NOT NULL;

-- XOR: цель либо в круге, либо в команде, либо ни там, ни там — но не в обоих.
ALTER TABLE goals
    DROP CONSTRAINT IF EXISTS goal_circle_or_team_xor;
ALTER TABLE goals
    ADD CONSTRAINT goal_circle_or_team_xor
    CHECK (circle_id IS NULL OR team_id IS NULL);

ALTER TABLE check_ins
    ADD COLUMN IF NOT EXISTS team_id BIGINT REFERENCES teams(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS approver_role TEXT;

ALTER TABLE check_ins
    DROP CONSTRAINT IF EXISTS check_ins_approver_role_valid;
ALTER TABLE check_ins
    ADD CONSTRAINT check_ins_approver_role_valid
    CHECK (approver_role IS NULL OR approver_role IN ('lead', 'trusted_approver'));

CREATE INDEX IF NOT EXISTS idx_goals_team_owner
    ON goals(team_id, owner_user_id) WHERE team_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_check_ins_team_status
    ON check_ins(team_id, status) WHERE team_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_check_ins_team_status;
DROP INDEX IF EXISTS idx_goals_team_owner;

ALTER TABLE check_ins DROP CONSTRAINT IF EXISTS check_ins_approver_role_valid;
ALTER TABLE check_ins DROP COLUMN IF EXISTS approver_role;
ALTER TABLE check_ins DROP COLUMN IF EXISTS team_id;

ALTER TABLE goals DROP CONSTRAINT IF EXISTS goal_circle_or_team_xor;
ALTER TABLE goals DROP COLUMN IF EXISTS team_id;
ALTER TABLE goals ALTER COLUMN circle_id SET NOT NULL;
-- +goose StatementEnd
