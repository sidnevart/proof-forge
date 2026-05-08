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

// LeadBriefContext holds aggregated team data for lead weekly brief.
// Privacy: never includes raw daily-log content.
type LeadBriefContext struct {
	LeaderID    int64
	DisplayName string
	TeamID      int64
	TeamName    string

	PeriodFrom time.Time
	PeriodTo   time.Time

	// Aggregates (only metadata)
	TotalProofsSubmitted int
	TotalProofsApproved  int
	TotalProofsRejected  int
	ApprovalLatencyAvg   string // e.g. "12h"

	Members []MemberBrief
}

// MemberBrief is a member summary without raw content.
type MemberBrief struct {
	UserID      int64
	DisplayName string
	ProofCount  int
	Streak      int
	Status      string // "active", "at_risk", "stalled"
}

// StreakContext holds data for streak risk reminder.
type StreakContext struct {
	UserID        int64
	DisplayName   string
	Timezone      string
	CurrentStreak int
	GoalTitle     string
	HoursLeft     int // hours until midnight
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

// GetLeadBriefContext gathers aggregate team data for a lead weekly brief.
// Privacy: never queries raw daily-log content.
func (a *ContextAssembler) GetLeadBriefContext(ctx context.Context, leaderID, teamID int64, from, to time.Time) (*LeadBriefContext, error) {
	lc := &LeadBriefContext{
		LeaderID: leaderID,
		TeamID:   teamID,
		PeriodFrom: from,
		PeriodTo:   to,
	}

	// 1. Team name
	err := a.pool.QueryRow(ctx,
		`SELECT name FROM teams WHERE id = $1`, teamID,
	).Scan(&lc.TeamName)
	if err != nil {
		return nil, fmt.Errorf("assembler team: %w", err)
	}

	// 2. Leader display name
	err = a.pool.QueryRow(ctx,
		`SELECT display_name FROM users WHERE id = $1`, leaderID,
	).Scan(&lc.DisplayName)
	if err != nil {
		return nil, fmt.Errorf("assembler leader: %w", err)
	}

	// 3. Team aggregates (submitted/approved/rejected counts)
	err = a.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE ci.status IN ('submitted', 'approved')) AS submitted,
			COUNT(*) FILTER (WHERE ci.status = 'approved') AS approved,
			COUNT(*) FILTER (WHERE ci.status = 'rejected') AS rejected
		FROM check_ins ci
		JOIN goals g ON g.id = ci.goal_id
		JOIN team_memberships tm ON tm.user_id = g.owner_user_id
		WHERE tm.team_id = $1
		  AND ci.submitted_at >= $2 AND ci.submitted_at <= $3
	`, teamID, from, to).Scan(&lc.TotalProofsSubmitted, &lc.TotalProofsApproved, &lc.TotalProofsRejected)
	if err != nil {
		return nil, fmt.Errorf("assembler team aggregates: %w", err)
	}

	// 4. Approval latency avg
	var latencyHours float64
	_ = a.pool.QueryRow(ctx, `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (ci.approved_at - ci.submitted_at)) / 3600), 0)
		FROM check_ins ci
		JOIN goals g ON g.id = ci.goal_id
		JOIN team_memberships tm ON tm.user_id = g.owner_user_id
		WHERE tm.team_id = $1
		  AND ci.status = 'approved'
		  AND ci.approved_at IS NOT NULL
		  AND ci.submitted_at >= $2 AND ci.submitted_at <= $3
	`, teamID, from, to).Scan(&latencyHours)
	lc.ApprovalLatencyAvg = fmt.Sprintf("%.0fh", latencyHours)

	// 5. Member summaries (metadata only)
	rows, err := a.pool.Query(ctx, `
		SELECT
			u.id,
			u.display_name,
			COUNT(ci.id) AS proof_count,
			COALESCE(MAX(g.current_streak_count), 0) AS streak
		FROM team_memberships tm
		JOIN users u ON u.id = tm.user_id
		LEFT JOIN goals g ON g.owner_user_id = u.id AND g.status = 'active'
		LEFT JOIN check_ins ci ON ci.goal_id = g.id
		  AND ci.submitted_at >= $2 AND ci.submitted_at <= $3
		WHERE tm.team_id = $1
		GROUP BY u.id, u.display_name
		ORDER BY proof_count DESC
	`, teamID, from, to)
	if err != nil {
		return nil, fmt.Errorf("assembler members: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var m MemberBrief
		if err := rows.Scan(&m.UserID, &m.DisplayName, &m.ProofCount, &m.Streak); err != nil {
			continue
		}
		if m.ProofCount == 0 {
			m.Status = "stalled"
		} else if m.Streak < 3 {
			m.Status = "at_risk"
		} else {
			m.Status = "active"
		}
		lc.Members = append(lc.Members, m)
	}

	return lc, nil
}

// GetStreakContext gathers data for streak reminder.
func (a *ContextAssembler) GetStreakContext(ctx context.Context, userID int64) (*StreakContext, error) {
	sc := &StreakContext{UserID: userID}

	var tz string
	err := a.pool.QueryRow(ctx, `
		SELECT display_name, COALESCE(timezone, 'UTC')
		FROM users WHERE id = $1
	`, userID).Scan(&sc.DisplayName, &tz)
	if err != nil {
		return nil, fmt.Errorf("assembler user: %w", err)
	}
	sc.Timezone = tz

	// Current streak (max across active goals)
	err = a.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(current_streak_count), 0) FROM goals
		WHERE owner_user_id = $1 AND status = 'active' AND archived_at IS NULL
	`, userID).Scan(&sc.CurrentStreak)
	if err != nil {
		return nil, fmt.Errorf("assembler streak: %w", err)
	}

	// Top active goal title
	var goalTitle string
	_ = a.pool.QueryRow(ctx, `
		SELECT title FROM goals
		WHERE owner_user_id = $1 AND status = 'active' AND archived_at IS NULL
		ORDER BY current_streak_count DESC, updated_at DESC
		LIMIT 1
	`, userID).Scan(&goalTitle)
	sc.GoalTitle = goalTitle

	// Hours left until midnight in user's timezone
	loc, _ := time.LoadLocation(tz)
	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
	sc.HoursLeft = int(midnight.Sub(now).Hours())

	return sc, nil
}
