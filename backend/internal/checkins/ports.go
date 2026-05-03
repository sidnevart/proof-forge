package checkins

import (
	"context"
	"io"
	"time"
)

type Repository interface {
	// CreateCheckIn inserts a draft check-in only when the goal is active and
	// the actor is the goal owner. Returns ErrGoalNotEligible otherwise.
	CreateCheckIn(ctx context.Context, params CreateCheckInParams) (CheckIn, error)

	// SetCheckInDeadline updates the deadline column on a check-in owned by
	// ownerUserID. Pass nil to clear. Returns ErrCheckInNotFound otherwise.
	SetCheckInDeadline(ctx context.Context, checkInID, ownerUserID int64, deadline *time.Time) error

	// GetCheckIn fetches a check-in and all its evidence plus the goal's
	// buddy_user_id so the service can enforce read permissions.
	GetCheckIn(ctx context.Context, checkInID int64) (CheckInView, error)

	// ListCheckInsByGoal returns check-ins visible to the given actor (owner OR buddy).
	ListCheckInsByGoal(ctx context.Context, goalID int64, actorID int64) ([]CheckIn, error)

	// UpdateCheckInStatus transitions status and sets the corresponding timestamp.
	UpdateCheckInStatus(ctx context.Context, params UpdateStatusParams) error

	// InsertEvidence appends one evidence item to a check-in.
	InsertEvidence(ctx context.Context, params InsertEvidenceParams) (EvidenceItem, error)

	// CountEvidence returns the current evidence item count for a check-in.
	CountEvidence(ctx context.Context, checkInID int64) (int, error)

	// RecordReview atomically inserts a review record, transitions the check-in
	// status, and updates goal progress (streak + health) when approved or rejected.
	RecordReview(ctx context.Context, params RecordReviewParams) (ReviewRecord, error)

	// DeleteCheckIn removes the check-in and cascades to evidence + reviews.
	DeleteCheckIn(ctx context.Context, checkInID int64) error
}

// Storage abstracts object storage so the checkins service stays
// independent of the underlying provider (S3, MinIO, local FS).
type Storage interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, mimeType string) error
	ObjectKey(checkInID int64, ext string) string
}

type CreateCheckInParams struct {
	GoalID      int64
	OwnerUserID int64
	Status      CheckInStatus
	DeadlineAt  *time.Time
}

type UpdateStatusParams struct {
	ID                 int64
	Status             CheckInStatus
	SubmittedAt        *time.Time
	ApprovedAt         *time.Time
	RejectedAt         *time.Time
	ChangesRequestedAt *time.Time
	Now                time.Time
}

type RecordReviewParams struct {
	CheckInID      int64
	GoalID         int64
	ReviewerUserID int64
	Decision       ReviewDecision
	Comment        string
	Now            time.Time
}

type InsertEvidenceParams struct {
	CheckInID     int64
	Kind          EvidenceKind
	TextContent   string
	ExternalURL   string
	StorageKey    string
	MIMEType      string
	FileSizeBytes int64
}
