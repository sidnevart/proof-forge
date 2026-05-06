package goals

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/ai"
	"github.com/sidnevart/proof-forge/backend/internal/platform/email"
	"github.com/sidnevart/proof-forge/backend/internal/users"
)

type noopEmailSender struct{}

func (noopEmailSender) SendBuddyInvite(_ context.Context, _ email.BuddyInviteParams) error {
	return nil
}
func (noopEmailSender) SendBuddyAccepted(_ context.Context, _ email.BuddyAcceptedParams) error {
	return nil
}

type repositoryStub struct {
	createGoal            func(context.Context, CreateGoalParams) (GoalView, error)
	listGoals             func(context.Context, int64) ([]GoalView, error)
	findInvite            func(context.Context, string) (InviteRecord, error)
	acceptInvite          func(context.Context, AcceptInviteParams) error
	isCircleMember        func(context.Context, int64, int64) (bool, error)
	isCircleMemberByEmail func(context.Context, int64, string) (bool, error)
	hasActiveGoalInCircle func(context.Context, int64, int64) (bool, error)
	findRefineCache       func(context.Context, string, time.Time) (GoalRefineResponse, bool, error)
	saveRefineCache       func(context.Context, string, GoalRefineResponse) error
	countRefineRequests   func(context.Context, int64, time.Time) (int, error)
	insertRefineRequest   func(context.Context, GoalRefineRequestLogParams) error
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

func (s repositoryStub) IsCircleMember(ctx context.Context, circleID int64, userID int64) (bool, error) {
	if s.isCircleMember == nil {
		return false, nil
	}
	return s.isCircleMember(ctx, circleID, userID)
}

func (s repositoryStub) IsCircleMemberByEmail(ctx context.Context, circleID int64, email string) (bool, error) {
	if s.isCircleMemberByEmail == nil {
		return false, nil
	}
	return s.isCircleMemberByEmail(ctx, circleID, email)
}

func (s repositoryStub) HasActiveGoalInCircle(ctx context.Context, circleID int64, ownerID int64) (bool, error) {
	if s.hasActiveGoalInCircle == nil {
		return false, nil
	}
	return s.hasActiveGoalInCircle(ctx, circleID, ownerID)
}

func (s repositoryStub) FindGoalRefineCache(ctx context.Context, draftHash string, minCreatedAt time.Time) (GoalRefineResponse, bool, error) {
	if s.findRefineCache == nil {
		return GoalRefineResponse{}, false, nil
	}
	return s.findRefineCache(ctx, draftHash, minCreatedAt)
}

func (s repositoryStub) SaveGoalRefineCache(ctx context.Context, draftHash string, response GoalRefineResponse) error {
	if s.saveRefineCache == nil {
		return nil
	}
	return s.saveRefineCache(ctx, draftHash, response)
}

func (s repositoryStub) CountGoalRefineRequestsSince(ctx context.Context, userID int64, since time.Time) (int, error) {
	if s.countRefineRequests == nil {
		return 0, nil
	}
	return s.countRefineRequests(ctx, userID, since)
}

func (s repositoryStub) InsertGoalRefineRequest(ctx context.Context, params GoalRefineRequestLogParams) error {
	if s.insertRefineRequest == nil {
		return nil
	}
	return s.insertRefineRequest(ctx, params)
}

func newTestStub() repositoryStub {
	return repositoryStub{
		createGoal:            func(context.Context, CreateGoalParams) (GoalView, error) { return GoalView{}, nil },
		listGoals:             func(context.Context, int64) ([]GoalView, error) { return nil, nil },
		isCircleMember:        func(context.Context, int64, int64) (bool, error) { return false, nil },
		isCircleMemberByEmail: func(context.Context, int64, string) (bool, error) { return false, nil },
		hasActiveGoalInCircle: func(context.Context, int64, int64) (bool, error) { return false, nil },
	}
}

func TestServiceCreateGoalRejectsSelfBuddy(t *testing.T) {
	service := NewService(newTestStub(), noopEmailSender{}, "", nil, 7*24*time.Hour)

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
	service := NewService(stub, noopEmailSender{}, "", nil, 7*24*time.Hour)

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

func TestCreateGoalAutoCreatesCircleWhenCircleIDOmitted(t *testing.T) {
	var captured CreateGoalParams

	stub := newTestStub()
	stub.createGoal = func(_ context.Context, params CreateGoalParams) (GoalView, error) {
		captured = params
		// Mirror what the real repository returns once it has fabricated a
		// circle: a non-nil CircleID derived from AutoCircle params.
		newCircleID := int64(42)
		params.CircleID = &newCircleID
		return GoalView{Goal: Goal{ID: 1, CircleID: newCircleID, Title: params.Title}}, nil
	}

	service := NewService(stub, noopEmailSender{}, "http://localhost:3000", nil, 7*24*time.Hour)
	service.tokenGenerate = func() (string, error) { return "tok-1", nil }
	service.circleCodeGenerate = func() (string, error) { return "circle-code", nil }

	owner := users.User{ID: 7, Email: "owner@example.com", DisplayName: "Owner"}
	view, err := service.CreateGoal(context.Background(), owner, CreateInput{
		Title:      "Бегать каждый день",
		BuddyName:  "Мария",
		BuddyEmail: "maria@example.com",
	})
	if err != nil {
		t.Fatalf("CreateGoal returned error: %v", err)
	}

	if captured.CircleID != nil {
		t.Fatalf("expected CircleID to be nil so repository auto-creates one, got %v", *captured.CircleID)
	}
	if captured.AutoCircle == nil {
		t.Fatalf("expected AutoCircle params to be populated for auto-create flow")
	}
	if captured.AutoCircle.InviteCode != "circle-code" {
		t.Fatalf("expected AutoCircle.InviteCode to come from circleCodeGenerate, got %q", captured.AutoCircle.InviteCode)
	}
	if captured.AutoCircle.Name != "Бегать каждый день" {
		t.Fatalf("expected AutoCircle.Name to be derived from goal title, got %q", captured.AutoCircle.Name)
	}
	if !captured.AutoCircle.EndsAt.After(captured.AutoCircle.StartsAt) {
		t.Fatalf("expected AutoCircle.EndsAt to be after StartsAt")
	}
	if got := captured.AutoCircle.EndsAt.Sub(captured.AutoCircle.StartsAt); got != 7*24*time.Hour {
		t.Fatalf("expected season length 7d, got %v", got)
	}
	if view.Goal.CircleID != 42 {
		t.Fatalf("expected returned goal.circle_id=42, got %d", view.Goal.CircleID)
	}
}

func TestCreateGoalRejectsSecondActiveGoalInCircle(t *testing.T) {
	stub := newTestStub()
	stub.isCircleMember = func(_ context.Context, circleID, userID int64) (bool, error) {
		return true, nil
	}
	stub.isCircleMemberByEmail = func(_ context.Context, _ int64, _ string) (bool, error) {
		return true, nil
	}
	stub.hasActiveGoalInCircle = func(_ context.Context, circleID, ownerID int64) (bool, error) {
		return true, nil
	}
	stub.createGoal = func(_ context.Context, _ CreateGoalParams) (GoalView, error) {
		t.Fatal("createGoal must not be called once HasActiveGoalInCircle reports true")
		return GoalView{}, nil
	}

	service := NewService(stub, noopEmailSender{}, "", nil, 7*24*time.Hour)

	_, err := service.CreateGoal(context.Background(), users.User{ID: 1, Email: "a@x.io"}, CreateInput{
		Title:      "Вторая цель",
		BuddyName:  "Buddy",
		BuddyEmail: "buddy@x.io",
		CircleID:   17,
	})
	if !errors.Is(err, ErrActiveGoalAlreadyExists) {
		t.Fatalf("expected ErrActiveGoalAlreadyExists, got %v", err)
	}
}

func TestCreateGoalPersistsProofExamplesAndCategory(t *testing.T) {
	var captured CreateGoalParams

	stub := newTestStub()
	stub.createGoal = func(_ context.Context, params CreateGoalParams) (GoalView, error) {
		captured = params
		return GoalView{}, nil
	}

	service := NewService(stub, noopEmailSender{}, "http://localhost:3000", nil, 7*24*time.Hour)
	service.tokenGenerate = func() (string, error) { return "token-123", nil }
	owner := users.User{ID: 1, Email: "owner@example.com", DisplayName: "Owner"}

	_, err := service.CreateGoal(context.Background(), owner, CreateInput{
		Title:         "Уточнить лендинг",
		Description:   "Сделать релиз измеримым",
		BuddyName:     "Peer",
		BuddyEmail:    "peer@example.com",
		ProofExamples: "- Ссылка\n- Скриншот\n- PR",
		Category:      "работа",
	})
	if err != nil {
		t.Fatalf("CreateGoal returned error: %v", err)
	}

	if captured.ProofExamples != "- Ссылка\n- Скриншот\n- PR" {
		t.Fatalf("expected proof examples to be forwarded, got %q", captured.ProofExamples)
	}
	if captured.Category != "работа" {
		t.Fatalf("expected category to be forwarded, got %q", captured.Category)
	}
}

func TestRefineGoalReturnsCachedResponseBeforeProviderCall(t *testing.T) {
	owner := users.User{ID: 7, Email: "owner@example.com"}
	cached := GoalRefineResponse{
		Category: "работа",
		Variants: []GoalRefineVariant{
			{
				Title:         "Cached",
				Smart:         "Cached smart goal",
				ProofExamples: []string{"One", "Two", "Three"},
			},
		},
	}

	stub := newTestStub()
	stub.countRefineRequests = func(_ context.Context, userID int64, since time.Time) (int, error) {
		if userID != owner.ID {
			t.Fatalf("expected user id %d, got %d", owner.ID, userID)
		}
		if since.IsZero() {
			t.Fatal("expected non-zero since time")
		}
		return 0, nil
	}
	stub.insertRefineRequest = func(_ context.Context, params GoalRefineRequestLogParams) error {
		if params.UserID != owner.ID {
			t.Fatalf("expected logged user id %d, got %d", owner.ID, params.UserID)
		}
		if params.DraftHash == "" {
			t.Fatal("expected non-empty draft hash")
		}
		return nil
	}
	stub.findRefineCache = func(_ context.Context, draftHash string, minCreatedAt time.Time) (GoalRefineResponse, bool, error) {
		if draftHash == "" {
			t.Fatal("expected non-empty draft hash")
		}
		if minCreatedAt.IsZero() {
			t.Fatal("expected non-zero cache cutoff")
		}
		return cached, true, nil
	}

	providerCalls := 0
	provider := refineProviderStub{
		refineGoal: func(context.Context, string) (ai.GoalRefineResult, error) {
			providerCalls++
			return ai.GoalRefineResult{}, nil
		},
	}

	service := NewService(
		stub,
		noopEmailSender{},
		"",
		nil,
		7*24*time.Hour,
		WithRefineProvider(provider),
	)

	result, err := service.RefineGoal(context.Background(), owner, RefineInput{DraftText: " ship launch checklist "})
	if err != nil {
		t.Fatalf("RefineGoal returned error: %v", err)
	}
	if providerCalls != 0 {
		t.Fatalf("expected provider not to be called on cache hit, got %d calls", providerCalls)
	}
	if !reflect.DeepEqual(result, cached) {
		t.Fatalf("expected cached response %#v, got %#v", cached, result)
	}
}

func TestRefineGoalReturnsRateLimitedError(t *testing.T) {
	owner := users.User{ID: 9, Email: "owner@example.com"}

	stub := newTestStub()
	stub.countRefineRequests = func(_ context.Context, userID int64, since time.Time) (int, error) {
		if userID != owner.ID {
			t.Fatalf("expected user id %d, got %d", owner.ID, userID)
		}
		if since.IsZero() {
			t.Fatal("expected non-zero since time")
		}
		return 5, nil
	}
	stub.insertRefineRequest = func(context.Context, GoalRefineRequestLogParams) error {
		t.Fatal("insert request log should not run when rate limited")
		return nil
	}

	provider := refineProviderStub{
		refineGoal: func(context.Context, string) (ai.GoalRefineResult, error) {
			t.Fatal("provider should not be called when rate limited")
			return ai.GoalRefineResult{}, nil
		},
	}

	service := NewService(
		stub,
		noopEmailSender{},
		"",
		nil,
		7*24*time.Hour,
		WithRefineProvider(provider),
	)

	_, err := service.RefineGoal(context.Background(), owner, RefineInput{DraftText: "improve onboarding"})
	if !errors.Is(err, ErrGoalRefineRateLimited) {
		t.Fatalf("expected ErrGoalRefineRateLimited, got %v", err)
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

	service := NewService(stub, noopEmailSender{}, "", nil, 7*24*time.Hour)
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

type refineProviderStub struct {
	refineGoal func(context.Context, string) (ai.GoalRefineResult, error)
}

func (s refineProviderStub) RefineGoal(ctx context.Context, draftText string) (ai.GoalRefineResult, error) {
	if s.refineGoal == nil {
		return ai.GoalRefineResult{}, nil
	}
	return s.refineGoal(ctx, draftText)
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

	service := NewService(stub, noopEmailSender{}, "", nil, 7*24*time.Hour)
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

	service := NewService(stub, noopEmailSender{}, "", nil, 7*24*time.Hour)
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

	service := NewService(stub, noopEmailSender{}, "", nil, 7*24*time.Hour)
	err := service.AcceptInvite(context.Background(), users.User{
		ID:    99,
		Email: "impostor@example.com",
	}, "rawtoken123")

	if !errors.Is(err, ErrUnauthorizedAcceptance) {
		t.Fatalf("expected ErrUnauthorizedAcceptance, got %v", err)
	}
}

func TestAcceptInviteNotFoundReturnsError(t *testing.T) {
	service := NewService(newTestStub(), noopEmailSender{}, "", nil, 7*24*time.Hour)
	err := service.AcceptInvite(context.Background(), users.User{
		ID:    2,
		Email: "buddy@example.com",
	}, "nonexistenttoken")

	if !errors.Is(err, ErrInviteNotFound) {
		t.Fatalf("expected ErrInviteNotFound, got %v", err)
	}
}

func TestGetInvitePreviewNotFoundReturnsError(t *testing.T) {
	service := NewService(newTestStub(), noopEmailSender{}, "", nil, 7*24*time.Hour)
	_, err := service.GetInvitePreview(context.Background(), "badtoken")
	if !errors.Is(err, ErrInviteNotFound) {
		t.Fatalf("expected ErrInviteNotFound, got %v", err)
	}
}
