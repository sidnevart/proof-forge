package contracts

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Service contains the business logic for proof contract management.
type Service struct {
	repo Repository
}

// NewService constructs a Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput holds the parameters for creating a proof contract.
type CreateInput struct {
	GoalID       int64
	UserID       int64
	BuddyUserID  *int64
	WhatToProve  string
	HowToProve   string
	DueAt        time.Time
}

// Create validates and persists a new ProofContract.
func (s *Service) Create(ctx context.Context, in CreateInput) (*ProofContract, error) {
	if err := ValidateWhatToProve(in.WhatToProve); err != nil {
		return nil, err
	}
	if in.DueAt.Before(time.Now()) {
		return nil, fmt.Errorf("%w: due_at must be in the future", ErrInvalidContract)
	}

	initialStatus := StatusPending
	if in.BuddyUserID == nil {
		// No buddy: immediately active.
		initialStatus = StatusActive
	}

	c := &ProofContract{
		GoalID:      in.GoalID,
		UserID:      in.UserID,
		BuddyUserID: in.BuddyUserID,
		WhatToProve: strings.TrimSpace(in.WhatToProve),
		HowToProve:  strings.TrimSpace(in.HowToProve),
		DueAt:       in.DueAt,
		Status:      initialStatus,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("create contract: %w", err)
	}
	return c, nil
}

// GetByID returns a contract by ID.
func (s *Service) GetByID(ctx context.Context, id int64) (*ProofContract, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByGoal returns all contracts for a goal.
func (s *Service) ListByGoal(ctx context.Context, goalID int64) ([]*ProofContract, error) {
	return s.repo.ListByGoal(ctx, goalID)
}

// ListActive returns active/pending contracts for the user.
func (s *Service) ListActive(ctx context.Context, userID int64) ([]*ProofContract, error) {
	active := StatusActive
	return s.repo.ListByUser(ctx, userID, &active)
}

// Activate transitions a pending contract to active (buddy accepted).
func (s *Service) Activate(ctx context.Context, id, actorID int64) error {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	// Only buddy may activate.
	if c.BuddyUserID == nil || *c.BuddyUserID != actorID {
		return ErrForbidden
	}
	if !c.CanTransitionTo(StatusActive) {
		return fmt.Errorf("%w: cannot activate from %s", ErrInvalidTransition, c.Status)
	}
	return s.repo.UpdateStatus(ctx, id, StatusActive)
}

// Fulfill transitions a contract to fulfilled (proof accepted).
func (s *Service) Fulfill(ctx context.Context, id, actorID int64) error {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	// Owner or buddy can mark fulfilled.
	if c.UserID != actorID && (c.BuddyUserID == nil || *c.BuddyUserID != actorID) {
		return ErrForbidden
	}
	if !c.CanTransitionTo(StatusFulfilled) {
		return fmt.Errorf("%w: cannot fulfill from %s", ErrInvalidTransition, c.Status)
	}
	return s.repo.UpdateStatus(ctx, id, StatusFulfilled)
}

// Cancel transitions a contract to cancelled.
func (s *Service) Cancel(ctx context.Context, id, actorID int64) error {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if c.UserID != actorID && (c.BuddyUserID == nil || *c.BuddyUserID != actorID) {
		return ErrForbidden
	}
	if !c.CanTransitionTo(StatusCancelled) {
		return fmt.Errorf("%w: cannot cancel from %s", ErrInvalidTransition, c.Status)
	}
	return s.repo.UpdateStatus(ctx, id, StatusCancelled)
}
