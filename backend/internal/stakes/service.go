package stakes

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
	return &Service{
		repo:  repo,
		clock: time.Now,
	}
}

func (s *Service) Create(ctx context.Context, actor users.User, goalID int64, input CreateInput) (StakeView, error) {
	description := strings.TrimSpace(input.Description)
	if description == "" {
		return StakeView{}, errors.Join(ErrInvalidInput, errors.New("description is required"))
	}
	if len(description) > 1000 {
		return StakeView{}, errors.Join(ErrInvalidInput, errors.New("description must be 1000 characters or fewer"))
	}

	ownerID, _, goalStatus, err := s.repo.FindGoalParticipants(ctx, goalID)
	if err != nil {
		return StakeView{}, fmt.Errorf("find goal participants: %w", err)
	}
	if actor.ID != ownerID {
		return StakeView{}, ErrNotGoalOwner
	}
	if goalStatus != "active" {
		return StakeView{}, ErrGoalNotActive
	}

	stake, err := s.repo.Insert(ctx, goalID, actor.ID, description)
	if err != nil {
		return StakeView{}, fmt.Errorf("insert stake: %w", err)
	}
	return StakeView{Stake: stake}, nil
}

func (s *Service) ListForGoal(ctx context.Context, actor users.User, goalID int64) ([]StakeView, error) {
	ownerID, buddyID, _, err := s.repo.FindGoalParticipants(ctx, goalID)
	if err != nil {
		return nil, fmt.Errorf("find goal participants: %w", err)
	}
	if actor.ID != ownerID && actor.ID != buddyID {
		return nil, ErrNotGoalOwner
	}

	views, err := s.repo.FindByGoal(ctx, goalID)
	if err != nil {
		return nil, fmt.Errorf("list stakes: %w", err)
	}
	return views, nil
}

func (s *Service) Cancel(ctx context.Context, actor users.User, stakeID int64) error {
	stake, err := s.repo.FindByID(ctx, stakeID)
	if err != nil {
		return err
	}
	if stake.Status != StatusActive {
		return ErrStakeNotActive
	}

	ownerID, _, _, err := s.repo.FindGoalParticipants(ctx, stake.GoalID)
	if err != nil {
		return fmt.Errorf("find goal participants: %w", err)
	}
	if actor.ID != ownerID {
		return ErrNotGoalOwner
	}

	return s.repo.UpdateStatus(ctx, stakeID, StatusCancelled, s.clock().UTC())
}

func (s *Service) Forfeit(ctx context.Context, actor users.User, stakeID int64, input ForfeitInput) (StakeView, error) {
	stake, err := s.repo.FindByID(ctx, stakeID)
	if err != nil {
		return StakeView{}, err
	}
	if stake.Status != StatusActive {
		return StakeView{}, ErrStakeNotActive
	}

	_, buddyID, _, err := s.repo.FindGoalParticipants(ctx, stake.GoalID)
	if err != nil {
		return StakeView{}, fmt.Errorf("find goal participants: %w", err)
	}
	if actor.ID != buddyID {
		return StakeView{}, ErrNotBuddy
	}

	now := s.clock().UTC()
	if err := s.repo.UpdateStatus(ctx, stakeID, StatusForfeited, now); err != nil {
		return StakeView{}, fmt.Errorf("update stake status: %w", err)
	}

	reason := strings.TrimSpace(input.Reason)
	forfeiture, err := s.repo.InsertForfeiture(ctx, stakeID, actor.ID, reason)
	if err != nil {
		return StakeView{}, fmt.Errorf("insert forfeiture: %w", err)
	}

	stake.Status = StatusForfeited
	stake.ForfeitedAt = &now
	return StakeView{Stake: stake, Forfeiture: &forfeiture}, nil
}
