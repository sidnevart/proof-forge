package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type db struct {
	pool *pgxpool.Pool
}

// proofsInWeek returns the count of approved check-ins in the given ISO week.
// weekStart must be Monday 00:00 UTC.
func (d *db) proofsInWeek(ctx context.Context, userID int64, weekStart time.Time) (int, error) {
	var n int
	err := d.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM check_ins
		WHERE owner_user_id = $1
		  AND status = 'approved'
		  AND approved_at >= $2
		  AND approved_at < $2 + INTERVAL '7 days'
	`, userID, weekStart).Scan(&n)
	return n, err
}

// proofStreakWeeks returns consecutive weeks (ending this week) with ≥1 approved check-in.
func (d *db) proofStreakWeeks(ctx context.Context, userID int64) (int, error) {
	// Count backwards from the current ISO-week until we find a week with 0 proofs.
	now := time.Now().UTC()
	weekStart := isoWeekStart(now)

	streak := 0
	for i := 0; i < 104; i++ { // cap at 2 years
		start := weekStart.AddDate(0, 0, -7*i)
		n, err := d.proofsInWeek(ctx, userID, start)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			break
		}
		streak++
	}
	return streak, nil
}

// activeWeeksInLast12 counts weeks with ≥1 approved check-in in the last 12 ISO weeks.
func (d *db) activeWeeksInLast12(ctx context.Context, userID int64) (int, error) {
	var n int
	err := d.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT date_trunc('week', approved_at))
		FROM check_ins
		WHERE owner_user_id = $1
		  AND status = 'approved'
		  AND approved_at >= NOW() - INTERVAL '84 days'
	`, userID).Scan(&n)
	return n, err
}

// personalRecordWeek returns the max approved check-ins in any single ISO week.
func (d *db) personalRecordWeek(ctx context.Context, userID int64) (int, error) {
	var n int
	err := d.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(cnt), 0)
		FROM (
			SELECT COUNT(*) AS cnt
			FROM check_ins
			WHERE owner_user_id = $1
			  AND status = 'approved'
			GROUP BY date_trunc('week', approved_at)
		) sub
	`, userID).Scan(&n)
	return n, err
}

// seasonCompletionPct returns challenge completion % (0-100) based on proof_contracts.
func (d *db) seasonCompletionPct(ctx context.Context, userID int64) (int, error) {
	var total, fulfilled int
	err := d.pool.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'fulfilled')
		FROM proof_contracts
		WHERE user_id = $1
		  AND status IN ('active', 'fulfilled')
	`, userID).Scan(&total, &fulfilled)
	if err != nil || total == 0 {
		return 0, err
	}
	return fulfilled * 100 / total, nil
}

// nextProofExpectedAt returns the earliest due_at from active contracts.
func (d *db) nextProofExpectedAt(ctx context.Context, userID int64) (*time.Time, error) {
	var t time.Time
	err := d.pool.QueryRow(ctx, `
		SELECT due_at FROM proof_contracts
		WHERE user_id = $1 AND status IN ('pending', 'active')
		ORDER BY due_at ASC
		LIMIT 1
	`, userID).Scan(&t)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (d *db) activeGoalsCount(ctx context.Context, userID int64) (int, error) {
	var n int
	err := d.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM goals
		WHERE owner_user_id = $1 AND status = 'active'
	`, userID).Scan(&n)
	return n, err
}

func (d *db) pendingContractsCount(ctx context.Context, userID int64) (int, error) {
	var n int
	err := d.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM proof_contracts
		WHERE user_id = $1 AND status IN ('pending', 'active')
	`, userID).Scan(&n)
	return n, err
}

// ── NowCard queries ───────────────────────────────────────────────────

