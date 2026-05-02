package stakes

import (
	"errors"
	"time"
)

type StakeStatus string

const (
	StatusActive    StakeStatus = "active"
	StatusForfeited StakeStatus = "forfeited"
	StatusCompleted StakeStatus = "completed"
	StatusCancelled StakeStatus = "cancelled"
)

var (
	ErrStakeNotFound  = errors.New("stake not found")
	ErrNotGoalOwner   = errors.New("only the goal owner can manage stakes")
	ErrNotBuddy       = errors.New("only the goal buddy can forfeit a stake")
	ErrStakeNotActive = errors.New("stake is not active")
	ErrGoalNotActive  = errors.New("goal is not active")
	ErrInvalidInput   = errors.New("invalid stake input")
)

type Stake struct {
	ID          int64       `json:"id"`
	GoalID      int64       `json:"goal_id"`
	OwnerUserID int64       `json:"owner_user_id"`
	Description string      `json:"description"`
	Status      StakeStatus `json:"status"`
	ForfeitedAt *time.Time  `json:"forfeited_at,omitempty"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	CancelledAt *time.Time  `json:"cancelled_at,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type StakeForfeiture struct {
	ID               int64     `json:"id"`
	StakeID          int64     `json:"stake_id"`
	DeclaredByUserID int64     `json:"declared_by_user_id"`
	Reason           string    `json:"reason"`
	CreatedAt        time.Time `json:"created_at"`
}

type StakeView struct {
	Stake      Stake            `json:"stake"`
	Forfeiture *StakeForfeiture `json:"forfeiture,omitempty"`
}

type CreateInput struct {
	Description string `json:"description"`
}

type ForfeitInput struct {
	Reason string `json:"reason"`
}
