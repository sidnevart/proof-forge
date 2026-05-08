package workspaces

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const workspaceCols = `id, owner_user_id, name, slug, type, is_active, frozen_at, frozen_reason, created_at, updated_at`

func scanWorkspace(row pgx.Row) (*Workspace, error) {
	var w Workspace
	err := row.Scan(
		&w.ID, &w.OwnerUserID, &w.Name, &w.Slug, &w.Type,
		&w.IsActive, &w.FrozenAt, &w.FrozenReason,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, fmt.Errorf("scan workspace: %w", err)
	}
	return &w, nil
}

func (r *PostgresRepository) Create(ctx context.Context, w *Workspace) error {
	const q = `
		INSERT INTO workspaces (owner_user_id, name, slug, type)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + workspaceCols

	row := r.pool.QueryRow(ctx, q, w.OwnerUserID, w.Name, w.Slug, w.Type)
	got, err := scanWorkspace(row)
	if err != nil {
		return err
	}
	*w = *got
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id int64) (*Workspace, error) {
	const q = `SELECT ` + workspaceCols + ` FROM workspaces WHERE id = $1`
	return scanWorkspace(r.pool.QueryRow(ctx, q, id))
}

func (r *PostgresRepository) GetBySlug(ctx context.Context, slug string) (*Workspace, error) {
	const q = `SELECT ` + workspaceCols + ` FROM workspaces WHERE slug = $1`
	return scanWorkspace(r.pool.QueryRow(ctx, q, slug))
}

func (r *PostgresRepository) ListByOwner(ctx context.Context, ownerUserID int64) ([]*Workspace, error) {
	const q = `SELECT ` + workspaceCols + ` FROM workspaces WHERE owner_user_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()
	return collectWorkspaces(rows)
}

func (r *PostgresRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE slug = $1)`, slug).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) Freeze(ctx context.Context, id int64, reason string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE workspaces SET frozen_at = NOW(), frozen_reason = $2, is_active = FALSE, updated_at = NOW() WHERE id = $1`,
		id, reason,
	)
	return err
}

func (r *PostgresRepository) Unfreeze(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE workspaces SET frozen_at = NULL, frozen_reason = NULL, is_active = TRUE, updated_at = NOW() WHERE id = $1`,
		id,
	)
	return err
}

func (r *PostgresRepository) ListAll(ctx context.Context, limit, offset int) ([]*Workspace, error) {
	const q = `SELECT ` + workspaceCols + ` FROM workspaces ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list all workspaces: %w", err)
	}
	defer rows.Close()
	return collectWorkspaces(rows)
}

func (r *PostgresRepository) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM workspaces`).Scan(&n)
	return n, err
}

func collectWorkspaces(rows pgx.Rows) ([]*Workspace, error) {
	var out []*Workspace
	for rows.Next() {
		var w Workspace
		var frozenAt *time.Time
		var frozenReason *string
		if err := rows.Scan(
			&w.ID, &w.OwnerUserID, &w.Name, &w.Slug, &w.Type,
			&w.IsActive, &frozenAt, &frozenReason,
			&w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan workspace row: %w", err)
		}
		w.FrozenAt = frozenAt
		w.FrozenReason = frozenReason
		out = append(out, &w)
	}
	return out, rows.Err()
}
