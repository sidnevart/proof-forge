package recaps

import (
	"context"
	"errors"
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

func (r *PostgresRepository) FindGoalsNeedingRecap(ctx context.Context, periodStart, periodEnd time.Time) ([]GoalForRecap, error) {
	const q = `
		SELECT DISTINCT g.id, g.owner_user_id, g.title, g.description
		FROM goals g
		JOIN check_ins ci ON ci.goal_id = g.id
		WHERE g.status = 'active'
		  AND ci.status = 'approved'
		  AND ci.approved_at >= $1
		  AND ci.approved_at < $2
		  AND NOT EXISTS (
		      SELECT 1 FROM weekly_recaps wr
		      WHERE wr.goal_id = g.id
		        AND wr.period_start = $1
		        AND wr.period_end = $2
		        AND wr.status IN ('done', 'generating', 'pending')
		  )
	`
	rows, err := r.pool.Query(ctx, q, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []GoalForRecap
	for rows.Next() {
		var g GoalForRecap
		if err := rows.Scan(&g.ID, &g.OwnerUserID, &g.Title, &g.Description); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, rows.Err()
}

func (r *PostgresRepository) FindApprovedCheckIns(ctx context.Context, goalID int64, periodStart, periodEnd time.Time) ([]ApprovedCheckIn, error) {
	const q = `
		SELECT ci.id, ci.approved_at,
		       ei.kind, ei.text_content, ei.external_url
		FROM check_ins ci
		LEFT JOIN evidence_items ei ON ei.check_in_id = ci.id
		WHERE ci.goal_id = $1
		  AND ci.status = 'approved'
		  AND ci.approved_at >= $2
		  AND ci.approved_at < $3
		ORDER BY ci.approved_at ASC, ei.id ASC
	`
	rows, err := r.pool.Query(ctx, q, goalID, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := make(map[int64]*ApprovedCheckIn)
	var order []int64

	for rows.Next() {
		var (
			ciID        int64
			approvedAt  time.Time
			kind        *string
			textContent *string
			externalURL *string
		)
		if err := rows.Scan(&ciID, &approvedAt, &kind, &textContent, &externalURL); err != nil {
			return nil, err
		}
		ci, ok := byID[ciID]
		if !ok {
			ci = &ApprovedCheckIn{ID: ciID, ApprovedAt: approvedAt}
			byID[ciID] = ci
			order = append(order, ciID)
		}
		if kind != nil {
			ev := EvidenceSummary{Kind: *kind}
			if textContent != nil {
				ev.TextContent = *textContent
			}
			if externalURL != nil {
				ev.ExternalURL = *externalURL
			}
			ci.Evidence = append(ci.Evidence, ev)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]ApprovedCheckIn, 0, len(order))
	for _, id := range order {
		result = append(result, *byID[id])
	}
	return result, nil
}

func (r *PostgresRepository) InsertRecap(ctx context.Context, p InsertRecapParams) (WeeklyRecap, error) {
	const q = `
		INSERT INTO weekly_recaps (goal_id, owner_user_id, period_start, period_end, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, goal_id, owner_user_id, period_start, period_end,
		          status, summary_text, model_name, generated_at, created_at
	`
	row := r.pool.QueryRow(ctx, q, p.GoalID, p.OwnerUserID, p.PeriodStart, p.PeriodEnd, string(p.Status))
	return scanRecap(row)
}

func (r *PostgresRepository) UpdateRecap(ctx context.Context, p UpdateRecapParams) error {
	var modelName *string
	var generatedAt *time.Time
	if p.Status == StatusDone {
		modelName = &p.ModelName
		t := p.GeneratedAt
		generatedAt = &t
	}
	const q = `
		UPDATE weekly_recaps
		SET status = $2, summary_text = $3, model_name = $4, generated_at = $5
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, q, p.ID, string(p.Status), p.SummaryText, modelName, generatedAt)
	return err
}

func (r *PostgresRepository) GetRecap(ctx context.Context, recapID int64) (WeeklyRecap, int64, error) {
	const q = `
		SELECT wr.id, wr.goal_id, wr.owner_user_id, wr.period_start, wr.period_end,
		       wr.status, wr.summary_text, wr.model_name, wr.generated_at, wr.created_at,
		       g.buddy_user_id
		FROM weekly_recaps wr
		JOIN goals g ON g.id = wr.goal_id
		WHERE wr.id = $1
	`
	row := r.pool.QueryRow(ctx, q, recapID)

	var (
		recap     WeeklyRecap
		buddyID   int64
		modelName *string
		genAt     *time.Time
	)
	err := row.Scan(
		&recap.ID, &recap.GoalID, &recap.OwnerUserID,
		&recap.PeriodStart, &recap.PeriodEnd,
		&recap.Status, &recap.SummaryText, &modelName, &genAt, &recap.CreatedAt,
		&buddyID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WeeklyRecap{}, 0, ErrRecapNotFound
		}
		return WeeklyRecap{}, 0, err
	}
	if modelName != nil {
		recap.ModelName = *modelName
	}
	recap.GeneratedAt = genAt
	return recap, buddyID, nil
}

func (r *PostgresRepository) ListRecapsByGoal(ctx context.Context, goalID, actorID int64) ([]WeeklyRecap, error) {
	const q = `
		SELECT wr.id, wr.goal_id, wr.owner_user_id, wr.period_start, wr.period_end,
		       wr.status, wr.summary_text, wr.model_name, wr.generated_at, wr.created_at
		FROM weekly_recaps wr
		JOIN goals g ON g.id = wr.goal_id
		WHERE wr.goal_id = $1
		  AND (g.owner_user_id = $2 OR g.buddy_user_id = $2)
		ORDER BY wr.period_start DESC
	`
	rows, err := r.pool.Query(ctx, q, goalID, actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recaps []WeeklyRecap
	for rows.Next() {
		var (
			recap     WeeklyRecap
			modelName *string
			genAt     *time.Time
		)
		err := rows.Scan(
			&recap.ID, &recap.GoalID, &recap.OwnerUserID,
			&recap.PeriodStart, &recap.PeriodEnd,
			&recap.Status, &recap.SummaryText, &modelName, &genAt, &recap.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if modelName != nil {
			recap.ModelName = *modelName
		}
		recap.GeneratedAt = genAt
		recaps = append(recaps, recap)
	}
	return recaps, rows.Err()
}

func scanRecap(row pgx.Row) (WeeklyRecap, error) {
	var (
		recap     WeeklyRecap
		modelName *string
		genAt     *time.Time
	)
	err := row.Scan(
		&recap.ID, &recap.GoalID, &recap.OwnerUserID,
		&recap.PeriodStart, &recap.PeriodEnd,
		&recap.Status, &recap.SummaryText, &modelName, &genAt, &recap.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WeeklyRecap{}, ErrRecapNotFound
		}
		return WeeklyRecap{}, err
	}
	if modelName != nil {
		recap.ModelName = *modelName
	}
	recap.GeneratedAt = genAt
	return recap, nil
}
