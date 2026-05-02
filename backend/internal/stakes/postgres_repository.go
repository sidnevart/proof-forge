package stakes

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

func (r *PostgresRepository) Insert(ctx context.Context, goalID, ownerUserID int64, description string) (Stake, error) {
	const query = `
		INSERT INTO stakes (goal_id, owner_user_id, description, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id, goal_id, owner_user_id, description, status,
		          forfeited_at, completed_at, cancelled_at,
		          created_at, updated_at
	`

	var s Stake
	var forfeitedAt, completedAt, cancelledAt sql.NullTime
	err := r.pool.QueryRow(ctx, query, goalID, ownerUserID, description).Scan(
		&s.ID, &s.GoalID, &s.OwnerUserID, &s.Description, &s.Status,
		&forfeitedAt, &completedAt, &cancelledAt,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return Stake{}, fmt.Errorf("insert stake: %w", err)
	}
	applyNullTimes(&s, forfeitedAt, completedAt, cancelledAt)
	return s, nil
}

func (r *PostgresRepository) FindByGoal(ctx context.Context, goalID int64) ([]StakeView, error) {
	const query = `
		SELECT s.id, s.goal_id, s.owner_user_id, s.description, s.status,
		       s.forfeited_at, s.completed_at, s.cancelled_at,
		       s.created_at, s.updated_at,
		       f.id, f.declared_by_user_id, f.reason, f.created_at
		FROM stakes s
		LEFT JOIN LATERAL (
			SELECT sf.id, sf.declared_by_user_id, sf.reason, sf.created_at
			FROM stake_forfeitures sf
			WHERE sf.stake_id = s.id
			ORDER BY sf.created_at DESC
			LIMIT 1
		) f ON true
		WHERE s.goal_id = $1
		ORDER BY s.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, goalID)
	if err != nil {
		return nil, fmt.Errorf("query stakes: %w", err)
	}
	defer rows.Close()

	var views []StakeView
	for rows.Next() {
		var (
			s                                  Stake
			forfeitedAt, completedAt, cancelledAt sql.NullTime
			fID                                sql.NullInt64
			fDeclaredBy                        sql.NullInt64
			fReason                            sql.NullString
			fCreatedAt                         sql.NullTime
		)
		if err := rows.Scan(
			&s.ID, &s.GoalID, &s.OwnerUserID, &s.Description, &s.Status,
			&forfeitedAt, &completedAt, &cancelledAt,
			&s.CreatedAt, &s.UpdatedAt,
			&fID, &fDeclaredBy, &fReason, &fCreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stake row: %w", err)
		}
		applyNullTimes(&s, forfeitedAt, completedAt, cancelledAt)

		view := StakeView{Stake: s}
		if fID.Valid {
			view.Forfeiture = &StakeForfeiture{
				ID:               fID.Int64,
				StakeID:          s.ID,
				DeclaredByUserID: fDeclaredBy.Int64,
				Reason:           fReason.String,
				CreatedAt:        fCreatedAt.Time,
			}
		}
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stake rows: %w", err)
	}
	return views, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, stakeID int64) (Stake, error) {
	const query = `
		SELECT id, goal_id, owner_user_id, description, status,
		       forfeited_at, completed_at, cancelled_at,
		       created_at, updated_at
		FROM stakes
		WHERE id = $1
	`

	var s Stake
	var forfeitedAt, completedAt, cancelledAt sql.NullTime
	err := r.pool.QueryRow(ctx, query, stakeID).Scan(
		&s.ID, &s.GoalID, &s.OwnerUserID, &s.Description, &s.Status,
		&forfeitedAt, &completedAt, &cancelledAt,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Stake{}, ErrStakeNotFound
		}
		return Stake{}, fmt.Errorf("find stake: %w", err)
	}
	applyNullTimes(&s, forfeitedAt, completedAt, cancelledAt)
	return s, nil
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, stakeID int64, status StakeStatus, now time.Time) error {
	var query string
	switch status {
	case StatusForfeited:
		query = `UPDATE stakes SET status = $2, forfeited_at = $3, updated_at = $3 WHERE id = $1`
	case StatusCompleted:
		query = `UPDATE stakes SET status = $2, completed_at = $3, updated_at = $3 WHERE id = $1`
	case StatusCancelled:
		query = `UPDATE stakes SET status = $2, cancelled_at = $3, updated_at = $3 WHERE id = $1`
	default:
		query = `UPDATE stakes SET status = $2, updated_at = $3 WHERE id = $1`
	}

	_, err := r.pool.Exec(ctx, query, stakeID, status, now)
	if err != nil {
		return fmt.Errorf("update stake status: %w", err)
	}
	return nil
}

func (r *PostgresRepository) InsertForfeiture(ctx context.Context, stakeID, declaredByUserID int64, reason string) (StakeForfeiture, error) {
	const query = `
		INSERT INTO stake_forfeitures (stake_id, declared_by_user_id, reason)
		VALUES ($1, $2, $3)
		RETURNING id, stake_id, declared_by_user_id, reason, created_at
	`

	var f StakeForfeiture
	err := r.pool.QueryRow(ctx, query, stakeID, declaredByUserID, reason).Scan(
		&f.ID, &f.StakeID, &f.DeclaredByUserID, &f.Reason, &f.CreatedAt,
	)
	if err != nil {
		return StakeForfeiture{}, fmt.Errorf("insert forfeiture: %w", err)
	}
	return f, nil
}

func (r *PostgresRepository) FindGoalParticipants(ctx context.Context, goalID int64) (int64, int64, string, error) {
	const query = `SELECT owner_user_id, buddy_user_id, status FROM goals WHERE id = $1`

	var ownerID, buddyID int64
	var status string
	err := r.pool.QueryRow(ctx, query, goalID).Scan(&ownerID, &buddyID, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, "", ErrStakeNotFound
		}
		return 0, 0, "", fmt.Errorf("find goal participants: %w", err)
	}
	return ownerID, buddyID, status, nil
}

func applyNullTimes(s *Stake, forfeited, completed, cancelled sql.NullTime) {
	if forfeited.Valid {
		t := forfeited.Time
		s.ForfeitedAt = &t
	}
	if completed.Valid {
		t := completed.Time
		s.CompletedAt = &t
	}
	if cancelled.Valid {
		t := cancelled.Time
		s.CancelledAt = &t
	}
}
