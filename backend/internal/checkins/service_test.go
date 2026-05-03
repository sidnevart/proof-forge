package checkins

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// --- stubs ---

type repoStub struct {
	createCheckIn  func(context.Context, CreateCheckInParams) (CheckIn, error)
	getCheckIn     func(context.Context, int64) (CheckInView, error)
	listCheckIns   func(context.Context, int64, int64) ([]CheckIn, error)
	updateStatus   func(context.Context, UpdateStatusParams) error
	insertEvidence func(context.Context, InsertEvidenceParams) (EvidenceItem, error)
	countEvidence  func(context.Context, int64) (int, error)
	recordReview   func(context.Context, RecordReviewParams) (ReviewRecord, error)
	deleteCheckIn  func(context.Context, int64) error
	setDeadline    func(context.Context, int64, int64, *time.Time) error
}

func (s repoStub) CreateCheckIn(ctx context.Context, p CreateCheckInParams) (CheckIn, error) {
	if s.createCheckIn != nil {
		return s.createCheckIn(ctx, p)
	}
	return CheckIn{ID: 1, GoalID: p.GoalID, OwnerUserID: p.OwnerUserID, Status: p.Status, CreatedAt: time.Now()}, nil
}
func (s repoStub) GetCheckIn(ctx context.Context, id int64) (CheckInView, error) {
	if s.getCheckIn != nil {
		return s.getCheckIn(ctx, id)
	}
	return CheckInView{}, ErrCheckInNotFound
}
func (s repoStub) ListCheckInsByGoal(ctx context.Context, gID, aID int64) ([]CheckIn, error) {
	if s.listCheckIns != nil {
		return s.listCheckIns(ctx, gID, aID)
	}
	return nil, nil
}
func (s repoStub) UpdateCheckInStatus(ctx context.Context, p UpdateStatusParams) error {
	if s.updateStatus != nil {
		return s.updateStatus(ctx, p)
	}
	return nil
}
func (s repoStub) InsertEvidence(ctx context.Context, p InsertEvidenceParams) (EvidenceItem, error) {
	if s.insertEvidence != nil {
		return s.insertEvidence(ctx, p)
	}
	return EvidenceItem{ID: 99, CheckInID: p.CheckInID, Kind: p.Kind}, nil
}
func (s repoStub) CountEvidence(ctx context.Context, id int64) (int, error) {
	if s.countEvidence != nil {
		return s.countEvidence(ctx, id)
	}
	return 0, nil
}
func (s repoStub) RecordReview(ctx context.Context, p RecordReviewParams) (ReviewRecord, error) {
	if s.recordReview != nil {
		return s.recordReview(ctx, p)
	}
	return ReviewRecord{ID: 1, CheckInID: p.CheckInID, ReviewerUserID: p.ReviewerUserID, Decision: p.Decision, Comment: p.Comment}, nil
}
func (s repoStub) DeleteCheckIn(ctx context.Context, id int64) error {
	if s.deleteCheckIn != nil {
		return s.deleteCheckIn(ctx, id)
	}
	return nil
}
func (s repoStub) SetCheckInDeadline(ctx context.Context, checkInID, ownerUserID int64, deadline *time.Time) error {
	if s.setDeadline != nil {
		return s.setDeadline(ctx, checkInID, ownerUserID, deadline)
	}
	return nil
}

type storageStub struct {
	putErr error
}

func (s storageStub) Put(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	return s.putErr
}
func (s storageStub) ObjectKey(checkInID int64, ext string) string {
	return "evidence/1/test" + ext
}

func pendingView(ownerID, buddyID int64) CheckInView {
	return CheckInView{
		CheckIn: CheckIn{
			ID: 1, GoalID: 10, OwnerUserID: ownerID,
			Status: StatusDraft, CreatedAt: time.Now(),
		},
		BuddyUserID: buddyID,
	}
}

func owner() users.User { return users.User{ID: 1, Email: "owner@example.com"} }
func buddy() users.User { return users.User{ID: 2, Email: "buddy@example.com"} }

// --- tests ---

func TestCreateCheckInGoalNotEligibleReturnsError(t *testing.T) {
	repo := repoStub{
		createCheckIn: func(_ context.Context, _ CreateCheckInParams) (CheckIn, error) {
			return CheckIn{}, ErrGoalNotEligible
		},
	}
	svc := NewService(repo, storageStub{})
	_, err := svc.CreateCheckIn(context.Background(), owner(), 10, nil)
	if !errors.Is(err, ErrGoalNotEligible) {
		t.Fatalf("expected ErrGoalNotEligible, got %v", err)
	}
}

