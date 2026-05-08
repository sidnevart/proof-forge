// Package stats computes personal metrics and the "What Now" smart card.
package stats

import "time"

// WeekTrend describes performance compared to last week.
type WeekTrend string

const (
	WeekTrendBetter    WeekTrend = "better"
	WeekTrendSame      WeekTrend = "same"
	WeekTrendWorse     WeekTrend = "worse"
	WeekTrendFirstWeek WeekTrend = "first_week"
)

// UserStats holds all personal metrics for the dashboard.
type UserStats struct {
	ProofStreak           int        `json:"proof_streak"`
	ProofsThisWeek        int        `json:"proofs_this_week"`
	ProofsLastWeek        int        `json:"proofs_last_week"`
	WeekVsLastWeek        int        `json:"week_vs_last_week"`
	WeekTrend             WeekTrend  `json:"week_trend"`
	ActiveWeeks           int        `json:"active_weeks"`
	PersonalRecordWeek    int        `json:"personal_record_week"`
	SeasonCompletionPct   int        `json:"season_completion_pct"`
	NextProofExpectedAt   *time.Time `json:"next_proof_expected_at,omitempty"`
	ActiveGoalsCount      int        `json:"active_goals_count"`
	PendingContractsCount int        `json:"pending_contracts_count"`
}

// NowUrgency maps to CSS design tokens on the frontend.
type NowUrgency string

const (
	NowUrgencyDanger  NowUrgency = "danger"
	NowUrgencyFire    NowUrgency = "fire"
	NowUrgencyWarn    NowUrgency = "warn"
	NowUrgencyWin     NowUrgency = "win"
	NowUrgencyNeutral NowUrgency = "neutral"
)

// NowAction is the single call-to-action in a NowCard.
type NowAction struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// NowCard is the smart card returned by GET /v1/me/now.
// Only one card is returned — the highest-priority item.
type NowCard struct {
	Type     string     `json:"type"`
	Title    string     `json:"title"`
	Subtitle string     `json:"subtitle"`
	Urgency  NowUrgency `json:"urgency"`
	Action   *NowAction `json:"action"`
}

// ── internal query result types ───────────────────────────────────────

type pendingReview struct {
	CheckInID   int64
	GoalID      int64
	BuddyName   string
	WaitingSince time.Time
}

type brokenContract struct {
	GoalID      int64
	GoalTitle   string
	WhatToProve string
}

type inactiveGoal struct {
	GoalID             int64
	GoalTitle          string
	DaysSinceLastProof int
}

type upcomingContract struct {
	GoalID      int64
	GoalTitle   string
	WhatToProve string
	DueAt       time.Time
}
