package recaps

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// --- stubs ---

type repoStub struct {
	findGoals        func(context.Context, time.Time, time.Time) ([]GoalForRecap, error)
	findApproved     func(context.Context, int64, time.Time, time.Time) ([]ApprovedCheckIn, error)
	insertRecap      func(context.Context, InsertRecapParams) (WeeklyRecap, error)
	updateRecap      func(context.Context, UpdateRecapParams) error
	getRecap         func(context.Context, int64) (WeeklyRecap, int64, error)
	listRecapsByGoal func(context.Context, int64, int64) ([]WeeklyRecap, error)
}

func (s repoStub) FindGoalsNeedingRecap(ctx context.Context, start, end time.Time) ([]GoalForRecap, error) {
	if s.findGoals != nil {
		return s.findGoals(ctx, start, end)
	}
	return nil, nil
}
func (s repoStub) FindApprovedCheckIns(ctx context.Context, gID int64, start, end time.Time) ([]ApprovedCheckIn, error) {
	if s.findApproved != nil {
		return s.findApproved(ctx, gID, start, end)
	}
	return nil, nil
}
func (s repoStub) InsertRecap(ctx context.Context, p InsertRecapParams) (WeeklyRecap, error) {
	if s.insertRecap != nil {
		return s.insertRecap(ctx, p)
	}
	return WeeklyRecap{ID: 1, GoalID: p.GoalID, Status: p.Status}, nil
}
func (s repoStub) UpdateRecap(ctx context.Context, p UpdateRecapParams) error {
	if s.updateRecap != nil {
		return s.updateRecap(ctx, p)
	}
	return nil
}
func (s repoStub) GetRecap(ctx context.Context, id int64) (WeeklyRecap, int64, error) {
	if s.getRecap != nil {
		return s.getRecap(ctx, id)
	}
	return WeeklyRecap{}, 0, ErrRecapNotFound
}
func (s repoStub) ListRecapsByGoal(ctx context.Context, gID, aID int64) ([]WeeklyRecap, error) {
	if s.listRecapsByGoal != nil {
		return s.listRecapsByGoal(ctx, gID, aID)
	}
	return nil, nil
}

type aiStub struct {
	summary   string
	modelName string
	err       error
}

func (a aiStub) Summarize(_ context.Context, _ string) (string, string, error) {
	return a.summary, a.modelName, a.err
}

func newSvc(repo Repository, ai AIProvider) *Service {
	return NewService(repo, ai, slog.New(slog.NewTextHandler(os.Stderr, nil)))
}

func ownerUser() users.User { return users.User{ID: 1, Email: "owner@example.com"} }
func buddyUser() users.User { return users.User{ID: 2, Email: "buddy@example.com"} }
func otherUser() users.User { return users.User{ID: 3, Email: "other@example.com"} }

// --- tests ---

