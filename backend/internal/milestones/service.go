package milestones

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

type Service struct {
	repo  Repository
	clock func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, clock: time.Now}
}

func (s *Service) Create(ctx context.Context, actor users.User, goalID int64, input CreateInput) (Milestone, error) {
	if err := input.Validate(); err != nil {
		return Milestone{}, err
	}

	ownerID, _, goalStatus, err := s.repo.FindGoalParticipants(ctx, goalID)
	if err != nil {
		return Milestone{}, fmt.Errorf("find goal participants: %w", err)
	}
	if actor.ID != ownerID {
		return Milestone{}, ErrNotGoalOwner
	}
	if goalStatus != "active" {
		return Milestone{}, ErrGoalNotActive
	}

	sortOrder, err := s.repo.NextSortOrder(ctx, goalID)
	if err != nil {
		return Milestone{}, fmt.Errorf("next sort order: %w", err)
	}

	m, err := s.repo.Insert(ctx, goalID, strings.TrimSpace(input.Title), strings.TrimSpace(input.Description), sortOrder)
	if err != nil {
		return Milestone{}, fmt.Errorf("insert milestone: %w", err)
	}
	return m, nil
}

func (s *Service) ListForGoal(ctx context.Context, actor users.User, goalID int64) ([]Milestone, error) {
	ownerID, buddyID, _, err := s.repo.FindGoalParticipants(ctx, goalID)
	if err != nil {
		return nil, fmt.Errorf("find goal participants: %w", err)
	}
	if actor.ID != ownerID && actor.ID != buddyID {
		return nil, ErrNotGoalOwner
	}
	return s.repo.FindByGoal(ctx, goalID)
}

func (s *Service) Update(ctx context.Context, actor users.User, milestoneID int64, input UpdateInput) (Milestone, error) {
	m, err := s.repo.FindByID(ctx, milestoneID)
	if err != nil {
		return Milestone{}, err
	}
	if m.Status != StatusPending {
		return Milestone{}, ErrMilestoneNotPending
	}

	ownerID, _, _, err := s.repo.FindGoalParticipants(ctx, m.GoalID)
	if err != nil {
		return Milestone{}, fmt.Errorf("find goal participants: %w", err)
	}
	if actor.ID != ownerID {
		return Milestone{}, ErrNotGoalOwner
	}

	var titlePtr, descPtr *string
	if input.Title != nil {
		t := strings.TrimSpace(*input.Title)
		if t == "" {
			return Milestone{}, errors.Join(ErrInvalidInput, errors.New("title cannot be empty"))
		}
		if len(t) > 200 {
			return Milestone{}, errors.Join(ErrInvalidInput, errors.New("title too long"))
		}
		titlePtr = &t
	}
	if input.Description != nil {
		d := strings.TrimSpace(*input.Description)
		if len(d) > 2000 {
			return Milestone{}, errors.Join(ErrInvalidInput, errors.New("description too long"))
		}
		descPtr = &d
	}

	return s.repo.Update(ctx, milestoneID, titlePtr, descPtr, input.SortOrder)
}

func (s *Service) Delete(ctx context.Context, actor users.User, milestoneID int64) error {
	m, err := s.repo.FindByID(ctx, milestoneID)
	if err != nil {
		return err
	}
	if m.Status != StatusPending {
		return ErrMilestoneNotPending
	}

	ownerID, _, _, err := s.repo.FindGoalParticipants(ctx, m.GoalID)
	if err != nil {
		return fmt.Errorf("find goal participants: %w", err)
	}
	if actor.ID != ownerID {
		return ErrNotGoalOwner
	}

	return s.repo.Delete(ctx, milestoneID)
}

func (s *Service) Complete(ctx context.Context, actor users.User, milestoneID int64) (Milestone, error) {
	m, err := s.repo.FindByID(ctx, milestoneID)
	if err != nil {
		return Milestone{}, err
	}
	if m.Status != StatusPending {
		return Milestone{}, ErrMilestoneNotPending
	}

	_, buddyID, _, err := s.repo.FindGoalParticipants(ctx, m.GoalID)
	if err != nil {
		return Milestone{}, fmt.Errorf("find goal participants: %w", err)
	}
	if actor.ID != buddyID {
		return Milestone{}, ErrNotBuddy
	}

	actorID := actor.ID
	return s.repo.UpdateStatus(ctx, milestoneID, StatusCompleted, &actorID, s.clock().UTC())
}

func (s *Service) Reopen(ctx context.Context, actor users.User, milestoneID int64) (Milestone, error) {
	m, err := s.repo.FindByID(ctx, milestoneID)
	if err != nil {
		return Milestone{}, err
	}
	if m.Status != StatusCompleted {
		return Milestone{}, ErrMilestoneNotComplete
	}

	_, buddyID, _, err := s.repo.FindGoalParticipants(ctx, m.GoalID)
	if err != nil {
		return Milestone{}, fmt.Errorf("find goal participants: %w", err)
	}
	if actor.ID != buddyID {
		return Milestone{}, ErrNotBuddy
	}

	return s.repo.UpdateStatus(ctx, milestoneID, StatusPending, nil, s.clock().UTC())
}
