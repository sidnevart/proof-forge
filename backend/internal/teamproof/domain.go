package teamproof

import (
	"errors"
	"time"
)

// TeamProof is a check-in scoped to a team goal.
type TeamProof struct {
	ID           int64
	TeamID       int64
	GoalID       int64
	OwnerUserID  int64
	Status       string // submitted, approved, rejected
	ProofText    string
	SubmittedAt  *time.Time
	ApprovedAt   *time.Time
	RejectedAt   *time.Time
	ApproverRole *string // lead | trusted_approver
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// FeedItem is the privacy-aware read model for team feed.
type FeedItem struct {
	ID            int64
	OwnerUserID   int64
	OwnerAlias    string
	GoalTitle     string
	ProofText     string
	Status        string
	StreakCurrent int
	CommentsCount int
	EvidenceKinds []string
	SubmittedAt   time.Time
	CanApprove    bool // computed per-viewer
}

// Comment is a proof thread comment.
type Comment struct {
	ID        int64
	ProofID   int64
	UserID    int64
	UserAlias string
	Text      string
	CreatedAt time.Time
}

// SubmitInput is the payload for assembling a weekly proof.
type SubmitInput struct {
	TeamID          int64
	OwnerUserID     int64
	GoalID          int64
	DailyLogEntryIDs []int64
	ProofText       string
	Evidence        []EvidenceInput
}

type EvidenceInput struct {
	Kind       string `json:"kind"`
	ExternalURL string `json:"external_url,omitempty"`
	Label      string `json:"label,omitempty"`
}

// Domain errors.
var (
	ErrProofNotFound      = errors.New("proof not found")
	ErrNotAuthor          = errors.New("only the author can submit or edit")
	ErrNotApprover        = errors.New("only lead or trusted approver can review")
	ErrCannotReviewSelf   = errors.New("cannot review your own proof")
	ErrAlreadyReviewed    = errors.New("proof already reviewed")
	ErrNoArtifact         = errors.New("proof requires at least one artifact")
	ErrInvalidGoal        = errors.New("goal does not belong to this team or author")
	ErrEntriesNotOwned    = errors.New("daily log entries must belong to the author")
	ErrEntryAlreadyUsed   = errors.New("daily log entry already consumed in another proof")
	ErrRejectCommentRequired = errors.New("reject requires a comment")
	ErrCannotComment      = errors.New("cannot comment on this proof")
)

// IsTerminalStatus returns true for approved or rejected.
func IsTerminalStatus(s string) bool {
	return s == "approved" || s == "rejected"
}

// ArtifactRequired validates that at least one artifact exists.
func ArtifactRequired(evidence []EvidenceInput, hasConsumedEntries bool) error {
	if hasConsumedEntries {
		return nil
	}
	for _, e := range evidence {
		if e.Kind != "" {
			return nil
		}
	}
	return ErrNoArtifact
}
