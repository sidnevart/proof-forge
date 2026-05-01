package recaps

import (
	"context"
	"time"
)

type Repository interface {
	// FindGoalsNeedingRecap returns active goals with approved check-ins in [periodStart, periodEnd)
	// that do not yet have a done, generating, or pending recap for that period.
	FindGoalsNeedingRecap(ctx context.Context, periodStart, periodEnd time.Time) ([]GoalForRecap, error)

	// FindApprovedCheckIns returns approved check-ins with their evidence in [periodStart, periodEnd).
	FindApprovedCheckIns(ctx context.Context, goalID int64, periodStart, periodEnd time.Time) ([]ApprovedCheckIn, error)

	// InsertRecap creates a new recap record.
	InsertRecap(ctx context.Context, p InsertRecapParams) (WeeklyRecap, error)

	// UpdateRecap sets the final status, summary, model, and generated_at.
	UpdateRecap(ctx context.Context, p UpdateRecapParams) error

	// GetRecap returns the recap plus the goal's buddy user ID for permission checks.
	GetRecap(ctx context.Context, recapID int64) (WeeklyRecap, int64, error)

	// ListRecapsByGoal returns all recaps for a goal the actor can see (owner or buddy).
	ListRecapsByGoal(ctx context.Context, goalID, actorID int64) ([]WeeklyRecap, error)
}

// AIProvider generates a text summary from a prompt.
// AI must NOT approve, reject, or evaluate proof — it only synthesizes observational text.
type AIProvider interface {
	Summarize(ctx context.Context, prompt string) (summary, modelName string, err error)
}

type InsertRecapParams struct {
	GoalID      int64
	OwnerUserID int64
	PeriodStart time.Time
	PeriodEnd   time.Time
	Status      RecapStatus
}

type UpdateRecapParams struct {
	ID          int64
	Status      RecapStatus
	SummaryText string
	ModelName   string
	GeneratedAt time.Time
}
