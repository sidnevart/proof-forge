package teams

import (
	"context"
	"time"
)

// Repository is the persistence boundary for the teams package. It is
// implemented by postgres_repository.go (production) and a fakeRepo defined
// in service_test.go (unit tests).
//
// Method semantics — what the repo MUST guarantee:
//
//   - CreateTeam: atomic insert of teams + lead membership; returned Detail
//     has my_role=lead, member_count=1.
//   - JoinTeam: looks up team by invite_code AND archived_at IS NULL FOR UPDATE,
//     enforces member_limit, idempotent for users who already joined this team.
//   - LeaveTeam: refuses to mark a lead as left when they are the ONLY active
//     lead; returns ErrCannotLeaveAsOnlyLead in that case (no row touched).
//   - SetAIConsent: writes to the user's own membership; the service layer is
//     responsible for caller==target check, but the repo accepts any pair.
//   - ChangeMemberRole: any unique-violation on the partial unique index for
//     leads must be translated into a meaningful error (typically
//     ErrMustHaveLead — caller cannot make a team leadless).
type Repository interface {
	// CreateTeam inserts the team row AND the lead's membership row in a
	// single transaction.
	CreateTeam(ctx context.Context, params CreateTeamParams) (Detail, error)

	// ListMyTeams returns all teams where the user has an ACTIVE membership.
	ListMyTeams(ctx context.Context, userID int64) ([]Detail, error)

	// GetTeamForUser returns the team detail for a user who must already be
	// an active member. Returns ErrNotMember if the user has no active
	// membership in the team.
	GetTeamForUser(ctx context.Context, teamID, userID int64) (Detail, error)

	// JoinTeam atomically: looks up the team by invite_code (must not be
	// archived), checks capacity, inserts a 'member' row. Idempotent for
	// already-active members (returns existing detail).
	JoinTeam(ctx context.Context, params JoinTeamParams) (Detail, error)

	// ChangeMemberRole — used only by the lead. Service layer guards calls
	// against demoting self. Repo just performs the UPDATE and refuses if
	// the partial unique index for leads is violated.
	ChangeMemberRole(ctx context.Context, params ChangeMemberRoleParams) error

	// RemoveMember marks the membership as 'removed' and stamps left_at.
	RemoveMember(ctx context.Context, params RemoveMemberParams) error

	// LeaveTeam marks the membership as 'left' and stamps left_at.
	// Returns ErrCannotLeaveAsOnlyLead when the lead is the only one.
	LeaveTeam(ctx context.Context, teamID, userID int64, at time.Time) error

	// SetAIConsent flips ai_consent on a membership. The service layer
	// enforces that the caller is the same user as the target.
	SetAIConsent(ctx context.Context, teamID, userID int64, consent bool) error

	// RegenerateInviteCode replaces the invite code with a new value.
	RegenerateInviteCode(ctx context.Context, teamID int64, newCode string) error

	// ArchiveTeam stamps archived_at on the team row.
	ArchiveTeam(ctx context.Context, teamID int64, at time.Time) error

	// GetMembership is the building block for authz checks. Returns
	// ErrNotMember when no row exists OR status != 'active'.
	GetMembership(ctx context.Context, teamID, userID int64) (Membership, error)
}

type CreateTeamParams struct {
	LeadUserID  int64
	Name        string
	InviteCode  string
	MemberLimit int
	AIMode      AIMode
	CreatedAt   time.Time
}

type JoinTeamParams struct {
	UserID     int64
	InviteCode string
	JoinedAt   time.Time
}

type ChangeMemberRoleParams struct {
	TeamID  int64
	UserID  int64
	NewRole Role
}

type RemoveMemberParams struct {
	TeamID int64
	UserID int64
	At     time.Time
}
