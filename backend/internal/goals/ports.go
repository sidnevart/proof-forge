package goals

import (
	"context"
	"time"
)

type Repository interface {
	CreateGoalWithInvite(context.Context, CreateGoalParams) (GoalView, error)
	ListGoalsByOwner(context.Context, int64) ([]GoalView, error)
	FindInviteByToken(ctx context.Context, tokenHash string) (InviteRecord, error)
	AcceptInvite(ctx context.Context, params AcceptInviteParams) error
}

type AcceptInviteParams struct {
	InviteID   int64
	PactID     int64
	GoalID     int64
	AcceptedAt time.Time
}

type CreateGoalParams struct {
	OwnerID          int64
	OwnerEmail       string
	Title            string
	Description      string
	BuddyName        string
	BuddyEmail       string
	GoalStatus       GoalStatus
	PactStatus       PactStatus
	InviteStatus     InviteStatus
	ProgressHealth   ProgressHealth
	InviteTokenHash  string
	InviteExpiresAt  time.Time
}