func TestGetRecap_OwnerCanAccess(t *testing.T) {
	recap := WeeklyRecap{ID: 7, GoalID: 10, OwnerUserID: 1, Status: StatusDone}
	svc := newSvc(repoStub{
		getRecap: func(_ context.Context, _ int64) (WeeklyRecap, int64, error) {
			return recap, 2, nil
		},
	}, aiStub{})

	got, err := svc.GetRecap(context.Background(), ownerUser(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 7 {
		t.Errorf("expected recap 7, got %d", got.ID)
	}
}

func TestGetRecap_BuddyCanAccess(t *testing.T) {
	recap := WeeklyRecap{ID: 7, GoalID: 10, OwnerUserID: 1, Status: StatusDone}
	svc := newSvc(repoStub{
		getRecap: func(_ context.Context, _ int64) (WeeklyRecap, int64, error) {
			return recap, 2, nil
		},
	}, aiStub{})

	if _, err := svc.GetRecap(context.Background(), buddyUser(), 7); err != nil {
		t.Fatalf("buddy should be able to access: %v", err)
	}
}

func TestGetRecap_OtherUserDenied(t *testing.T) {
	recap := WeeklyRecap{ID: 7, GoalID: 10, OwnerUserID: 1}
	svc := newSvc(repoStub{
		getRecap: func(_ context.Context, _ int64) (WeeklyRecap, int64, error) {
			return recap, 2, nil
		},
	}, aiStub{})

	_, err := svc.GetRecap(context.Background(), otherUser(), 7)
	if !errors.Is(err, ErrNotAuthorized) {
		t.Errorf("expected ErrNotAuthorized, got %v", err)
	}
}

func TestGetRecap_NotFound(t *testing.T) {
	svc := newSvc(repoStub{}, aiStub{})
	_, err := svc.GetRecap(context.Background(), ownerUser(), 999)
	if !errors.Is(err, ErrRecapNotFound) {
		t.Errorf("expected ErrRecapNotFound, got %v", err)
	}
}

func TestSweepAndGenerate_GeneratesForEligibleGoal(t *testing.T) {
	var updatedParams UpdateRecapParams
	goal := GoalForRecap{ID: 10, OwnerUserID: 1, Title: "Ship MVP", Description: "Ship it"}

	svc := newSvc(repoStub{
		findGoals: func(_ context.Context, _, _ time.Time) ([]GoalForRecap, error) {
			return []GoalForRecap{goal}, nil
		},
		findApproved: func(_ context.Context, _ int64, _, _ time.Time) ([]ApprovedCheckIn, error) {
			return []ApprovedCheckIn{
				{ID: 5, ApprovedAt: time.Now(), Evidence: []EvidenceSummary{{Kind: "text", TextContent: "Merged PR"}}},
			}, nil
		},
		updateRecap: func(_ context.Context, p UpdateRecapParams) error {
			updatedParams = p
			return nil
		},
	}, aiStub{summary: "Great progress this week!", modelName: "gpt-4o-mini"})

	if err := svc.SweepAndGenerate(context.Background(), time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedParams.Status != StatusDone {
		t.Errorf("expected StatusDone, got %v", updatedParams.Status)
	}
	if updatedParams.SummaryText != "Great progress this week!" {
		t.Errorf("unexpected summary: %q", updatedParams.SummaryText)
	}
	if updatedParams.ModelName != "gpt-4o-mini" {
		t.Errorf("unexpected model: %q", updatedParams.ModelName)
	}
}

func TestSweepAndGenerate_AIFailureMarksFailed(t *testing.T) {
	var updatedStatus RecapStatus
	svc := newSvc(repoStub{
		findGoals: func(_ context.Context, _, _ time.Time) ([]GoalForRecap, error) {
			return []GoalForRecap{{ID: 10, OwnerUserID: 1}}, nil
		},
		updateRecap: func(_ context.Context, p UpdateRecapParams) error {
			updatedStatus = p.Status
			return nil
		},
	}, aiStub{err: errors.New("api error")})

	// single failure must not be returned — sweep continues
	_ = svc.SweepAndGenerate(context.Background(), time.Now())
	if updatedStatus != StatusFailed {
		t.Errorf("expected StatusFailed, got %v", updatedStatus)
	}
}

func TestSweepAndGenerate_NoGoalsNoError(t *testing.T) {
	svc := newSvc(repoStub{}, aiStub{})
	if err := svc.SweepAndGenerate(context.Background(), time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsoWeek_Wednesday(t *testing.T) {
	// 2026-04-29 is a Wednesday
	wednesday := time.Date(2026, 4, 29, 15, 0, 0, 0, time.UTC)
	start, end := isoWeek(wednesday)

	wantStart := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC) // Monday
	wantEnd := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)    // next Monday

	if !start.Equal(wantStart) {
		t.Errorf("expected start %v, got %v", wantStart, start)
	}
	if !end.Equal(wantEnd) {
		t.Errorf("expected end %v, got %v", wantEnd, end)
	}
}

func TestBuildPrompt_ContainsGoalAndEvidence(t *testing.T) {
	goal := GoalForRecap{ID: 1, Title: "Run 5k", Description: "Daily run goal"}
	period := [2]time.Time{
		time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
	}
	checkIns := []ApprovedCheckIn{
		{
			ID:         5,
			ApprovedAt: time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC),
			Evidence: []EvidenceSummary{
				{Kind: "text", TextContent: "Ran 5.2 km"},
				{Kind: "link", ExternalURL: "https://strava.com/act/1"},
			},
		},
	}

	prompt := BuildPrompt(goal, period, checkIns)

	for _, want := range []string{"Run 5k", "Daily run goal", "Ran 5.2 km", "https://strava.com/act/1", "Do NOT approve"} {
		if !contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