func TestCreateCheckInHappyPath(t *testing.T) {
	svc := NewService(repoStub{}, storageStub{})
	ci, err := svc.CreateCheckIn(context.Background(), owner(), 10, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ci.Status != StatusDraft {
		t.Fatalf("expected draft, got %s", ci.Status)
	}
}

func TestGetDetailUnauthorizedBuddy(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 3), nil // buddy is 3, not 2
		},
	}
	svc := NewService(repo, storageStub{})
	_, err := svc.GetDetail(context.Background(), buddy(), 1) // actor.ID=2
	if !errors.Is(err, ErrNotAuthorized) {
		t.Fatalf("expected ErrNotAuthorized, got %v", err)
	}
}

func TestGetDetailOwnerCanSeeOwnCheckIn(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 2), nil
		},
	}
	svc := NewService(repo, storageStub{})
	_, err := svc.GetDetail(context.Background(), owner(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDetailBuddyCanSeeGoalCheckIn(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 2), nil // buddy_user_id = 2
		},
	}
	svc := NewService(repo, storageStub{})
	_, err := svc.GetDetail(context.Background(), buddy(), 1)
	if err != nil {
		t.Fatalf("buddy should be authorized, got %v", err)
	}
}

func TestSubmitNonOwnerReturnsError(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 2), nil
		},
	}
	svc := NewService(repo, storageStub{})
	err := svc.Submit(context.Background(), buddy(), 1)
	if !errors.Is(err, ErrNotOwner) {
		t.Fatalf("expected ErrNotOwner, got %v", err)
	}
}

func TestSubmitAlreadySubmittedReturnsError(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			v := pendingView(1, 2)
			v.CheckIn.Status = StatusSubmitted
			return v, nil
		},
	}
	svc := NewService(repo, storageStub{})
	err := svc.Submit(context.Background(), owner(), 1)
	if !errors.Is(err, ErrCannotSubmit) {
		t.Fatalf("expected ErrCannotSubmit, got %v", err)
	}
}

func TestSubmitDraftSucceeds(t *testing.T) {
	var captured UpdateStatusParams
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 2), nil
		},
		updateStatus: func(_ context.Context, p UpdateStatusParams) error {
			captured = p
			return nil
		},
	}
	svc := NewService(repo, storageStub{})
	if err := svc.Submit(context.Background(), owner(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Status != StatusSubmitted {
		t.Fatalf("expected submitted status, got %s", captured.Status)
	}
	if captured.SubmittedAt == nil {
		t.Fatalf("SubmittedAt should be set")
	}
}

func TestAddTextEvidenceEmptyContentReturnsError(t *testing.T) {
	svc := NewService(repoStub{}, storageStub{})
	_, err := svc.AddTextEvidence(context.Background(), owner(), 1, AddTextInput{Content: "  "})
	if !errors.Is(err, ErrInvalidEvidenceInput) {
		t.Fatalf("expected ErrInvalidEvidenceInput, got %v", err)
	}
}

func TestAddTextEvidenceApprovedCheckInReturnsError(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			v := pendingView(1, 2)
			v.CheckIn.Status = StatusApproved
			return v, nil
		},
	}
	svc := NewService(repo, storageStub{})
	_, err := svc.AddTextEvidence(context.Background(), owner(), 1, AddTextInput{Content: "proof"})
	if !errors.Is(err, ErrCannotAddEvidence) {
		t.Fatalf("expected ErrCannotAddEvidence, got %v", err)
	}
}

func TestAddTextEvidenceHappyPath(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 2), nil
		},
	}
	svc := NewService(repo, storageStub{})
	item, err := svc.AddTextEvidence(context.Background(), owner(), 1, AddTextInput{Content: "I did the thing"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Kind != KindText {
		t.Fatalf("expected text kind, got %s", item.Kind)
	}
}

func TestAddLinkEvidenceInvalidURLReturnsError(t *testing.T) {
	svc := NewService(repoStub{}, storageStub{})
	_, err := svc.AddLinkEvidence(context.Background(), owner(), 1, AddLinkInput{URL: "not-a-url"})
	if !errors.Is(err, ErrInvalidEvidenceInput) {
		t.Fatalf("expected ErrInvalidEvidenceInput, got %v", err)
	}
}

func TestAddFileEvidenceTooLargeReturnsError(t *testing.T) {
	svc := NewService(repoStub{}, storageStub{})
	big := make([]byte, MaxFileSizeBytes+1)
	_, err := svc.AddFileEvidence(context.Background(), owner(), 1, big, "image/png")
	if !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("expected ErrFileTooLarge, got %v", err)
	}
}

