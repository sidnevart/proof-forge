-- +goose Up

INSERT INTO community_memberships (community_space_id, user_id, role, status)
SELECT cs.id, cs.owner_user_id, 'community_leader', 'active'
FROM community_spaces cs
ON CONFLICT (community_space_id, user_id)
DO UPDATE SET role = 'community_leader', status = 'active', left_at = NULL;

-- +goose Down

-- Historical repair only. Down is intentionally empty to avoid removing real owner access.
