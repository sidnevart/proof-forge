package checkins

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/analytics"
	"github.com/sidnevart/proof-forge/backend/internal/users"
)

type Service struct {
	repo       Repository
	storage    Storage
	emitter    DomainEventEmitter
	membership MembershipChecker
	tracker    analytics.Tracker
	clock      func() time.Time
}

// NoopMembershipChecker satisfies MembershipChecker without doing anything.
// Used when circles service is not wired (tests, etc).
type NoopMembershipChecker struct{}

func (NoopMembershipChecker) IsCircleMemberForGoal(_ context.Context, _, _ int64) (bool, error) {
	return false, nil
}

func NewService(repo Repository, storage Storage, emitter ...DomainEventEmitter) *Service {
	var em DomainEventEmitter = NoopEmitter{}
	if len(emitter) > 0 && emitter[0] != nil {
		em = emitter[0]
	}
	return &Service{
		repo:       repo,
		storage:    storage,
		emitter:    em,
		membership: NoopMembershipChecker{},
		tracker:    analytics.NoopRecorder{},
		clock:      time.Now,
	}
}

// WithTracker injects a pilot analytics tracker into the service.
func (s *Service) WithTracker(t analytics.Tracker) *Service {
	s.tracker = t
	return s
}

// WithMembershipChecker injects a circle membership checker so the Review()
// method can authorise any circle member (Round-A pivot).
func (s *Service) WithMembershipChecker(m MembershipChecker) *Service {
	s.membership = m
	return s
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
	// Owner or legacy buddy always have access.
	if actor.ID == view.CheckIn.OwnerUserID || actor.ID == view.BuddyUserID {
		return view, nil
	}
	// Round-A: any active circle member can view the check-in.
	isMember, err := s.membership.IsCircleMemberForGoal(ctx, actor.ID, view.CheckIn.GoalID)
	if err != nil {
		return CheckInView{}, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
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
	if err := s.repo.UpdateCheckInStatus(ctx, UpdateStatusParams{
		ID:          checkInID,
		Status:      StatusSubmitted,
		SubmittedAt: &now,
		Now:         now,
	}); err != nil {
		return err
	}

	ownerName := actor.DisplayName
	if ownerName == "" {
		ownerName = actor.Email
	}
	_ = s.emitter.Emit(ctx, "checkin.submitted", map[string]any{
		"check_in_id":        checkInID,
		"owner_user_id":      actor.ID,
		"owner_display_name": ownerName,
		"goal_id":            view.CheckIn.GoalID,
	})

	goalID := view.CheckIn.GoalID
	s.tracker.TrackAsync(analytics.PilotEvent{
		UserID: actor.ID,
		GoalID: &goalID,
		Name:   analytics.EventWeeklyProofSubmitted,
		Props:  map[string]any{"goal_id": goalID},
	})

	// First-proof event fires exactly once per user lifecycle.
	if count, err := s.repo.CountSubmittedByUser(ctx, actor.ID); err == nil && count == 1 {
		s.tracker.TrackAsync(analytics.PilotEvent{
			UserID: actor.ID,
			GoalID: &goalID,
			Name:   analytics.EventFirstProofSubmitted,
			Props:  map[string]any{"goal_id": goalID},
		})
	}

	return nil
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

	// Cannot review your own check-in.
	if actor.ID == view.CheckIn.OwnerUserID {
		return ReviewRecord{}, ErrCannotReviewOwn
	}

	// Round-A pivot: any active circle member may review, not just the buddy.
	// Fallback: legacy buddy path still works if membership check fails.
	if actor.ID != view.BuddyUserID {
		isMember, err := s.membership.IsCircleMemberForGoal(ctx, actor.ID, view.CheckIn.GoalID)
		if err != nil {
			return ReviewRecord{}, fmt.Errorf("check membership: %w", err)
		}
		if !isMember {
			return ReviewRecord{}, ErrNotAuthorized
		}
	}

	if view.CheckIn.Status != StatusSubmitted {
		return ReviewRecord{}, ErrCannotReview
	}

	now := s.clock().UTC()
	record, err := s.repo.RecordReview(ctx, RecordReviewParams{
		CheckInID:      checkInID,
		GoalID:         view.CheckIn.GoalID,
		ReviewerUserID: actor.ID,
		Decision:       input.Decision,
		Comment:        strings.TrimSpace(input.Comment),
		Now:            now,
	})
	if err != nil {
		return ReviewRecord{}, err
	}

	eventKind := "checkin.approved"
	if input.Decision == DecisionReject {
		eventKind = "checkin.rejected"
	}
	_ = s.emitter.Emit(ctx, eventKind, map[string]any{
		"check_in_id":   checkInID,
		"owner_user_id": view.CheckIn.OwnerUserID,
		"goal_id":       view.CheckIn.GoalID,
		"decision":      string(input.Decision),
	})

	return record, nil
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
