package analytics

import (
	"errors"
	"time"
)

// EventName is the closed list of analytics events.
type EventName string

const (
	EventDailyLogPrompted        EventName = "daily_log_prompted"
	EventDailyLogSubmitted       EventName = "daily_log_submitted"
	EventDailyLogSkipped         EventName = "daily_log_skipped"
	EventStreakIncremented       EventName = "streak_incremented"
	EventStreakBroken            EventName = "streak_broken"
	EventStreakFrozen            EventName = "streak_frozen"
	EventWeeklyAssemblyPrompted  EventName = "weekly_assembly_prompted"
	EventWeeklyAssemblyCompleted EventName = "weekly_assembly_completed"
	EventProofSubmitted          EventName = "proof_submitted"
	EventProofApproved           EventName = "proof_approved"
	EventProofRejected           EventName = "proof_rejected"
	EventProofCommented          EventName = "proof_commented"
	EventTeamFeedOpened          EventName = "team_feed_opened"
	EventNotificationSent        EventName = "notification_sent"
	EventNotificationClicked     EventName = "notification_clicked"
	EventPersonalizationInvoked  EventName = "personalization_invoked"
	EventPersonalizationFallback EventName = "personalization_fallback"

	// AI Companion events.
	EventCompanionTriggered EventName = "companion_triggered"
	EventCompanionFired     EventName = "companion_fired"
	EventCompanionFallback  EventName = "companion_fallback"
	EventCompanionDismissed EventName = "companion_dismissed"
	EventCompanionAccepted  EventName = "companion_accepted"
	EventCompanionRejected  EventName = "companion_rejected"

	// Pilot instrumentation events.
	EventProofContractCreated EventName = "proof_contract_created"
	EventFirstProofSubmitted  EventName = "first_proof_submitted"
	EventWeeklyProofSubmitted EventName = "weekly_proof_submitted"
	EventBuddyMatched         EventName = "buddy_matched"
	EventSeasonCompleted      EventName = "season_completed"
	EventWorkspaceCreated     EventName = "workspace_created"
	EventTeamspaceCreated     EventName = "teamspace_created"
	EventAiSuggestionUsed     EventName = "ai_suggestion_used"
	EventAntiProofSubmitted   EventName = "anti_proof_submitted"
)

// Source identifies who emitted the event.
type Source string

const (
	SourceWeb         Source = "web"
	SourceTelegramBot Source = "telegram_bot"
	SourceCron        Source = "cron"
	SourceSystem      Source = "system"
)

// Event is a single analytics row.
type Event struct {
	ID         int64
	Ts         time.Time
	EventName  string
	UserID     *int64
	TeamID     *int64
	GoalID     *int64
	ProofID    *int64
	Source     Source
	Properties map[string]any
}

// AllowedFrontendEvents is the whitelist for POST /v1/analytics/event.
var AllowedFrontendEvents = map[EventName]bool{
	EventTeamFeedOpened:      true,
	EventNotificationClicked: true,
	EventAiSuggestionUsed:    true,
	EventAntiProofSubmitted:  true,
	EventCompanionDismissed:  true,
	EventCompanionAccepted:   true,
	EventCompanionRejected:   true,
}

var (
	ErrUnknownEvent     = errors.New("unknown event name")
	ErrForbiddenSource  = errors.New("mutation events must be server-side")
	ErrInvalidProperty  = errors.New("property contains forbidden key")
)

// ForbiddenKeys are never allowed in properties.
var ForbiddenKeys = []string{
	"text_content", "comment", "prompt", "completion", "invite_code", "token",
}

// ValidateProperties rejects maps that contain forbidden keys.
func ValidateProperties(props map[string]any) error {
	for _, key := range ForbiddenKeys {
		if _, ok := props[key]; ok {
			return ErrInvalidProperty
		}
	}
	return nil
}
