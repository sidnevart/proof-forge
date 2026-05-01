package goals

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

type repositoryStub struct {
	createGoal   func(context.Context, CreateGoalParams) (GoalView, error)
	listGoals    func(context.Context, int64) ([]GoalView, error)
	findInvite   func(context.Context, string) (InviteRecord, error)
	acceptInvite func(context.Context, AcceptInviteParams) error
}

func (s repositoryStub) CreateGoalWithInvite(ctx context.Context, params CreateGoalParams) (GoalView, error) {
	return s.createGoal(ctx, params)
}

func (s repositoryStub) ListGoalsByOwner(ctx context.Context, ownerID int64) ([]GoalView, error) {
	return s.listGoals(ctx, ownerID)
}

func (s repositoryStub) FindInviteByToken(ctx context.Context, tokenHash string) (InviteRecord, error) {
	if s.findInvite == nil {
		return InviteRecord{}, ErrInviteNotFound
	}
	return s.findInvite(ctx, tokenHash)
}

func (s repositoryStub) AcceptInvite(ctx context.Context, params AcceptInviteParams) error {
	if s.acceptInvite == nil {
		return nil
	}
	return s.acceptInvite(ctx, params)
}

func newTestStub() repositoryStub {
	return repositoryStub{
		createGoal: func(context.Context, CreateGoalParams) (GoalView, error) { return GoalView{}, nil },
		listGoals:  func(context.Context, int64) ([]GoalView, error) { return nil, nil },
	}
}

func TestServiceCreateGoalRejectsSelfBuddy(t *testing.T) {
	service := NewService(newTestStub(), 7*24*time.Hour)

	_, err := service.CreateGoal(context.Background(), users.User{
		ID:    1,
		Email: "owner@example.com",
	}, CreateInput{
		Title:      "Ship MVP",
		BuddyName:  "Owner",
		BuddyEmail: "owner@example.com",
	})
	if !errors.Is(err, ErrInvalidGoalInput) {
		t.Fatalf("expected ErrInvalidGoalInput, got %v", err)
	}
}

func TestServiceDashboardBuildsSummary(t *testing.T) {
	stub := newTestStub()
	stub.listGoals = func(context.Context, int64) ([]GoalView, error) {
		return []GoalView{
			{Goal: Goal{Status: GoalStatusPendingBuddyAcceptance}},
			{Goal: Goal{Status: GoalStatusActive}},
		}, nil
	}
	service := NewService(stub, 7*24*time.Hour)

	dashboard, err := service.Dashboard(context.Background(), users.User{ID: 1})
	if err != nil {
		t.Fatalf("Dashboard() error = %v", err)
	}
	if dashboard.Summary.TotalGoals != 2 {
		t.Fatalf("expected total goals 2, got %d", dashboard.Summary.TotalGoals)
	}
	if dashboard.Summary.PendingBuddyAcceptance != 1 {
		t.Fatalf("expected pending goals 1, got %d", dashboard.Summary.PendingBuddyAcceptance)
	}
	if dashboard.Summary.ActiveGoals != 1 {
		t.Fatalf("expected active goals 1, got %d", dashboard.Summary.ActiveGoals)
	}
}

func TestAcceptInviteHappyPath(t *testing.T) {
	future := time.Now().Add(7 * 24 * time.Hour)
	stub := newTestStub()
	stub.findInvite = func(_ context.Context, _ string) (InviteRecord, error) {
		return InviteRecord{
			InviteID:     10,
			PactID:       20,
			GoalID:       30,
			InviteStatus: InviteStatusPending,
			InviteeEmail: "buddy@example.com",
			ExpiresAt:    future,
		}, nil
	}
	var capturedParams AcceptInviteParams
	stub.acceptInvite = func(_ context.Context, p AcceptInviteParams) error {
		capturedParams = p
		return nil
	}

	service := NewService(stub, 7*24*time.Hour)
	err := service.AcceptInvite(context.Background(), users.User{
		ID:    2,
		Email: "buddy@example.com",
	}, "rawtoken123")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedParams.InviteID != 10 || capturedParams.PactID != 20 || capturedParams.GoalID != 30 {
		t.Fatalf("wrong params passed to AcceptInvite: %+v", capturedParams)
	}
}

func TestAcceptInviteExpiredReturnsErrInviteExpired(t *testing.T) {
	stub := newTestStub()
	stub.findInvite = func(_ context.Context, _ string) (InviteRecord, error) {
		return InviteRecord{
			InviteID:     10,
			InviteStatus: InviteStatusPending,
			InviteeEmail: "buddy@example.com",
			ExpiresAt:    time.Now().Add(-1 * time.Hour),
		}, nil
	}

	service := NewService(stub, 7*24*time.Hour)
	err := service.AcceptInvite(context.Background(), users.User{
		ID:    2,
		Email: "buddy@example.com",
	}, "rawtoken123")

	if !errors.Is(err, ErrInviteExpired) {
		t.Fatalf("expected ErrInviteExpired, got %v", err)
	}
}

func TestAcceptInviteAlreadyAcceptedReturnsError(t *testing.T) {
	stub := newTestStub()
	stub.findInvite = func(_ context.Context, _ string) (InviteRecord, error) {
		return InviteRecord{
			InviteID:     10,
			InviteStatus: InviteStatusAccepted,
			InviteeEmail: "buddy@example.com",
			ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		}, nil
	}

	service := NewService(stub, 7*24*time.Hour)
	err := service.AcceptInvite(context.Background(), users.User{
		ID:    2,
		Email: "buddy@example.com",
	}, "rawtoken123")

	if !errors.Is(err, ErrInviteAlreadyAccepted) {
		t.Fatalf("expected ErrInviteAlreadyAccepted, got %v", err)
	}
}

func TestAcceptInviteWrongEmailReturnsUnauthorized(t *testing.T) {
	stub := newTestStub()
	stub.findInvite = func(_ context.Context, _ string) (InviteRecord, error) {
		return InviteRecord{
			InviteID:     10,
			InviteStatus: InviteStatusPending,
			InviteeEmail: "buddy@example.com",
			ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		}, nil
	}

	service := NewService(stub, 7*24*time.Hour)
	err := service.AcceptInvite(context.Background(), users.User{
		ID:    99,
		Email: "impostor@example.com",
	}, "rawtoken123")

	if !errors.Is(err, ErrUnauthorizedAcceptance) {
		t.Fatalf("expected ErrUnauthorizedAcceptance, got %v", err)
	}
}

func TestAcceptInviteNotFoundReturnsError(t *testing.T) {
	service := NewService(newTestStub(), 7*24*time.Hour)
	err := service.AcceptInvite(context.Background(), users.User{
		ID:    2,
		Email: "buddy@example.com",
	}, "nonexistenttoken")

	if !errors.Is(err, ErrInviteNotFound) {
		t.Fatalf("expected ErrInviteNotFound, got %v", err)
	}
}

func TestGetInvitePreviewNotFoundReturnsError(t *testing.T) {
	service := NewService(newTestStub(), 7*24*time.Hour)
	_, err := service.GetInvitePreview(context.Background(), "badtoken")
	if !errors.Is(err, ErrInviteNotFound) {
		t.Fatalf("expected ErrInviteNotFound, got %v", err)
	}
}