func (d *db) pendingBuddyReview(ctx context.Context, userID int64, olderThan time.Duration) (*pendingReview, error) {
	row := d.pool.QueryRow(ctx, `
		SELECT cr.id, ci.id, u.display_name, cr.created_at, g.id
		FROM check_in_reviews cr
		JOIN check_ins ci ON ci.id = cr.check_in_id
		JOIN goals g ON g.id = ci.goal_id
		JOIN users u ON u.id = cr.reviewer_user_id
		WHERE g.owner_user_id = $1
		  AND cr.created_at < NOW() - ($2 * INTERVAL '1 second')
		  AND ci.status = 'changes_requested'
		ORDER BY cr.created_at ASC
		LIMIT 1
	`, userID, int(olderThan.Seconds()))

	var r pendingReview
	var reviewID int64
	if err := row.Scan(&reviewID, &r.CheckInID, &r.BuddyName, &r.WaitingSince, &r.GoalID); err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("pending buddy review: %w", err)
	}
	return &r, nil
}

func (d *db) brokenContract(ctx context.Context, userID int64) (*brokenContract, error) {
	row := d.pool.QueryRow(ctx, `
		SELECT pc.id, pc.what_to_prove, g.title, g.id
		FROM proof_contracts pc
		JOIN goals g ON g.id = pc.goal_id
		WHERE pc.user_id = $1 AND pc.status = 'broken'
		ORDER BY pc.due_at ASC
		LIMIT 1
	`, userID)

	var c brokenContract
	var id int64
	if err := row.Scan(&id, &c.WhatToProve, &c.GoalTitle, &c.GoalID); err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("broken contract: %w", err)
	}
	return &c, nil
}

func (d *db) contractDueWithin(ctx context.Context, userID int64, from, to time.Time) (*upcomingContract, error) {
	row := d.pool.QueryRow(ctx, `
		SELECT pc.id, pc.what_to_prove, g.title, g.id, pc.due_at
		FROM proof_contracts pc
		JOIN goals g ON g.id = pc.goal_id
		WHERE pc.user_id = $1 AND pc.status IN ('pending', 'active')
		  AND pc.due_at >= $2 AND pc.due_at < $3
		ORDER BY pc.due_at ASC
		LIMIT 1
	`, userID, from, to)

	var c upcomingContract
	var id int64
	if err := row.Scan(&id, &c.WhatToProve, &c.GoalTitle, &c.GoalID, &c.DueAt); err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("contract due within: %w", err)
	}
	return &c, nil
}

func (d *db) inactiveRegularGoal(ctx context.Context, userID int64, minDays int) (*inactiveGoal, error) {
	row := d.pool.QueryRow(ctx, `
		SELECT g.id, g.title,
		       EXTRACT(DAY FROM NOW() - COALESCE(MAX(ci.approved_at), g.created_at))::INT AS days_since
		FROM goals g
		LEFT JOIN check_ins ci ON ci.goal_id = g.id AND ci.status = 'approved'
		WHERE g.owner_user_id = $1
		  AND g.status = 'active'
		  AND g.movement_mode = 'regular_rhythm'
		GROUP BY g.id, g.title, g.created_at
		HAVING EXTRACT(DAY FROM NOW() - COALESCE(MAX(ci.approved_at), g.created_at)) >= $2
		ORDER BY days_since DESC
		LIMIT 1
	`, userID, minDays)

	var ig inactiveGoal
	if err := row.Scan(&ig.GoalID, &ig.GoalTitle, &ig.DaysSinceLastProof); err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("inactive goal: %w", err)
	}
	return &ig, nil
}

func (d *db) hasActiveGoals(ctx context.Context, userID int64) (bool, error) {
	var ok bool
	err := d.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM goals WHERE owner_user_id = $1 AND status = 'active')
	`, userID).Scan(&ok)
	return ok, err
}

// ── helpers ───────────────────────────────────────────────────────────

func isoWeekStart(t time.Time) time.Time {
	t = t.UTC()
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := t.AddDate(0, 0, -(weekday - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

func isNoRows(err error) bool {
	return err == pgx.ErrNoRows
}
