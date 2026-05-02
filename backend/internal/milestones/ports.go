package milestones

import (
	"context"
	"time"
)

type Repository interface {
	Insert(ctx context.Context, goalID int64, title, description string, sortOrder int) (Milestone, error)
	FindByGoal(ctx context.Context, goalID int64) ([]Milestone, error)
	FindByID(ctx context.Context, milestoneID int64) (Milestone, error)
	Update(ctx context.Context, milestoneID int64, title, description *string, sortOrder *int) (Milestone, error)
	Delete(ctx context.Context, milestoneID int64) error
	UpdateStatus(ctx context.Context, milestoneID int64, status MilestoneStatus, completedByUserID *int64, now time.Time) (Milestone, error)
	FindGoalParticipants(ctx context.Context, goalID int64) (ownerID, buddyID int64, goalStatus string, err error)
	NextSortOrder(ctx context.Context, goalID int64) (int, error)
}
