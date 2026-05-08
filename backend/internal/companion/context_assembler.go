package companion

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ContextAssembler collects user data for companion features.
type ContextAssembler struct {
	pool *pgxpool.Pool
}

// NewContextAssembler creates a new assembler.
func NewContextAssembler(pool *pgxpool.Pool) *ContextAssembler {
	return &ContextAssembler{pool: pool}
}

// UserContext holds the minimal data needed for evening ping.
type UserContext struct {
	UserID      int64
	DisplayName string
	Timezone    string // e.g. "Europe/Moscow", default UTC

	// Streak state (from the most active goal or aggregated)
	CurrentStreak int
	BestStreak    int

	// Goals
	ActiveGoalsCount int
	ActiveGoalTitles []string

	// Proofs today
	ProofsToday int

	// Day of week
	DayOfWeek time.Weekday
	LocalTime time.Time
}

// WeeklyContext holds data for weekly recap.
type WeeklyContext struct {
	UserID      int64
	DisplayName string
	PeriodFrom  time.Time
	PeriodTo    time.Time

	TotalProofs        int
	BuddyApprovedCount int
	ActiveWeeks        int
	CurrentStreak      int

	Goals []GoalWeeklySummary
}

// GoalWeeklySummary is per-goal stats for the week.
type GoalWeeklySummary struct {
	GoalID      int64
	Title       string
	ProofCount  int
	Mode        string
	Status      string
	TopProofs   []string // first 150 chars of each
}

// GetUserContext gathers data for evening ping.
func (a *ContextAssembler) GetUserContext(ctx context.Context, userID int64) (*UserContext, error) {
	uc := &UserContext{UserID: userID}

	// 1. User basics
	var tz string
	err := a.pool.QueryRow(ctx,
		`SELECT display_name, COALESCE(timezone, 'UTC') FROM users WHERE id = $1`,
		userID,
	).Scan(&uc.DisplayName, &tz)
	if err != nil {
		return nil, fmt.Errorf("assembler user: %w", err)
	}
	uc.Timezone = tz

	// 2. Streak from the most active goal
	err = a.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(current_streak_count), 0) FROM goals
		WHERE owner_user_id = $1 AND status = 'active' AND archived_at IS NULL
	`, userID).Scan(&uc.CurrentStreak)
	if err != nil {
		return nil, fmt.Errorf("assembler streak: %w", err)
	}

	// 3. Active goals count and titles
	rows, err := a.pool.Query(ctx, `
		SELECT title FROM goals
		WHERE owner_user_id = $1 AND status = 'active' AND archived_at IS NULL
		ORDER BY updated_at DESC
		LIMIT 5
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("assembler goals: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err == nil {
			uc.ActiveGoalTitles = append(uc.ActiveGoalTitles, title)
			uc.ActiveGoalsCount++
		}
	}

	// 4. Proofs today (approved check-ins today)
	today := time.Now().UTC().Format("2006-01-02")
	err = a.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM check_ins
		WHERE owner_user_id = $1 AND status = 'approved'
		  AND approved_at >= $2::date AND approved_at < ($2::date + INTERVAL '1 day')
	`, userID, today).Scan(&uc.ProofsToday)
	if err != nil {
		return nil, fmt.Errorf("assembler proofs today: %w", err)
	}

	// 5. Local time
	loc, err := time.LoadLocation(uc.Timezone)
	if err != nil {
		loc = time.UTC
	}
	uc.LocalTime = time.Now().In(loc)
	uc.DayOfWeek = uc.LocalTime.Weekday()

	return uc, nil
}

// GetWeeklyContext gathers data for weekly recap.
func (a *ContextAssembler) GetWeeklyContext(ctx context.Context, userID int64, from, to time.Time) (*WeeklyContext, error) {
	wc := &WeeklyContext{
		UserID:     userID,
		PeriodFrom: from,
		PeriodTo:   to,
	}

	// 1. User basics
	err := a.pool.QueryRow(ctx,
		`SELECT display_name FROM users WHERE id = $1`, userID,
	).Scan(&wc.DisplayName)
	if err != nil {
		return nil, fmt.Errorf("assembler user: %w", err)
	}

	// 2. Total proofs and approved count for period
	err = a.pool.QueryRow(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE status = 'approved')
		FROM check_ins
		WHERE owner_user_id = $1
		  AND submitted_at >= $2 AND submitted_at <= $3
	`, userID, from, to).Scan(&wc.TotalProofs, &wc.BuddyApprovedCount)
	if err != nil {
		return nil, fmt.Errorf("assembler proofs: %w", err)
	}

	// 3. Current streak (max across active goals)
	err = a.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(current_streak_count), 0) FROM goals
		WHERE owner_user_id = $1 AND status = 'active' AND archived_at IS NULL
	`, userID).Scan(&wc.CurrentStreak)
	if err != nil {
		return nil, fmt.Errorf("assembler streak: %w", err)
	}

	// 4. Goal summaries for period
	rows, err := a.pool.Query(ctx, `
		SELECT g.id, g.title, g.status,
		       COUNT(ci.id) AS proof_count
		FROM goals g
		LEFT JOIN check_ins ci ON ci.goal_id = g.id
		  AND ci.submitted_at >= $2 AND ci.submitted_at <= $3
		WHERE g.owner_user_id = $1 AND g.archived_at IS NULL
		GROUP BY g.id, g.title, g.status
		ORDER BY proof_count DESC
		LIMIT 10
	`, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("assembler goals: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var gs GoalWeeklySummary
		if err := rows.Scan(&gs.GoalID, &gs.Title, &gs.Status, &gs.ProofCount); err != nil {
			continue
		}
		wc.Goals = append(wc.Goals, gs)
	}

	// 5. Top proofs snippets for each goal
	for i := range wc.Goals {
		proofRows, err := a.pool.Query(ctx, `
			SELECT LEFT(e.text_content, 150)
			FROM check_ins ci
			JOIN evidence_items e ON e.check_in_id = ci.id AND e.kind = 'text'
			WHERE ci.goal_id = $1 AND ci.submitted_at >= $2 AND ci.submitted_at <= $3
			ORDER BY ci.submitted_at DESC
			LIMIT 3
		`, wc.Goals[i].GoalID, from, to)
		if err != nil {
			continue
		}
		for proofRows.Next() {
			var snippet string
			if err := proofRows.Scan(&snippet); err == nil {
				wc.Goals[i].TopProofs = append(wc.Goals[i].TopProofs, snippet)
			}
		}
		proofRows.Close()
	}

	return wc, nil
}
