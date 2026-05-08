package goals

import (
	"context"
	"time"
)

type Repository interface {
	CreateGoalWithInvite(context.Context, CreateGoalParams) (GoalView, error)
	// ListGoalsForUser returns every goal where the user is either the owner
	// or the invited buddy. Each row carries a ViewerRole computed against the
	// caller's userID so the UI can label «вы — автор» / «вы — партнёр».
	ListGoalsForUser(context.Context, int64) ([]GoalView, error)
	FindGoalRefineCache(ctx context.Context, draftHash string, minCreatedAt time.Time) (GoalRefineResponse, bool, error)
	SaveGoalRefineCache(ctx context.Context, draftHash string, response GoalRefineResponse) error
	CountGoalRefineRequestsSince(ctx context.Context, userID int64, since time.Time) (int, error)
	InsertGoalRefineRequest(ctx context.Context, params GoalRefineRequestLogParams) error
	FindInviteByToken(ctx context.Context, tokenHash string) (InviteRecord, error)
	AcceptInvite(ctx context.Context, params AcceptInviteParams) error
	IsCircleMember(ctx context.Context, circleID int64, userID int64) (bool, error)
	IsCircleMemberByEmail(ctx context.Context, circleID int64, email string) (bool, error)
	HasActiveGoalInCircle(ctx context.Context, circleID int64, ownerID int64) (bool, error)
}

type AcceptInviteParams struct {
	InviteID   int64
	PactID     int64
	GoalID     int64
	AcceptedAt time.Time
}

type GoalRefineRequestLogParams struct {
	UserID    int64
	DraftHash string
}

// CreateGoalParams describes a request to create a goal with its invite. When
// CircleID is non-nil, the goal is attached to that existing circle. When
// CircleID is nil, AutoCircle MUST be populated and the repository creates a
// brand-new circle (with a 7-day season and an owner membership) atomically in
// the same transaction as the goal/pact/invite.
type CreateGoalParams struct {
	OwnerID               int64
	OwnerEmail            string
	Title                 string
	Description           string
	BuddyName             string
	BuddyEmail            string
	ProofExamples         string
	Category              string
	CircleID              *int64
	AutoCircle            *AutoCircleParams
	GoalStatus            GoalStatus
	PactStatus            PactStatus
	InviteStatus          InviteStatus
	ProgressHealth        ProgressHealth
	InviteTokenHash       string
	InviteExpiresAt       time.Time
	MovementMode          MovementMode
	RhythmCadence         *RhythmCadence
	ChallengeDurationDays *int
	ChallengeStartsAt     *time.Time
	ChallengeEndsAt       *time.Time
}

// AutoCircleParams supplies the fields needed to create a fresh circle alongside
// the goal when the caller did not pick an existing circle. The repository owns
// the actual SQL; the service decides the values (name, invite code, season
// window) so they stay deterministic in tests.
type AutoCircleParams struct {
	Name        string
	InviteCode  string
	MemberLimit int
	StartsAt    time.Time
	EndsAt      time.Time
}
