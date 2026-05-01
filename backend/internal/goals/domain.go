package goals

import (
	"errors"
	"strings"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

type GoalStatus string
type PactStatus string
type InviteStatus string
type ProgressHealth string

const (
	GoalStatusPendingBuddyAcceptance GoalStatus = "pending_buddy_acceptance"
	GoalStatusActive                 GoalStatus = "active"

	PactStatusInvited PactStatus = "invited"
	PactStatusActive  PactStatus = "active"

	InviteStatusPending  InviteStatus = "pending"
	InviteStatusAccepted InviteStatus = "accepted"

	ProgressHealthUnknown ProgressHealth = "unknown"
)

var (
	ErrInvalidGoalInput       = errors.New("invalid goal input")
	ErrInviteNotFound         = errors.New("invite not found")
	ErrInviteExpired          = errors.New("invite expired")
	ErrInviteAlreadyAccepted  = errors.New("invite already accepted")
	ErrUnauthorizedAcceptance = errors.New("only the invited buddy can accept this invite")
)

// InviteRecord is the read model returned when looking up an invite by token.
// It carries just enough for the buddy to decide whether to accept.
type InviteRecord struct {
	InviteID     int64
	InviteStatus InviteStatus
	ExpiresAt    time.Time
	GoalID       int64
	GoalTitle    string
	GoalStatus   GoalStatus
	PactID       int64
	InviterID    int64
	InviteeID    int64
	InviteeEmail string
	OwnerName    string
}

type CreateInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	BuddyName   string `json:"buddy_name"`
	BuddyEmail  string `json:"buddy_email"`
}

type Goal struct {
	ID                    int64          `json:"id"`
	Title                 string         `json:"title"`
	Description           string         `json:"description"`
	Status                GoalStatus     `json:"status"`
	CurrentProgressHealth ProgressHealth `json:"current_progress_health"`
	CurrentStreakCount    int            `json:"current_streak_count"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

type Buddy struct {
	ID          int64  `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type Pact struct {
	ID         int64      `json:"id"`
	Status     PactStatus `json:"status"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
}

type Invite struct {
	ID              int64        `json:"id"`
	Status          InviteStatus `json:"status"`
	ExpiresAt       time.Time    `json:"expires_at"`
	AcceptanceToken string       `json:"acceptance_token,omitempty"`
}

type GoalView struct {
	Goal   Goal   `json:"goal"`
	Buddy  Buddy  `json:"buddy"`
	Pact   Pact   `json:"pact"`
	Invite Invite `json:"invite"`
}

type DashboardSummary struct {
	TotalGoals              int `json:"total_goals"`
	PendingBuddyAcceptance  int `json:"pending_buddy_acceptance"`
	ActiveGoals             int `json:"active_goals"`
}

type Dashboard struct {
	Summary DashboardSummary `json:"summary"`
	Goals   []GoalView       `json:"goals"`
}

func (in CreateInput) Normalize() CreateInput {
	return CreateInput{
		Title:       strings.TrimSpace(in.Title),
		Description: strings.TrimSpace(in.Description),
		BuddyName:   strings.TrimSpace(in.BuddyName),
		BuddyEmail:  strings.ToLower(strings.TrimSpace(in.BuddyEmail)),
	}
}

func (in CreateInput) Validate(owner users.User) error {
	normalized := in.Normalize()

	switch {
	case normalized.Title == "":
		return errors.Join(ErrInvalidGoalInput, errors.New("title is required"))
	case len(normalized.BuddyName) < 2:
		return errors.Join(ErrInvalidGoalInput, errors.New("buddy_name must be at least 2 characters"))
	case normalized.BuddyEmail == "" || !strings.Contains(normalized.BuddyEmail, "@"):
		return errors.Join(ErrInvalidGoalInput, errors.New("valid buddy_email is required"))
	case normalized.BuddyEmail == strings.ToLower(strings.TrimSpace(owner.Email)):
		return errors.Join(ErrInvalidGoalInput, errors.New("buddy_email must belong to another person"))
	default:
		return nil
	}
}
