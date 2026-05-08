package initiatives

import "context"

// Repository is the storage interface for initiatives.
type Repository interface {
	Create(ctx context.Context, input CreateInput, creatorID int64) (Initiative, error)
	FindByID(ctx context.Context, id int64) (Initiative, error)
	ListBySpace(ctx context.Context, spaceType string, spaceID int64) ([]Initiative, error)
	// JoinOrGet creates a goal for the user linked to the initiative, or returns
	// the existing goal ID if the user already joined (idempotent).
	JoinOrGet(ctx context.Context, initiativeID, userID int64) (JoinResult, error)
	// PendingProofs returns submitted check_ins from OTHER participants (not the viewer).
	PendingProofs(ctx context.Context, initiativeID, viewerID int64) ([]PendingProof, error)
	// Approve records a peer approval and marks the check_in as approved.
	Approve(ctx context.Context, checkinID, approverID int64, comment string) error
	// CheckinAuthorID returns the owner_user_id of the given check_in.
	CheckinAuthorID(ctx context.Context, checkinID int64) (int64, error)
	// IsParticipant returns true if the user has a goal linked to the initiative.
	IsParticipant(ctx context.Context, initiativeID, userID int64) (bool, error)
	Participants(ctx context.Context, initiativeID int64) ([]ParticipantProgress, error)
	Archive(ctx context.Context, id int64) error
}

// SpaceMemberChecker verifies that a user belongs to a given space.
type SpaceMemberChecker interface {
	IsTeamspaceMember(ctx context.Context, teamID, userID int64) (bool, error)
	IsCommunityMember(ctx context.Context, communitySpaceID, userID int64) (bool, error)
}
