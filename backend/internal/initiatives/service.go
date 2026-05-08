package initiatives

import (
	"context"
	"errors"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

type Service struct {
	repo    Repository
	members SpaceMemberChecker
}

func NewService(repo Repository, members SpaceMemberChecker) *Service {
	return &Service{repo: repo, members: members}
}

func (s *Service) Create(ctx context.Context, actor users.User, input CreateInput) (Initiative, error) {
	if err := input.Validate(); err != nil {
		return Initiative{}, err
	}
	if err := s.checkSpaceMembership(ctx, input.SpaceType, input.SpaceID, actor.ID); err != nil {
		return Initiative{}, err
	}
	return s.repo.Create(ctx, input, actor.ID)
}

func (s *Service) List(ctx context.Context, spaceType string, spaceID int64) ([]Initiative, error) {
	return s.repo.ListBySpace(ctx, spaceType, spaceID)
}

func (s *Service) GetDetail(ctx context.Context, id int64) (Detail, error) {
	initiative, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	participants, err := s.repo.Participants(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	return Detail{
		Initiative:   initiative,
		Participants: participants,
		PendingCount: initiative.PendingProofCount,
	}, nil
}

func (s *Service) Join(ctx context.Context, actor users.User, initiativeID int64) (JoinResult, error) {
	initiative, err := s.repo.FindByID(ctx, initiativeID)
	if err != nil {
		return JoinResult{}, err
	}
	if initiative.Status == StatusArchived {
		return JoinResult{}, ErrInitiativeArchived
	}
	if err := s.checkSpaceMembershipForInitiative(ctx, initiative, actor.ID); err != nil {
		return JoinResult{}, err
	}
	return s.repo.JoinOrGet(ctx, initiativeID, actor.ID)
}

func (s *Service) PendingProofs(ctx context.Context, actor users.User, initiativeID int64) ([]PendingProof, error) {
	ok, err := s.repo.IsParticipant(ctx, initiativeID, actor.ID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotInitiativeMember
	}
	return s.repo.PendingProofs(ctx, initiativeID, actor.ID)
}

func (s *Service) Approve(ctx context.Context, actor users.User, initiativeID, checkinID int64, comment string) error {
	ok, err := s.repo.IsParticipant(ctx, initiativeID, actor.ID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotInitiativeMember
	}

	authorID, err := s.repo.CheckinAuthorID(ctx, checkinID)
	if err != nil {
		return err
	}
	if authorID == actor.ID {
		return ErrCannotApproveSelf
	}

	return s.repo.Approve(ctx, checkinID, actor.ID, comment)
}

func (s *Service) checkSpaceMembership(ctx context.Context, spaceType string, spaceID, userID int64) error {
	switch spaceType {
	case "teamspace":
		ok, err := s.members.IsTeamspaceMember(ctx, spaceID, userID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotSpaceMember
		}
	case "community":
		ok, err := s.members.IsCommunityMember(ctx, spaceID, userID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotSpaceMember
		}
	}
	return nil
}

func (s *Service) checkSpaceMembershipForInitiative(ctx context.Context, initiative Initiative, userID int64) error {
	switch initiative.SpaceType {
	case "teamspace":
		if initiative.TeamspaceID == nil {
			return errors.New("initiative has no teamspace_id")
		}
		return s.checkSpaceMembership(ctx, "teamspace", *initiative.TeamspaceID, userID)
	case "community":
		if initiative.CommunitySpaceID == nil {
			return errors.New("initiative has no community_space_id")
		}
		return s.checkSpaceMembership(ctx, "community", *initiative.CommunitySpaceID, userID)
	}
	return nil
}
