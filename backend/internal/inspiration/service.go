package inspiration

import (
	"context"
	"strings"
)

type Repository interface {
	ListPublicGoals(ctx context.Context, p FeedParams) ([]PublicGoal, error)
	ListPublicProofs(ctx context.Context, p FeedParams) ([]PublicProof, error)
	ListCheckInsForUserCircles(ctx context.Context, actorID int64, p CircleFeedParams) ([]CircleFeedItem, error)
	SetGoalVisibility(ctx context.Context, goalID, ownerID int64, isPublic bool) error
	SetCheckInVisibility(ctx context.Context, checkInID, ownerID int64, isPublic bool, attachmentIDs []int64) error
	UpdateSharingPrefs(ctx context.Context, userID int64, in SharingPrefsInput) error
	CreateReport(ctx context.Context, reporterID int64, in ReportInput) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListCirclesFeed(ctx context.Context, actorID int64, p CircleFeedParams) ([]CircleFeedItem, error) {
	return s.repo.ListCheckInsForUserCircles(ctx, actorID, p)
}

func (s *Service) ListTemplates(ctx context.Context, p FeedParams) ([]PublicGoal, error) {
	return s.repo.ListPublicGoals(ctx, p)
}

func (s *Service) ListProofs(ctx context.Context, p FeedParams) ([]PublicProof, error) {
	return s.repo.ListPublicProofs(ctx, p)
}

func (s *Service) SetGoalVisibility(ctx context.Context, goalID, actorID int64, in VisibilityInput) error {
	return s.repo.SetGoalVisibility(ctx, goalID, actorID, in.IsPublic)
}

func (s *Service) SetCheckInVisibility(ctx context.Context, checkInID, actorID int64, in VisibilityInput) error {
	return s.repo.SetCheckInVisibility(ctx, checkInID, actorID, in.IsPublic, in.PublicAttachmentIDs)
}

func (s *Service) UpdateSharingPrefs(ctx context.Context, userID int64, in SharingPrefsInput) error {
	in.Alias = strings.TrimSpace(in.Alias)
	return s.repo.UpdateSharingPrefs(ctx, userID, in)
}

func (s *Service) CreateReport(ctx context.Context, reporterID int64, in ReportInput) error {
	if in.Kind != "goal" && in.Kind != "checkin" {
		return ErrInvalidReportInput
	}
	if in.ID <= 0 {
		return ErrInvalidReportInput
	}
	return s.repo.CreateReport(ctx, reporterID, in)
}
