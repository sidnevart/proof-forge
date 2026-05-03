package goals

import (
	"context"
	"time"
)

type Repository interface {
	CreateGoalWithInvite(context.Context, CreateGoalParams) (GoalView, error)
	ListGoalsForUser(ctx context.Context, userID int64) ([]GoalView, error)
	FindGoalForUser(ctx context.Context, goalID, userID int64) (GoalView, error)
	FindInviteByToken(ctx context.Context, tokenHash string) (InviteRecord, error)
	FindInviteByGoal(ctx context.Context, goalID int64) (InviteRecord, error)
	AcceptInvite(ctx context.Context, params AcceptInviteParams) error
	// DeleteGoal removes the goal and cascades to pacts, invites, check-ins,
	// evidence, reviews, milestones, stakes, and recaps via FK ON DELETE CASCADE.
	// Only deletes when ownerUserID matches; returns ErrGoalNotFound otherwise.
	DeleteGoal(ctx context.Context, goalID, ownerUserID int64) error
	// SetGoalDeadline updates the deadline_at column. Pass nil to clear.
	// Only updates when ownerUserID matches; returns ErrGoalNotFound otherwise.
	SetGoalDeadline(ctx context.Context, goalID, ownerUserID int64, deadline *time.Time) error
}

type AcceptInviteParams struct {
	InviteID   int64
	PactID     int64
	GoalID     int64
	AcceptedAt time.Time
}

type CreateGoalParams struct {
	OwnerID         int64
	OwnerEmail      string
	Title           string
	Description     string
	BuddyName       string
	BuddyEmail      string
	GoalStatus      GoalStatus
	PactStatus      PactStatus
	InviteStatus    InviteStatus
	ProgressHealth  ProgressHealth
	InviteTokenHash string
	InviteExpiresAt time.Time
	DeadlineAt      *time.Time
}
