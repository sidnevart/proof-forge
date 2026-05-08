package workspaces

import (
	"context"
	"fmt"
	"strings"
)

// Service contains the business logic for workspace management.
type Service struct {
	repo Repository
}

// NewService constructs a Service backed by the given Repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput holds the parameters for creating a workspace.
type CreateInput struct {
	OwnerUserID int64
	Name        string
	Slug        string
	Type        string
}

// Create validates inputs and persists a new workspace.
func (s *Service) Create(ctx context.Context, in CreateInput) (*Workspace, error) {
	if err := ValidateName(in.Name); err != nil {
		return nil, err
	}

	slug := strings.TrimSpace(in.Slug)
	if err := ValidateSlug(slug); err != nil {
		return nil, err
	}

	wtype, err := ParseType(in.Type)
	if err != nil {
		return nil, err
	}

	exists, err := s.repo.SlugExists(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("check slug: %w", err)
	}
	if exists {
		return nil, ErrSlugTaken
	}

	w := &Workspace{
		OwnerUserID: in.OwnerUserID,
		Name:        strings.TrimSpace(in.Name),
		Slug:        slug,
		Type:        wtype,
		IsActive:    true,
	}
	if err := s.repo.Create(ctx, w); err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	return w, nil
}

// GetByID returns a workspace by ID.
func (s *Service) GetByID(ctx context.Context, id int64) (*Workspace, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByOwner returns all workspaces owned by a user.
func (s *Service) ListByOwner(ctx context.Context, ownerUserID int64) ([]*Workspace, error) {
	return s.repo.ListByOwner(ctx, ownerUserID)
}

// CheckSlugAvailable returns true when the slug is not yet taken.
func (s *Service) CheckSlugAvailable(ctx context.Context, slug string) (bool, error) {
	if err := ValidateSlug(slug); err != nil {
		return false, err
	}
	exists, err := s.repo.SlugExists(ctx, slug)
	return !exists, err
}

// Freeze freezes a workspace (platform admin action).
func (s *Service) Freeze(ctx context.Context, id int64, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("%w: freeze reason is required", ErrInvalidWorkspace)
	}
	return s.repo.Freeze(ctx, id, strings.TrimSpace(reason))
}

// Unfreeze unfreezes a workspace (platform admin action).
func (s *Service) Unfreeze(ctx context.Context, id int64) error {
	return s.repo.Unfreeze(ctx, id)
}

// ListAll returns paginated workspaces for the admin panel.
func (s *Service) ListAll(ctx context.Context, limit, offset int) ([]*Workspace, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListAll(ctx, limit, offset)
}
