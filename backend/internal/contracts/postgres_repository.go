package contracts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository constructs a PostgresRepository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const contractCols = `id, goal_id, user_id, buddy_user_id, what_to_prove, how_to_prove,
	due_at, status, fulfilled_at, broken_at, cancelled_at, created_at, updated_at`

func scanContract(row pgx.Row) (*ProofContract, error) {
	var c ProofContract
	err := row.Scan(
		&c.ID, &c.GoalID, &c.UserID, &c.BuddyUserID,
		&c.WhatToProve, &c.HowToProve,
		&c.DueAt, &c.Status,
		&c.FulfilledAt, &c.BrokenAt, &c.CancelledAt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrContractNotFound
		}
		return nil, fmt.Errorf("scan contract: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepository) Create(ctx context.Context, c *ProofContract) error {
	const q = `
		INSERT INTO proof_contracts (goal_id, user_id, buddy_user_id, what_to_prove, how_to_prove, due_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ` + contractCols

	row := r.pool.QueryRow(ctx, q,
		c.GoalID, c.UserID, c.BuddyUserID,
		c.WhatToProve, c.HowToProve,
		c.DueAt, c.Status,
	)
	got, err := scanContract(row)
	if err != nil {
		return err
	}
	*c = *got
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id int64) (*ProofContract, error) {
	const q = `SELECT ` + contractCols + ` FROM proof_contracts WHERE id = $1`
	return scanContract(r.pool.QueryRow(ctx, q, id))
}

func (r *PostgresRepository) ListByGoal(ctx context.Context, goalID int64) ([]*ProofContract, error) {
	const q = `SELECT ` + contractCols + ` FROM proof_contracts WHERE goal_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, goalID)
	if err != nil {
		return nil, fmt.Errorf("list contracts: %w", err)
	}
	defer rows.Close()
	return collectContracts(rows)
}

func (r *PostgresRepository) ListByUser(ctx context.Context, userID int64, status *Status) ([]*ProofContract, error) {
	var rows pgx.Rows
	var err error

	if status != nil {
		rows, err = r.pool.Query(ctx,
			`SELECT `+contractCols+` FROM proof_contracts WHERE user_id = $1 AND status = $2 ORDER BY due_at ASC`,
			userID, *status,
		)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT `+contractCols+` FROM proof_contracts WHERE user_id = $1 ORDER BY due_at ASC`,
			userID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("list by user: %w", err)
	}
	defer rows.Close()
	return collectContracts(rows)
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, id int64, status Status) error {
	var tsCol string
	switch status {
	case StatusFulfilled:
		tsCol = ", fulfilled_at = NOW()"
	case StatusBroken:
		tsCol = ", broken_at = NOW()"
	case StatusCancelled:
		tsCol = ", cancelled_at = NOW()"
	}
	q := `UPDATE proof_contracts SET status = $2, updated_at = NOW()` + tsCol + ` WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, id, status)
	return err
}

func (r *PostgresRepository) MarkBrokenOverdue(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE proof_contracts
		SET status = 'broken', broken_at = NOW(), updated_at = NOW()
		WHERE status IN ('pending', 'active') AND due_at < $1`,
		time.Now(),
	)
	if err != nil {
		return 0, fmt.Errorf("mark broken overdue: %w", err)
	}
	return tag.RowsAffected(), nil
}

func collectContracts(rows pgx.Rows) ([]*ProofContract, error) {
	var out []*ProofContract
	for rows.Next() {
		var c ProofContract
		if err := rows.Scan(
			&c.ID, &c.GoalID, &c.UserID, &c.BuddyUserID,
			&c.WhatToProve, &c.HowToProve,
			&c.DueAt, &c.Status,
			&c.FulfilledAt, &c.BrokenAt, &c.CancelledAt,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan contract row: %w", err)
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}
