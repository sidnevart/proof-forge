package circles

import (
	"context"
	"time"
)

type Repository interface {
	CreateCircle(context.Context, CreateCircleParams) (Detail, error)
	ListCirclesForUser(context.Context, int64) ([]Detail, error)
	GetCircleForUser(context.Context, int64, int64) (Detail, error)
	JoinCircle(context.Context, JoinCircleParams) (Detail, error)
	ListStandingSnapshots(context.Context, int64, time.Time, time.Time, time.Time, time.Time, time.Time, time.Time) ([]StandingSnapshot, error)
	InviteToCircle(ctx context.Context, params InviteToCircleParams) (CircleInvitation, error)
	ListInvitationsForUser(ctx context.Context, targetEmail string) ([]CircleInvitation, error)
	AcceptCircleInvitation(ctx context.Context, invitationID, userID int64) error
	DeclineCircleInvitation(ctx context.Context, invitationID, userID int64) error

	// IsActiveMemberOfGoalCircle checks whether userID is an active member of
	// the circle that owns the given goal.
	IsActiveMemberOfGoalCircle(ctx context.Context, userID, goalID int64) (bool, error)

	// Season end actions
	GetSeason(ctx context.Context, circleID, seasonID int64) (Season, error)
	MarkSeasonCompleted(ctx context.Context, seasonID int64, action SeasonEndAction) error
	CreateExtendedSeason(ctx context.Context, circleID int64, startsAt, endsAt time.Time) (Season, error)
	LazyCompleteExpiredSeason(ctx context.Context, circleID int64) error
}

type CreateCircleParams struct {
	OwnerUserID int64
	Name        string
	InviteCode  string
	MemberLimit int
	StartsAt    time.Time
	EndsAt      time.Time
}

type JoinCircleParams struct {
	UserID     int64
	InviteCode string
	JoinedAt   time.Time
}

type InviteToCircleParams struct {
	CircleID      int64
	InviterUserID int64
	TargetEmail   string
	Message       string
}
