package recaps

import (
	"errors"
	"time"
)

type RecapStatus string

const (
	StatusPending    RecapStatus = "pending"
	StatusGenerating RecapStatus = "generating"
	StatusDone       RecapStatus = "done"
	StatusFailed     RecapStatus = "failed"
)

var (
	ErrRecapNotFound = errors.New("recap not found")
	ErrNotAuthorized = errors.New("not authorized to access this recap")
)

type WeeklyRecap struct {
	ID          int64       `json:"id"`
	GoalID      int64       `json:"goal_id"`
	OwnerUserID int64       `json:"owner_user_id"`
	PeriodStart time.Time   `json:"period_start"`
	PeriodEnd   time.Time   `json:"period_end"`
	Status      RecapStatus `json:"status"`
	SummaryText string      `json:"summary_text"`
	ModelName   string      `json:"model_name,omitempty"`
	GeneratedAt *time.Time  `json:"generated_at,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}

// GoalForRecap carries the goal context needed for recap generation.
type GoalForRecap struct {
	ID          int64
	OwnerUserID int64
	Title       string
	Description string
}

// ApprovedCheckIn is a read model for prompt construction.
// AI receives this data only to summarize — it must not make any approval decisions.
type ApprovedCheckIn struct {
	ID         int64
	ApprovedAt time.Time
	Evidence   []EvidenceSummary
}

type EvidenceSummary struct {
	Kind        string
	TextContent string
	ExternalURL string
}
