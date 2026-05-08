package community

import "context"

// Repository is the storage abstraction for CommunitySpace persistence.
type Repository interface {
	Create(ctx context.Context, cs *CommunitySpace) error
	GetByID(ctx context.Context, id int64) (*CommunitySpace, error)
	GetByInviteCode(ctx context.Context, code string) (*CommunitySpace, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
	ListByWorkspace(ctx context.Context, workspaceID int64) ([]*CommunitySpace, error)

	// Membership operations.
	AddMember(ctx context.Context, m *Membership) error
	GetMembership(ctx context.Context, communityID, userID int64) (*Membership, error)
	UpdateMemberStatus(ctx context.Context, communityID, userID int64, status MemberStatus) error
	CountActiveMembers(ctx context.Context, communityID int64) (int64, error)
}
