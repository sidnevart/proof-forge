package stakes

import (
	"context"
	"time"
)

type Repository interface {
	Insert(ctx context.Context, goalID, ownerUserID int64, description string) (Stake, error)
	FindByGoal(ctx context.Context, goalID int64) ([]StakeView, error)
	FindByID(ctx context.Context, stakeID int64) (Stake, error)
	UpdateStatus(ctx context.Context, stakeID int64, status StakeStatus, now time.Time) error
	InsertForfeiture(ctx context.Context, stakeID, declaredByUserID int64, reason string) (StakeForfeiture, error)
	FindGoalParticipants(ctx context.Context, goalID int64) (ownerID, buddyID int64, goalStatus string, err error)
}
