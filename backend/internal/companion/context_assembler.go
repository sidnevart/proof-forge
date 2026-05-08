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

// ProofDraftContext holds data for proof draft assembly.
type ProofDraftContext struct {
	UserID      int64
	DisplayName string
	TeamID      int64
	TeamName    string
	GoalID      int64
	GoalTitle   string

	Notes []DailyLogNote
}

// DailyLogNote is one unconsumed entry.
type DailyLogNote struct {
	NoteID      int64
	LogDate     time.Time
	TextContent string
	HasArtifact bool
}

// BuddyStalledContext holds data for buddy stalled alert.
type BuddyStalledContext struct {
	UserID      int64
	DisplayName string
	BuddyID     int64
	BuddyName   string
	ProofID     int64
	GoalTitle   string
	SubmittedAt time.Time
	HoursStalled int
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

// GetProofDraftContext gathers unconsumed daily log entries for proof draft assembly.
func (a *ContextAssembler) GetProofDraftContext(ctx context.Context, userID int64) (*ProofDraftContext, error) {
	pdc := &ProofDraftContext{UserID: userID}

	// 1. User basics and active team/goal
	var teamID int64
	var goalID int64
	err := a.pool.QueryRow(ctx, `
		SELECT u.display_name, t.id, t.name, g.id, g.title
		FROM users u
		CROSS JOIN LATERAL (
			SELECT t.id, t.name
			FROM teams t
			JOIN team_memberships tm ON tm.team_id = t.id
			WHERE tm.user_id = u.id AND tm.status = 'active'
			ORDER BY t.created_at DESC
			LIMIT 1
		) t
		CROSS JOIN LATERAL (
			SELECT g.id, g.title
			FROM goals g
			WHERE g.owner_user_id = u.id AND g.status = 'active' AND g.archived_at IS NULL
			ORDER BY g.current_streak_count DESC, g.updated_at DESC
			LIMIT 1
		) g
		WHERE u.id = $1
	`, userID).Scan(&pdc.DisplayName, &teamID, &pdc.TeamName, &goalID, &pdc.GoalTitle)
	if err != nil {
		return nil, fmt.Errorf("assembler user/team/goal: %w", err)
	}
	pdc.TeamID = teamID
	pdc.GoalID = goalID

	// 2. Unconsumed daily log entries for this user+team in last 7 days
	rows, err := a.pool.Query(ctx, `
		SELECT id, log_date, text_content, has_artifact
		FROM daily_log_entries
		WHERE user_id = $1 AND team_id = $2
		  AND consumed_in_check_in_id IS NULL
		  AND status = 'logged'
		  AND log_date >= CURRENT_DATE - INTERVAL '7 days'
		ORDER BY log_date DESC
		LIMIT 10
	`, userID, teamID)
	if err != nil {
		return nil, fmt.Errorf("assembler daily log: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var n DailyLogNote
		if err := rows.Scan(&n.NoteID, &n.LogDate, &n.TextContent, &n.HasArtifact); err != nil {
			continue
		}
		pdc.Notes = append(pdc.Notes, n)
	}

	return pdc, nil
}

// GoalRiskContext holds data for goal risk alert.
type GoalRiskContext struct {
	UserID           int64
	DisplayName      string
	GoalID           int64
	GoalTitle        string
	DaysWithoutProof int
	CurrentStreak    int
	BuddyName        string
	TeamName         string
}

// StreakMilestoneContext holds data for streak milestone celebration.
type StreakMilestoneContext struct {
	UserID        int64
	DisplayName   string
	GoalID        int64
	GoalTitle     string
	Milestone     int // 7, 30, or 100
	CurrentStreak int
	TeamName      string
}

// LeaderFairPlayContext holds data for leader fair play nudge.
type LeaderFairPlayContext struct {
	LeaderID            int64
	DisplayName         string
	TeamID              int64
	TeamName            string
	PeriodFrom          time.Time
	PeriodTo            time.Time
	P95LatencyHours     float64
	MaxLatencyHours     float64
	PendingApprovals    int
	StalledProofsCount  int
}

// GetBuddyStalledContext finds stalled proofs awaiting buddy approval.
func (a *ContextAssembler) GetBuddyStalledContext(ctx context.Context, minHours int) ([]*BuddyStalledContext, error) {
	rows, err := a.pool.Query(ctx, `
		SELECT
			ci.id,
			ci.owner_user_id,
			u.display_name,
			ci.goal_id,
			g.title,
			ci.submitted_at,
			g.buddy_user_id,
			bu.display_name,
			EXTRACT(EPOCH FROM (NOW() - ci.submitted_at)) / 3600
		FROM check_ins ci
		JOIN goals g ON g.id = ci.goal_id
		JOIN users u ON u.id = ci.owner_user_id
		JOIN users bu ON bu.id = g.buddy_user_id
		WHERE ci.status = 'submitted'
		  AND ci.submitted_at <= NOW() - INTERVAL '1 hour' * $1
		  AND ci.approved_at IS NULL
		  AND ci.rejected_at IS NULL
		  AND ci.changes_requested_at IS NULL
		ORDER BY ci.submitted_at ASC
		LIMIT 50
	`, minHours)
	if err != nil {
		return nil, fmt.Errorf("assembler stalled: %w", err)
	}
	defer rows.Close()

	var results []*BuddyStalledContext
	for rows.Next() {
		var b BuddyStalledContext
		var hours float64
		if err := rows.Scan(
			&b.ProofID, &b.UserID, &b.DisplayName,
			&b.GoalTitle, &b.GoalTitle, &b.SubmittedAt,
			&b.BuddyID, &b.BuddyName, &hours,
		); err != nil {
			continue
		}
		b.HoursStalled = int(hours)
		results = append(results, &b)
	}
	return results, nil
}

// GetGoalRiskContext finds active goals with no approved proof in last 14 days.
func (a *ContextAssembler) GetGoalRiskContext(ctx context.Context) ([]*GoalRiskContext, error) {
	rows, err := a.pool.Query(ctx, `
		SELECT
			g.id,
			g.title,
			g.owner_user_id,
			u.display_name,
			COALESCE(g.current_streak_count, 0),
			bu.display_name,
			t.name,
			CURRENT_DATE - COALESCE(MAX(ci.approved_at)::date, g.created_at::date)
		FROM goals g
		JOIN users u ON u.id = g.owner_user_id
		LEFT JOIN users bu ON bu.id = g.buddy_user_id
		LEFT JOIN teams t ON t.id = g.team_id
		LEFT JOIN check_ins ci ON ci.goal_id = g.id AND ci.status = 'approved'
		WHERE g.status = 'active'
		  AND g.archived_at IS NULL
		GROUP BY g.id, g.title, g.owner_user_id, u.display_name, g.current_streak_count, bu.display_name, t.name, g.created_at
		HAVING CURRENT_DATE - COALESCE(MAX(ci.approved_at)::date, g.created_at::date) >= 14
		LIMIT 50
	`)
	if err != nil {
		return nil, fmt.Errorf("assembler goal risk: %w", err)
	}
	defer rows.Close()

	var results []*GoalRiskContext
	for rows.Next() {
		var r GoalRiskContext
		if err := rows.Scan(
			&r.GoalID, &r.GoalTitle, &r.UserID, &r.DisplayName,
			&r.CurrentStreak, &r.BuddyName, &r.TeamName, &r.DaysWithoutProof,
		); err != nil {
			continue
		}
		results = append(results, &r)
	}
	return results, nil
}

// GetStreakMilestoneContext finds active goals that hit 7, 30, or 100 streak today.
func (a *ContextAssembler) GetStreakMilestoneContext(ctx context.Context) ([]*StreakMilestoneContext, error) {
	rows, err := a.pool.Query(ctx, `
		SELECT
			g.owner_user_id,
			u.display_name,
			g.id,
			g.title,
			g.current_streak_count,
			t.name
		FROM goals g
		JOIN users u ON u.id = g.owner_user_id
		LEFT JOIN teams t ON t.id = g.team_id
		WHERE g.status = 'active'
		  AND g.archived_at IS NULL
		  AND g.current_streak_count IN (7, 30, 100)
		  AND NOT EXISTS (
			SELECT 1 FROM ai_companion_fired acf
			WHERE acf.trigger_id = 'streak_milestone:' || g.owner_user_id || ':' || g.id || ':' || g.current_streak_count
		  )
		LIMIT 50
	`)
	if err != nil {
		return nil, fmt.Errorf("assembler streak milestone: %w", err)
	}
	defer rows.Close()

	var results []*StreakMilestoneContext
	for rows.Next() {
		var m StreakMilestoneContext
		if err := rows.Scan(
			&m.UserID, &m.DisplayName, &m.GoalID, &m.GoalTitle,
			&m.CurrentStreak, &m.TeamName,
		); err != nil {
			continue
		}
		m.Milestone = m.CurrentStreak
		results = append(results, &m)
	}
	return results, nil
}

// GetLeaderFairPlayContext gathers approval latency data for leader fair play nudge.
func (a *ContextAssembler) GetLeaderFairPlayContext(ctx context.Context, leaderID, teamID int64, from, to time.Time) (*LeaderFairPlayContext, error) {
	fc := &LeaderFairPlayContext{
		LeaderID: leaderID,
		TeamID:   teamID,
		PeriodFrom: from,
		PeriodTo:   to,
	}

	// Leader display name + team name
	err := a.pool.QueryRow(ctx, `
		SELECT u.display_name, t.name
		FROM users u, teams t
		WHERE u.id = $1 AND t.id = $2
	`, leaderID, teamID).Scan(&fc.DisplayName, &fc.TeamName)
	if err != nil {
		return nil, fmt.Errorf("assembler fair play leader: %w", err)
	}

	// P95 and max approval latency
	_ = a.pool.QueryRow(ctx, `
		SELECT
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (ci.approved_at - ci.submitted_at)) / 3600), 0),
			COALESCE(MAX(EXTRACT(EPOCH FROM (ci.approved_at - ci.submitted_at)) / 3600), 0)
		FROM check_ins ci
		JOIN goals g ON g.id = ci.goal_id
		JOIN team_memberships tm ON tm.user_id = g.owner_user_id
		WHERE tm.team_id = $1
		  AND ci.status = 'approved'
		  AND ci.approved_at IS NOT NULL
		  AND ci.submitted_at >= $2 AND ci.submitted_at <= $3
	`, teamID, from, to).Scan(&fc.P95LatencyHours, &fc.MaxLatencyHours)

	// Pending approvals count
	_ = a.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM check_ins ci
		JOIN goals g ON g.id = ci.goal_id
		JOIN team_memberships tm ON tm.user_id = g.owner_user_id
		WHERE tm.team_id = $1
		  AND ci.status = 'submitted'
	`, teamID).Scan(&fc.PendingApprovals)

	// Stalled proofs (>48h)
	_ = a.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM check_ins ci
		JOIN goals g ON g.id = ci.goal_id
		JOIN team_memberships tm ON tm.user_id = g.owner_user_id
		WHERE tm.team_id = $1
		  AND ci.status = 'submitted'
		  AND ci.submitted_at <= NOW() - INTERVAL '48 hours'
	`, teamID).Scan(&fc.StalledProofsCount)

	return fc, nil
}
