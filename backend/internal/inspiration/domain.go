package inspiration

import (
	"errors"
	"time"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrNotAuthorized      = errors.New("not authorized")
	ErrAlreadyPublic      = errors.New("already public")
	ErrInvalidReportInput = errors.New("invalid report input")
)

// PublicGoal is the safe public view of a goal template.
type PublicGoal struct {
	ID            int64    `json:"id"`
	Title         string   `json:"title"`
	Smart         string   `json:"smart"`
	ProofExamples string   `json:"proof_examples"`
	Category      string   `json:"category"`
	AuthorAlias   string   `json:"author_alias"`
	CreatedAt     time.Time `json:"created_at"`
}

// PublicProof is the safe public view of a check-in proof.
type PublicProof struct {
	ID          int64     `json:"id"`
	GoalTitle   string    `json:"goal_title"`
	Category    string    `json:"category"`
	AuthorAlias string    `json:"author_alias"`
	Streak      int       `json:"streak"`
	TextContent string    `json:"text_content,omitempty"`
	ExternalURL string    `json:"external_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type FeedParams struct {
	Query           string
	Category        string
	SimilarToGoalID int64
	Cursor          int64
	Limit           int
}

type VisibilityInput struct {
	IsPublic              bool    `json:"is_public"`
	PublicAttachmentIDs   []int64 `json:"public_attachment_ids,omitempty"`
}

type SharingPrefsInput struct {
	ShareDefault   bool   `json:"share_default"`
	IsAnonymous    bool   `json:"is_anonymous"`
	Alias          string `json:"alias"`
}

type ReportInput struct {
	Kind   string `json:"kind"`
	ID     int64  `json:"id"`
	Reason string `json:"reason"`
}

// CircleFeedItem is one proof entry in the authenticated circle feed.
type CircleFeedItem struct {
	ID          int64     `json:"id"`
	GoalID      int64     `json:"goal_id"`
	GoalTitle   string    `json:"goal_title"`
	Category    string    `json:"category"`
	CircleID    int64     `json:"circle_id"`
	CircleName  string    `json:"circle_name"`
	AuthorAlias string    `json:"author_alias"`
	OwnerUserID int64     `json:"-"` // used to compute can_approve, not exposed
	TextContent string    `json:"text_content,omitempty"`
	ExternalURL string    `json:"external_url,omitempty"`
	SubmittedAt time.Time `json:"submitted_at"`
	Status      string    `json:"status"` // "submitted" | "approved"
	CanApprove  bool      `json:"can_approve"`
	Streak      int       `json:"streak"`
}

// CircleFeedParams holds pagination params for the authenticated circles feed.
type CircleFeedParams struct {
	Cursor int64
	Limit  int
}
