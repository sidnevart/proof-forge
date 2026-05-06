-- +goose Up

-- 1. Add role column to circle_memberships
ALTER TABLE circle_memberships
    ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'observer';

-- Backfill: owner of circle gets 'owner' role
UPDATE circle_memberships cm
SET role = 'owner'
FROM circles c
WHERE cm.circle_id = c.id AND cm.user_id = c.owner_user_id;

-- Backfill: existing buddies (active pacts) get 'buddy' role
UPDATE circle_memberships cm
SET role = 'buddy'
FROM goals g
JOIN pacts p ON p.goal_id = g.id
WHERE g.circle_id IS NOT NULL
  AND cm.circle_id = g.circle_id
  AND cm.user_id = g.buddy_user_id
  AND cm.role = 'observer';

-- 2. Backfill: create legacy circles for goals without circle_id, attach them.
--    Each orphan owner gets one legacy circle that holds all their orphan goals.
WITH orphan_owners AS (
    SELECT DISTINCT owner_user_id FROM goals WHERE circle_id IS NULL
),
new_circles AS (
    INSERT INTO circles (owner_user_id, name, invite_code, member_limit, created_at, updated_at)
    SELECT
        o.owner_user_id,
        'Личный архив',
        'legacy-' || o.owner_user_id::text || '-' || extract(epoch from NOW())::bigint::text,
        1,
        NOW(), NOW()
    FROM orphan_owners o
    RETURNING id, owner_user_id
),
new_seasons AS (
    INSERT INTO circle_seasons (circle_id, status, starts_at, ends_at)
    SELECT nc.id, 'completed', NOW() - INTERVAL '7 days', NOW()
    FROM new_circles nc
    RETURNING id, circle_id
),
new_memberships AS (
    INSERT INTO circle_memberships (circle_id, user_id, status, role, joined_at)
    SELECT nc.id, nc.owner_user_id, 'active', 'owner', NOW()
    FROM new_circles nc
    RETURNING id
)
UPDATE goals g
SET circle_id = nc.id
FROM new_circles nc
WHERE g.owner_user_id = nc.owner_user_id AND g.circle_id IS NULL;

-- 3. Make goals.circle_id NOT NULL now that all rows are backfilled
ALTER TABLE goals ALTER COLUMN circle_id SET NOT NULL;

-- 4. Enforce single active goal per (circle, owner) — this is the inviarant
--    «1 круг = 1 цель на сезон». Owner can have only one active or pending goal.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_circle_owner_active_goal
    ON goals(circle_id, owner_user_id)
    WHERE status IN ('pending_buddy_acceptance', 'active');

-- 5. Switch season length to 7 days. For existing active seasons with ends_at
--    > starts_at + 7 days — clamp them to the new model. Seasons that are
--    already past 7 days from start get marked as completed.
UPDATE circle_seasons
SET status = 'completed'
WHERE status = 'active' AND starts_at + INTERVAL '7 days' < NOW();

UPDATE circle_seasons
SET ends_at = starts_at + INTERVAL '7 days'
WHERE status = 'active' AND ends_at > starts_at + INTERVAL '7 days';

-- 6. circle_invitations table (used in Slice 3, schema added now for atomic DDL)
CREATE TABLE IF NOT EXISTS circle_invitations (
    id             BIGSERIAL PRIMARY KEY,
    circle_id      BIGINT NOT NULL REFERENCES circles(id) ON DELETE CASCADE,
    inviter_user_id BIGINT NOT NULL REFERENCES users(id),
    target_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    target_email   TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'pending',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    responded_at   TIMESTAMPTZ,
    UNIQUE (circle_id, target_email)
);

CREATE INDEX IF NOT EXISTS idx_circle_invitations_target_email
    ON circle_invitations(target_email, status);
CREATE INDEX IF NOT EXISTS idx_circle_invitations_circle
    ON circle_invitations(circle_id, status);

-- 7. Season end-of-life metadata (Slice 4)
ALTER TABLE circle_seasons
    ADD COLUMN IF NOT EXISTS ended_action TEXT,
    ADD COLUMN IF NOT EXISTS successor_circle_id BIGINT REFERENCES circles(id);

-- +goose Down
ALTER TABLE circle_seasons DROP COLUMN IF EXISTS successor_circle_id;
ALTER TABLE circle_seasons DROP COLUMN IF EXISTS ended_action;
DROP INDEX IF EXISTS idx_circle_invitations_circle;
DROP INDEX IF EXISTS idx_circle_invitations_target_email;
DROP TABLE IF EXISTS circle_invitations;
DROP INDEX IF EXISTS uniq_circle_owner_active_goal;
ALTER TABLE goals ALTER COLUMN circle_id DROP NOT NULL;
ALTER TABLE circle_memberships DROP COLUMN IF EXISTS role;
