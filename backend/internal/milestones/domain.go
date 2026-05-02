package milestones

import (
	"errors"
	"strings"
	"time"
)

type MilestoneStatus string

const (
	StatusPending   MilestoneStatus = "pending"
	StatusCompleted MilestoneStatus = "completed"
)

var (
	ErrMilestoneNotFound    = errors.New("milestone not found")
	ErrNotGoalOwner         = errors.New("only the goal owner can manage milestones")
	ErrNotBuddy             = errors.New("only the goal buddy can complete milestones")
	ErrMilestoneNotPending  = errors.New("milestone is not pending")
	ErrMilestoneNotComplete = errors.New("milestone is not completed")
	ErrGoalNotActive        = errors.New("goal is not active")
	ErrInvalidInput         = errors.New("invalid milestone input")
)

type Milestone struct {
	ID                int64           `json:"id"`
	GoalID            int64           `json:"goal_id"`
	Title             string          `json:"title"`
	Description       string          `json:"description"`
	Status            MilestoneStatus `json:"status"`
	SortOrder         int             `json:"sort_order"`
	CompletedAt       *time.Time      `json:"completed_at,omitempty"`
	CompletedByUserID *int64          `json:"completed_by_user_id,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type CreateInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateInput struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

func (in CreateInput) Validate() error {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return errors.Join(ErrInvalidInput, errors.New("title is required"))
	}
	if len(title) > 200 {
		return errors.Join(ErrInvalidInput, errors.New("title must be 200 characters or fewer"))
	}
	if len(in.Description) > 2000 {
		return errors.Join(ErrInvalidInput, errors.New("description must be 2000 characters or fewer"))
	}
	return nil
}
