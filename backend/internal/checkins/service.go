package checkins

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

type Service struct {
	repo    Repository
	storage Storage
	clock   func() time.Time
}

func NewService(repo Repository, storage Storage) *Service {
	return &Service{
		repo:    repo,
		storage: storage,
		clock:   time.Now,
	}
}

func (s *Service) CreateCheckIn(ctx context.Context, actor users.User, goalID int64) (CheckIn, error) {
	ci, err := s.repo.CreateCheckIn(ctx, CreateCheckInParams{
		GoalID:      goalID,
		OwnerUserID: actor.ID,
		Status:      StatusDraft,
	})
	if err != nil {
		return CheckIn{}, fmt.Errorf("create check-in: %w", err)
	}
	return ci, nil
}

func (s *Service) GetDetail(ctx context.Context, actor users.User, checkInID int64) (CheckInView, error) {
	view, err := s.repo.GetCheckIn(ctx, checkInID)
	if err != nil {
		return CheckInView{}, err
	}
	if actor.ID != view.CheckIn.OwnerUserID && actor.ID != view.BuddyUserID {
		return CheckInView{}, ErrNotAuthorized
	}
	return view, nil
}

func (s *Service) ListForGoal(ctx context.Context, actor users.User, goalID int64) ([]CheckIn, error) {
	list, err := s.repo.ListCheckInsByGoal(ctx, goalID, actor.ID)
	if err != nil {
		return nil, fmt.Errorf("list check-ins: %w", err)
	}
	return list, nil
}

func (s *Service) Submit(ctx context.Context, actor users.User, checkInID int64) error {
	view, err := s.repo.GetCheckIn(ctx, checkInID)
	if err != nil {
		return err
	}
	if actor.ID != view.CheckIn.OwnerUserID {
		return ErrNotOwner
	}
	if !canSubmit(view.CheckIn.Status) {
		return ErrCannotSubmit
	}

	now := s.clock().UTC()
	return s.repo.UpdateCheckInStatus(ctx, UpdateStatusParams{
		ID:          checkInID,
		Status:      StatusSubmitted,
		SubmittedAt: &now,
		Now:         now,
	})
}

func (s *Service) AddTextEvidence(ctx context.Context, actor users.User, checkInID int64, input AddTextInput) (EvidenceItem, error) {
	if err := input.Validate(); err != nil {
		return EvidenceItem{}, err
	}
	view, err := s.requireOwnerEditable(ctx, actor, checkInID)
	if err != nil {
		return EvidenceItem{}, err
	}
	if err := s.checkEvidenceCap(ctx, view); err != nil {
		return EvidenceItem{}, err
	}

	item, err := s.repo.InsertEvidence(ctx, InsertEvidenceParams{
		CheckInID:   checkInID,
		Kind:        KindText,
		TextContent: strings.TrimSpace(input.Content),
	})
	if err != nil {
		return EvidenceItem{}, fmt.Errorf("insert text evidence: %w", err)
	}
	return item, nil
}

func (s *Service) AddLinkEvidence(ctx context.Context, actor users.User, checkInID int64, input AddLinkInput) (EvidenceItem, error) {
	if err := input.Validate(); err != nil {
		return EvidenceItem{}, err
	}
	view, err := s.requireOwnerEditable(ctx, actor, checkInID)
	if err != nil {
		return EvidenceItem{}, err
	}
	if err := s.checkEvidenceCap(ctx, view); err != nil {
		return EvidenceItem{}, err
	}

	item, err := s.repo.InsertEvidence(ctx, InsertEvidenceParams{
		CheckInID:   checkInID,
		Kind:        KindLink,
		ExternalURL: SanitizeURL(input.URL),
	})
	if err != nil {
		return EvidenceItem{}, fmt.Errorf("insert link evidence: %w", err)
	}
	return item, nil
}

func (s *Service) AddFileEvidence(ctx context.Context, actor users.User, checkInID int64, data []byte, clientMIME string) (EvidenceItem, error) {
	if int64(len(data)) > MaxFileSizeBytes {
		return EvidenceItem{}, ErrFileTooLarge
	}
	// Validate from file content, ignoring the client-supplied Content-Type.
	mimeType := DetectMIME(data, clientMIME)
	kind, ok := mimeToKind(mimeType)
	if !ok {
		return EvidenceItem{}, ErrUnsupportedMIME
	}

	view, err := s.requireOwnerEditable(ctx, actor, checkInID)
	if err != nil {
		return EvidenceItem{}, err
	}
	if err := s.checkEvidenceCap(ctx, view); err != nil {
		return EvidenceItem{}, err
	}

	ext := fileExtension(mimeType)
	key := s.storage.ObjectKey(checkInID, ext)
	if err := s.storage.Put(ctx, key, bytes.NewReader(data), int64(len(data)), mimeType); err != nil {
		return EvidenceItem{}, fmt.Errorf("upload file: %w", err)
	}

	item, err := s.repo.InsertEvidence(ctx, InsertEvidenceParams{
		CheckInID:     checkInID,
		Kind:          kind,
		StorageKey:    key,
		MIMEType:      mimeType,
		FileSizeBytes: int64(len(data)),
	})
	if err != nil {
		return EvidenceItem{}, fmt.Errorf("insert file evidence: %w", err)
	}
	return item, nil
}

func (s *Service) Review(ctx context.Context, actor users.User, checkInID int64, input ReviewInput) (ReviewRecord, error) {
	view, err := s.repo.GetCheckIn(ctx, checkInID)
	if err != nil {
		return ReviewRecord{}, err
	}
	if actor.ID != view.BuddyUserID {
		return ReviewRecord{}, ErrNotBuddy
	}
	if view.CheckIn.Status != StatusSubmitted {
		return ReviewRecord{}, ErrCannotReview
	}

	now := s.clock().UTC()
	return s.repo.RecordReview(ctx, RecordReviewParams{
		CheckInID:      checkInID,
		GoalID:         view.CheckIn.GoalID,
		ReviewerUserID: actor.ID,
		Decision:       input.Decision,
		Comment:        strings.TrimSpace(input.Comment),
		Now:            now,
	})
}

func (s *Service) requireOwnerEditable(ctx context.Context, actor users.User, checkInID int64) (CheckInView, error) {
	view, err := s.repo.GetCheckIn(ctx, checkInID)
	if err != nil {
		return CheckInView{}, err
	}
	if actor.ID != view.CheckIn.OwnerUserID {
		return CheckInView{}, ErrNotOwner
	}
	if !canAddEvidence(view.CheckIn.Status) {
		return CheckInView{}, ErrCannotAddEvidence
	}
	return view, nil
}

func (s *Service) checkEvidenceCap(ctx context.Context, view CheckInView) error {
	count, err := s.repo.CountEvidence(ctx, view.CheckIn.ID)
	if err != nil {
		return fmt.Errorf("count evidence: %w", err)
	}
	if count >= MaxEvidenceItems {
		return ErrTooManyEvidenceItems
	}
	return nil
}
