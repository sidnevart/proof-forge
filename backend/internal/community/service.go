package community

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Service contains the business logic for community space management.
type Service struct {
	repo Repository
}

// NewService constructs a Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput holds the parameters for creating a community space.
type CreateInput struct {
	OwnerUserID int64
	WorkspaceID *int64
	Name        string
	Slug        string
	Description string
	IsPublic    bool
}

// Create validates and persists a new CommunitySpace.
func (s *Service) Create(ctx context.Context, in CreateInput) (*CommunitySpace, error) {
	if err := ValidateName(in.Name); err != nil {
		return nil, err
	}

	slug := strings.TrimSpace(in.Slug)

	exists, err := s.repo.SlugExists(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("check slug: %w", err)
	}
	if exists {
		return nil, ErrSlugTaken
	}

	prefix := slug
	if len(prefix) > 2 {
		prefix = slug[:2]
	}
	code, err := GenerateInviteCode(prefix)
	if err != nil {
		return nil, fmt.Errorf("generate invite code: %w", err)
	}

	cs := &CommunitySpace{
		OwnerUserID: in.OwnerUserID,
		WorkspaceID: in.WorkspaceID,
		Name:        strings.TrimSpace(in.Name),
		Slug:        slug,
		Description: strings.TrimSpace(in.Description),
		InviteCode:  code,
		IsPublic:    in.IsPublic,
	}
	if err := s.repo.Create(ctx, cs); err != nil {
		return nil, fmt.Errorf("create community space: %w", err)
	}
	return cs, nil
}

// GetByID returns a community space by ID.
func (s *Service) GetByID(ctx context.Context, id int64) (*CommunitySpace, error) {
	return s.repo.GetByID(ctx, id)
}

// Join adds the given user as a member via invite code.
func (s *Service) Join(ctx context.Context, userID int64, inviteCode string) (*Membership, error) {
	cs, err := s.repo.GetByInviteCode(ctx, strings.TrimSpace(inviteCode))
	if err != nil {
		if errors.Is(err, ErrCommunityNotFound) {
			return nil, ErrInvalidInviteCode
		}
		return nil, fmt.Errorf("get by invite: %w", err)
	}

	existing, err := s.repo.GetMembership(ctx, cs.ID, userID)
	if err != nil && !errors.Is(err, ErrNotMember) {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if existing != nil && existing.Status == MemberStatusActive {
		return nil, ErrAlreadyMember
	}

	m := &Membership{
		CommunitySpaceID: cs.ID,
		UserID:           userID,
		Role:             RoleMember,
		Status:           MemberStatusActive,
	}
	if err := s.repo.AddMember(ctx, m); err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}
	return m, nil
}