func TestAddFileEvidenceUnsupportedMIMEReturnsError(t *testing.T) {
	svc := NewService(repoStub{}, storageStub{})
	_, err := svc.AddFileEvidence(context.Background(), owner(), 1, []byte("data"), "video/mp4")
	if !errors.Is(err, ErrUnsupportedMIME) {
		t.Fatalf("expected ErrUnsupportedMIME, got %v", err)
	}
}

func TestAddFileEvidenceStorageFailureReturnsError(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 2), nil
		},
	}
	svc := NewService(repo, storageStub{putErr: errors.New("s3 down")})
	data := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte("x"), 92)...)
	_, err := svc.AddFileEvidence(context.Background(), owner(), 1, data, "image/png")
	if err == nil {
		t.Fatal("expected error from storage failure")
	}
}

func TestAddFileEvidenceHappyPath(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 2), nil
		},
	}
	svc := NewService(repo, storageStub{})
	// Use a real PNG magic header so DetectMIME accepts it.
	data := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte("x"), 92)...)
	item, err := svc.AddFileEvidence(context.Background(), owner(), 1, data, "image/png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Kind != KindImage {
		t.Fatalf("expected image kind, got %s", item.Kind)
	}
}

func TestReviewNotBuddyReturnsError(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			v := pendingView(1, 2)
			v.CheckIn.Status = StatusSubmitted
			return v, nil
		},
	}
	svc := NewService(repo, storageStub{})
	_, err := svc.Review(context.Background(), owner(), 1, ReviewInput{Decision: DecisionApprove})
	if !errors.Is(err, ErrNotBuddy) {
		t.Fatalf("expected ErrNotBuddy, got %v", err)
	}
}

func TestReviewNotSubmittedReturnsError(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 2), nil // status is draft
		},
	}
	svc := NewService(repo, storageStub{})
	_, err := svc.Review(context.Background(), buddy(), 1, ReviewInput{Decision: DecisionApprove})
	if !errors.Is(err, ErrCannotReview) {
		t.Fatalf("expected ErrCannotReview, got %v", err)
	}
}

func TestReviewApproveHappyPath(t *testing.T) {
	var captured RecordReviewParams
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			v := pendingView(1, 2)
			v.CheckIn.Status = StatusSubmitted
			return v, nil
		},
		recordReview: func(_ context.Context, p RecordReviewParams) (ReviewRecord, error) {
			captured = p
			return ReviewRecord{ID: 1, CheckInID: p.CheckInID, Decision: p.Decision}, nil
		},
	}
	svc := NewService(repo, storageStub{})
	rec, err := svc.Review(context.Background(), buddy(), 1, ReviewInput{Decision: DecisionApprove, Comment: "great work"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Decision != DecisionApprove {
		t.Fatalf("expected approved decision, got %s", rec.Decision)
	}
	if captured.Decision != DecisionApprove {
		t.Fatalf("repo received wrong decision: %s", captured.Decision)
	}
	if captured.Comment != "great work" {
		t.Fatalf("expected comment to be passed, got %q", captured.Comment)
	}
}

func TestReviewRejectHappyPath(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			v := pendingView(1, 2)
			v.CheckIn.Status = StatusSubmitted
			return v, nil
		},
	}
	svc := NewService(repo, storageStub{})
	rec, err := svc.Review(context.Background(), buddy(), 1, ReviewInput{Decision: DecisionReject})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Decision != DecisionReject {
		t.Fatalf("expected rejected decision, got %s", rec.Decision)
	}
}

func TestReviewRequestChangesHappyPath(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			v := pendingView(1, 2)
			v.CheckIn.Status = StatusSubmitted
			return v, nil
		},
	}
	svc := NewService(repo, storageStub{})
	rec, err := svc.Review(context.Background(), buddy(), 1, ReviewInput{Decision: DecisionRequestChanges, Comment: "need more detail"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Decision != DecisionRequestChanges {
		t.Fatalf("expected changes_requested decision, got %s", rec.Decision)
	}
}

func TestCapExceededReturnsError(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 2), nil
		},
		countEvidence: func(_ context.Context, _ int64) (int, error) {
			return MaxEvidenceItems, nil
		},
	}
	svc := NewService(repo, storageStub{})
	_, err := svc.AddTextEvidence(context.Background(), owner(), 1, AddTextInput{Content: "one more"})
	if !errors.Is(err, ErrTooManyEvidenceItems) {
		t.Fatalf("expected ErrTooManyEvidenceItems, got %v", err)
	}
}
