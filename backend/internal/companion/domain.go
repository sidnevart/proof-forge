package companion

import (
	"context"
	"errors"
	"time"
)

// Feature identifies a companion trigger use-case.
type Feature string

const (
	FeatureEveningPing      Feature = "evening_ping"
	FeatureWeeklyRecap      Feature = "weekly_recap"
	FeatureLeadWeeklyBrief  Feature = "lead_weekly_brief"
	FeatureStreakReminder   Feature = "streak_reminder"
	FeatureProofDraft       Feature = "proof_draft"
	FeatureBuddyStalled     Feature = "buddy_stalled"
	FeatureGoalRisk         Feature = "goal_risk"
	FeatureStreakMilestone  Feature = "streak_milestone"
	FeatureLeaderFairPlay   Feature = "leader_fair_play"
	FeatureTeamHealth       Feature = "team_health"
)

// Surface identifies where the companion delivers output.
type Surface string

const (
	SurfaceTelegram   Surface = "telegram"
	SurfaceInApp      Surface = "inapp"
	SurfaceEmail      Surface = "email"
	SurfaceWebSocket Surface = "websocket"
)

// CompanionFeatureMeta defines rate limits and default surfaces per feature.
type CompanionFeatureMeta struct {
	Feature      Feature
	MaxPerDay    int
	MaxPerWeek   int
	Surfaces     []Surface
	RequiresTeam bool
}

var FeatureMeta = map[Feature]CompanionFeatureMeta{
	FeatureEveningPing:     {Feature: FeatureEveningPing, MaxPerDay: 1, Surfaces: []Surface{SurfaceTelegram}},
	FeatureWeeklyRecap:     {Feature: FeatureWeeklyRecap, MaxPerWeek: 1, Surfaces: []Surface{SurfaceTelegram, SurfaceInApp}},
	FeatureLeadWeeklyBrief: {Feature: FeatureLeadWeeklyBrief, MaxPerWeek: 1, Surfaces: []Surface{SurfaceTelegram, SurfaceEmail}, RequiresTeam: true},
	FeatureStreakReminder:  {Feature: FeatureStreakReminder, MaxPerDay: 1, Surfaces: []Surface{SurfaceTelegram}},
	FeatureProofDraft:      {Feature: FeatureProofDraft, MaxPerWeek: 5, Surfaces: []Surface{SurfaceInApp}},
	FeatureBuddyStalled:    {Feature: FeatureBuddyStalled, MaxPerDay: 1, Surfaces: []Surface{SurfaceTelegram, SurfaceInApp}},
	FeatureGoalRisk:        {Feature: FeatureGoalRisk, MaxPerDay: 1, Surfaces: []Surface{SurfaceInApp, SurfaceTelegram}},
	FeatureStreakMilestone: {Feature: FeatureStreakMilestone, MaxPerDay: 1, Surfaces: []Surface{SurfaceTelegram, SurfaceWebSocket}},
	FeatureLeaderFairPlay:  {Feature: FeatureLeaderFairPlay, MaxPerWeek: 1, Surfaces: []Surface{SurfaceTelegram}},
	FeatureTeamHealth:      {Feature: FeatureTeamHealth, MaxPerWeek: 1, Surfaces: []Surface{SurfaceTelegram, SurfaceEmail}, RequiresTeam: true},
}

// Errors.
var (
	ErrAlreadyFired      = errors.New("companion trigger already fired for this user")
	ErrRateLimitExceeded = errors.New("companion rate limit exceeded")
	ErrFeatureDisabled   = errors.New("companion feature is disabled")
	ErrNoTelegramLinked  = errors.New("user has no linked telegram")
)

// ProofDraft is an AI-assembled proof candidate.
type ProofDraft struct {
	ID         string
	UserID     int64
	TeamID     int64
	GoalID     *int64
	NoteIDs    []int64
	Rationale  string
	Confidence string
	CreatedAt  time.Time
}

// Notification is an in-app AI insight.
type Notification struct {
	ID          string
	UserID      int64
	Feature     Feature
	Title       string
	Body        string
	Actions     []NotificationAction
	DismissedAt *time.Time
	CreatedAt   time.Time
}

// NotificationAction is a button inside an in-app notification.
type NotificationAction struct {
	Label  string `json:"label"`
	Action string `json:"action"`
	URL    string `json:"url,omitempty"`
}

// CacheEntry is a cached AI output.
type CacheEntry struct {
	CacheKey   string
	Feature    Feature
	Payload    []byte
	ExpiresAt  time.Time
	CreatedAt  time.Time
}

// DeliveryPayload is what gets sent to a surface.
type DeliveryPayload struct {
	Text    string
	Actions []NotificationAction
}

// Repository is the storage contract for companion state.
type Repository interface {
	// ai_companion_fired
	HasFired(ctx context.Context, userID int64, triggerID string) (bool, error)
	RecordFired(ctx context.Context, userID int64, feature Feature, triggerID string) error
	CountFiredToday(ctx context.Context, userID int64, feature Feature) (int, error)
	CountFiredThisWeek(ctx context.Context, userID int64, feature Feature) (int, error)

	// ai_proof_drafts
	SaveProofDraft(ctx context.Context, draft *ProofDraft) error
	GetProofDrafts(ctx context.Context, userID int64, activeOnly bool) ([]*ProofDraft, error)
	GetProofDraftByID(ctx context.Context, userID int64, id string) (*ProofDraft, error)
	ConsumeProofDraft(ctx context.Context, userID int64, id string) error

	// ai_companion_cache
	GetCache(ctx context.Context, cacheKey string) (*CacheEntry, error)
	SetCache(ctx context.Context, entry *CacheEntry) error
	DeleteCache(ctx context.Context, cacheKey string) error

	// ai_notifications
	SaveNotification(ctx context.Context, n *Notification) error
	GetNotifications(ctx context.Context, userID int64, activeOnly bool) ([]*Notification, error)
	DismissNotification(ctx context.Context, userID int64, id string) error
}
