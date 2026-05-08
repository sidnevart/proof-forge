package community

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using pgxpool.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository constructs a PostgresRepository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const csCols = `id, workspace_id, owner_user_id, name, slug, description, invite_code, is_public, created_at, updated_at`

func scanCS(row pgx.Row) (*CommunitySpace, error) {
	var cs CommunitySpace
	err := row.Scan(
		&cs.ID, &cs.WorkspaceID, &cs.OwnerUserID,
		&cs.Name, &cs.Slug, &cs.Description,
		&cs.InviteCode, &cs.IsPublic,
		&cs.CreatedAt, &cs.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCommunityNotFound
		}
		return nil, fmt.Errorf("scan community space: %w", err)
	}
	return &cs, nil
}

func (r *PostgresRepository) Create(ctx context.Context, cs *CommunitySpace) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create community space: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO community_spaces (workspace_id, owner_user_id, name, slug, description, invite_code, is_public)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ` + csCols

	row := tx.QueryRow(ctx, q,
		cs.WorkspaceID, cs.OwnerUserID, cs.Name, cs.Slug,
		cs.Description, cs.InviteCode, cs.IsPublic,
	)
	got, err := scanCS(row)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO community_memberships (community_space_id, user_id, role, status)
		VALUES ($1, $2, $3, $4)
	`, got.ID, got.OwnerUserID, RoleCommunityLeader, MemberStatusActive); err != nil {
		return fmt.Errorf("create owner membership: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create community space: %w", err)
	}
	*cs = *got
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id int64) (*CommunitySpace, error) {
	const q = `SELECT ` + csCols + ` FROM community_spaces WHERE id = $1`
	return scanCS(r.pool.QueryRow(ctx, q, id))
}

func (r *PostgresRepository) GetByInviteCode(ctx context.Context, code string) (*CommunitySpace, error) {
	const q = `SELECT ` + csCols + ` FROM community_spaces WHERE invite_code = $1`
	return scanCS(r.pool.QueryRow(ctx, q, code))
}

func (r *PostgresRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM community_spaces WHERE slug = $1)`, slug).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) ListByWorkspace(ctx context.Context, workspaceID int64) ([]*CommunitySpace, error) {
	const q = `SELECT ` + csCols + ` FROM community_spaces WHERE workspace_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list by workspace: %w", err)
	}
	defer rows.Close()

	var out []*CommunitySpace
	for rows.Next() {
		var cs CommunitySpace
		if err := rows.Scan(
			&cs.ID, &cs.WorkspaceID, &cs.OwnerUserID,
			&cs.Name, &cs.Slug, &cs.Description,
			&cs.InviteCode, &cs.IsPublic,
			&cs.CreatedAt, &cs.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		out = append(out, &cs)
	}
	return out, rows.Err()
}

const memberCols = `id, community_space_id, user_id, role, status, joined_at, left_at`

func scanMember(row pgx.Row) (*Membership, error) {
	var m Membership
	err := row.Scan(&m.ID, &m.CommunitySpaceID, &m.UserID, &m.Role, &m.Status, &m.JoinedAt, &m.LeftAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotMember
		}
		return nil, fmt.Errorf("scan membership: %w", err)
	}
	return &m, nil
}

func (r *PostgresRepository) AddMember(ctx context.Context, m *Membership) error {
	const q = `
		INSERT INTO community_memberships (community_space_id, user_id, role, status)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + memberCols

	row := r.pool.QueryRow(ctx, q, m.CommunitySpaceID, m.UserID, m.Role, m.Status)
	got, err := scanMember(row)
	if err != nil {
		return err
	}
	*m = *got
	return nil
}

func (r *PostgresRepository) GetMembership(ctx context.Context, communityID, userID int64) (*Membership, error) {
	const q = `SELECT ` + memberCols + ` FROM community_memberships WHERE community_space_id = $1 AND user_id = $2`
	return scanMember(r.pool.QueryRow(ctx, q, communityID, userID))
}

func (r *PostgresRepository) UpdateMemberStatus(ctx context.Context, communityID, userID int64, status MemberStatus) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE community_memberships SET status = $3 WHERE community_space_id = $1 AND user_id = $2`,
		communityID, userID, status,
	)
	return err
}

func (r *PostgresRepository) CountActiveMembers(ctx context.Context, communityID int64) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM community_memberships WHERE community_space_id = $1 AND status = 'active'`,
		communityID,
	).Scan(&n)
	return n, err
}
