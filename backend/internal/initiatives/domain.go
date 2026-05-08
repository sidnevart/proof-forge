package initiatives

import (
	"errors"
	"strings"
	"time"
)

type InitiativeStatus string

const (
	StatusActive   InitiativeStatus = "active"
	StatusArchived InitiativeStatus = "archived"
)

var (
	ErrInitiativeNotFound  = errors.New("initiative not found")
	ErrNotSpaceMember      = errors.New("user is not a member of this space")
	ErrAlreadyJoined       = errors.New("already joined this initiative")
	ErrCannotApproveSelf   = errors.New("cannot approve your own proof")
	ErrNotInitiativeMember = errors.New("user is not a participant of this initiative")
	ErrInitiativeArchived  = errors.New("initiative is archived")
	ErrCheckinNotFound     = errors.New("checkin not found or already approved")
	ErrInvalidInput        = errors.New("invalid initiative input")
)

type Initiative struct {
	ID                int64            `json:"id"`
	SpaceType         string           `json:"space_type"`
	TeamspaceID       *int64           `json:"teamspace_id,omitempty"`
	CommunitySpaceID  *int64           `json:"community_space_id,omitempty"`
	CreatorID         int64            `json:"creator_id"`
	Title             string           `json:"title"`
	Description       string           `json:"description"`
	ProofCriteria     string           `json:"proof_criteria"`
	Status            InitiativeStatus `json:"status"`
	ParticipantCount  int              `json:"participant_count"`
	ActiveToday       int              `json:"active_today"`
	PendingProofCount int              `json:"pending_proof_count"`
	CreatedAt         time.Time        `json:"created_at"`
}

type CreateInput struct {
	SpaceType        string `json:"space_type"`
	SpaceID          int64  `json:"-"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	ProofCriteria    string `json:"proof_criteria"`
}

func (in CreateInput) Validate() error {
	title := strings.TrimSpace(in.Title)
	if len(title) < 3 || len(title) > 120 {
		return errors.Join(ErrInvalidInput, errors.New("title must be 3–120 characters"))
	}
	if len(strings.TrimSpace(in.ProofCriteria)) < 10 {
		return errors.Join(ErrInvalidInput, errors.New("proof_criteria must be at least 10 characters"))
	}
	if in.SpaceType != "teamspace" && in.SpaceType != "community" {
		return errors.Join(ErrInvalidInput, errors.New("space_type must be teamspace or community"))
	}
	return nil
}

type JoinResult struct {
	GoalID  int64 `json:"goal_id"`
	Created bool  `json:"created"`
}

type PendingProof struct {
	CheckinID   int64     `json:"checkin_id"`
	AuthorID    int64     `json:"author_id"`
	AuthorName  string    `json:"author_name"`
	Content     string    `json:"content"`
	SubmittedAt time.Time `json:"submitted_at"`
}

type ParticipantProgress struct {
	UserID      int64  `json:"user_id"`
	DisplayName string `json:"display_name"`
	ProofCount  int    `json:"proof_count"`
}

type Detail struct {
	Initiative   Initiative            `json:"initiative"`
	Participants []ParticipantProgress `json:"participants"`
	PendingCount int                   `json:"pending_count"`
}
