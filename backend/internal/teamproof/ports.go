package teamproof

import (
	"context"
	"time"
)

// Repository is the storage contract for team proofs.
type Repository interface {
	// Proof write path.
	CreateProof(ctx context.Context, p *TeamProof, evidence []EvidenceInput, consumedEntryIDs []int64) (*TeamProof, error)
	UpdateProofStatus(ctx context.Context, proofID int64, status string, approverRole string, now time.Time) error

	// Proof read path.
	GetProof(ctx context.Context, proofID int64) (*TeamProof, error)
	ListFeed(ctx context.Context, teamID int64, viewerUserID int64, role string, cursor int64, limit int) ([]FeedItem, error)

	// Comments.
	AddComment(ctx context.Context, proofID, userID int64, text string, now time.Time) (*Comment, error)
	ListComments(ctx context.Context, proofID int64) ([]Comment, error)

	// Validation helpers.
	GetMembership(ctx context.Context, teamID, userID int64) (MembershipInfo, error)
	GetGoalTeamOwner(ctx context.Context, goalID int64) (teamID int64, ownerUserID int64, err error)
	GetDailyLogEntries(ctx context.Context, entryIDs []int64, userID, teamID int64) ([]DailyLogEntryInfo, error)
	MarkEntriesConsumed(ctx context.Context, entryIDs []int64, proofID int64, now time.Time) error
}

// MembershipInfo mirrors the subset needed for authorization.
type MembershipInfo struct {
	Role     string
	Status   string
	Timezone string
}

// DailyLogEntryInfo is the read-only slice needed for proof assembly.
type DailyLogEntryInfo struct {
	ID        int64
	LogDate   time.Time
	Status    string
	HasArtifact bool
	ConsumedInCheckInID *int64
}
