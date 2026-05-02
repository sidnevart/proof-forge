package milestones

import (
	"context"
	"database/sql"
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

func (r *PostgresRepository) Insert(ctx context.Context, goalID int64, title, description string, sortOrder int) (Milestone, error) {
	const query = `
		INSERT INTO milestones (goal_id, title, description, status, sort_order)
		VALUES ($1, $2, $3, 'pending', $4)
		RETURNING id, goal_id, title, description, status, sort_order,
		          completed_at, completed_by_user_id, created_at, updated_at
	`
	return scanRow(r.pool.QueryRow(ctx, query, goalID, title, description, sortOrder))
}

func (r *PostgresRepository) FindByGoal(ctx context.Context, goalID int64) ([]Milestone, error) {
	const query = `
		SELECT id, goal_id, title, description, status, sort_order,
		       completed_at, completed_by_user_id, created_at, updated_at
		FROM milestones
		WHERE goal_id = $1
		ORDER BY status DESC, sort_order ASC, id ASC
	`

	rows, err := r.pool.Query(ctx, query, goalID)
	if err != nil {
		return nil, fmt.Errorf("query milestones: %w", err)
	}
	defer rows.Close()

	out := make([]Milestone, 0)
	for rows.Next() {
		m, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan milestone: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate milestones: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, milestoneID int64) (Milestone, error) {
	const query = `
		SELECT id, goal_id, title, description, status, sort_order,
		       completed_at, completed_by_user_id, created_at, updated_at
		FROM milestones WHERE id = $1
	`
	m, err := scanRow(r.pool.QueryRow(ctx, query, milestoneID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Milestone{}, ErrMilestoneNotFound
		}
		return Milestone{}, fmt.Errorf("find milestone: %w", err)
	}
	return m, nil
}

func (r *PostgresRepository) Update(ctx context.Context, milestoneID int64, title, description *string, sortOrder *int) (Milestone, error) {
	const query = `
		UPDATE milestones SET
			title       = COALESCE($2, title),
			description = COALESCE($3, description),
			sort_order  = COALESCE($4, sort_order),
			updated_at  = NOW()
		WHERE id = $1
		RETURNING id, goal_id, title, description, status, sort_order,
		          completed_at, completed_by_user_id, created_at, updated_at
	`
	m, err := scanRow(r.pool.QueryRow(ctx, query, milestoneID, title, description, sortOrder))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Milestone{}, ErrMilestoneNotFound
		}
		return Milestone{}, fmt.Errorf("update milestone: %w", err)
	}
	return m, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, milestoneID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM milestones WHERE id = $1`, milestoneID)
	if err != nil {
		return fmt.Errorf("delete milestone: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMilestoneNotFound
	}
	return nil
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, milestoneID int64, status MilestoneStatus, completedByUserID *int64, now time.Time) (Milestone, error) {
	var query string
	if status == StatusCompleted {
		query = `
			UPDATE milestones SET
				status = $2,
				completed_at = $3,
				completed_by_user_id = $4,
				updated_at = $3
			WHERE id = $1
			RETURNING id, goal_id, title, description, status, sort_order,
			          completed_at, completed_by_user_id, created_at, updated_at
		`
	} else {
		query = `
			UPDATE milestones SET
				status = $2,
				completed_at = NULL,
				completed_by_user_id = NULL,
				updated_at = $3
			WHERE id = $1
			RETURNING id, goal_id, title, description, status, sort_order,
			          completed_at, completed_by_user_id, created_at, updated_at
		`
	}

	var row pgx.Row
	if status == StatusCompleted {
		row = r.pool.QueryRow(ctx, query, milestoneID, status, now, completedByUserID)
	} else {
		row = r.pool.QueryRow(ctx, query, milestoneID, status, now)
	}

	m, err := scanRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Milestone{}, ErrMilestoneNotFound
		}
		return Milestone{}, fmt.Errorf("update milestone status: %w", err)
	}
	return m, nil
}

func (r *PostgresRepository) FindGoalParticipants(ctx context.Context, goalID int64) (int64, int64, string, error) {
	const query = `SELECT owner_user_id, buddy_user_id, status FROM goals WHERE id = $1`
	var ownerID, buddyID int64
	var status string
	err := r.pool.QueryRow(ctx, query, goalID).Scan(&ownerID, &buddyID, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, "", ErrMilestoneNotFound
		}
		return 0, 0, "", fmt.Errorf("find goal participants: %w", err)
	}
	return ownerID, buddyID, status, nil
}

func (r *PostgresRepository) NextSortOrder(ctx context.Context, goalID int64) (int, error) {
	const query = `SELECT COALESCE(MAX(sort_order), -1) + 1 FROM milestones WHERE goal_id = $1`
	var next int
	if err := r.pool.QueryRow(ctx, query, goalID).Scan(&next); err != nil {
		return 0, fmt.Errorf("next sort order: %w", err)
	}
	return next, nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanRow(row rowScanner) (Milestone, error) {
	var m Milestone
	var completedAt sql.NullTime
	var completedBy sql.NullInt64
	if err := row.Scan(
		&m.ID, &m.GoalID, &m.Title, &m.Description, &m.Status, &m.SortOrder,
		&completedAt, &completedBy, &m.CreatedAt, &m.UpdatedAt,
	); err != nil {
		return Milestone{}, err
	}
	if completedAt.Valid {
		t := completedAt.Time
		m.CompletedAt = &t
	}
	if completedBy.Valid {
		v := completedBy.Int64
		m.CompletedByUserID = &v
	}
	return m, nil
}
