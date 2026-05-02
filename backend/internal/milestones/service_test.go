package milestones

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

type mockRepo struct {
	milestones  map[int64]Milestone
	goalOwner   int64
	goalBuddy   int64
	goalStatus  string
	nextID      int64
	sortCounter int
}

func newMockRepo(ownerID, buddyID int64, goalStatus string) *mockRepo {
	return &mockRepo{
		milestones: make(map[int64]Milestone),
		goalOwner:  ownerID,
		goalBuddy:  buddyID,
		goalStatus: goalStatus,
		nextID:     1,
	}
}

func (m *mockRepo) Insert(_ context.Context, goalID int64, title, description string, sortOrder int) (Milestone, error) {
	now := time.Now().UTC()
	ms := Milestone{
		ID: m.nextID, GoalID: goalID, Title: title, Description: description,
		Status: StatusPending, SortOrder: sortOrder,
		CreatedAt: now, UpdatedAt: now,
	}
	m.milestones[ms.ID] = ms
	m.nextID++
	return ms, nil
}

func (m *mockRepo) FindByGoal(_ context.Context, _ int64) ([]Milestone, error) {
	out := make([]Milestone, 0, len(m.milestones))
	for _, ms := range m.milestones {
		out = append(out, ms)
	}
	return out, nil
}

func (m *mockRepo) FindByID(_ context.Context, milestoneID int64) (Milestone, error) {
	ms, ok := m.milestones[milestoneID]
	if !ok {
		return Milestone{}, ErrMilestoneNotFound
	}
	return ms, nil
}

func (m *mockRepo) Update(_ context.Context, milestoneID int64, title, description *string, sortOrder *int) (Milestone, error) {
	ms, ok := m.milestones[milestoneID]
	if !ok {
		return Milestone{}, ErrMilestoneNotFound
	}
	if title != nil {
		ms.Title = *title
	}
	if description != nil {
		ms.Description = *description
	}
	if sortOrder != nil {
		ms.SortOrder = *sortOrder
	}
	ms.UpdatedAt = time.Now().UTC()
	m.milestones[milestoneID] = ms
	return ms, nil
}

func (m *mockRepo) Delete(_ context.Context, milestoneID int64) error {
	if _, ok := m.milestones[milestoneID]; !ok {
		return ErrMilestoneNotFound
	}
	delete(m.milestones, milestoneID)
	return nil
}

func (m *mockRepo) UpdateStatus(_ context.Context, milestoneID int64, status MilestoneStatus, completedByUserID *int64, now time.Time) (Milestone, error) {
	ms, ok := m.milestones[milestoneID]
	if !ok {
		return Milestone{}, ErrMilestoneNotFound
	}
	ms.Status = status
	ms.UpdatedAt = now
	if status == StatusCompleted {
		ms.CompletedAt = &now
		ms.CompletedByUserID = completedByUserID
	} else {
		ms.CompletedAt = nil
		ms.CompletedByUserID = nil
	}
	m.milestones[milestoneID] = ms
	return ms, nil
}

func (m *mockRepo) FindGoalParticipants(_ context.Context, _ int64) (int64, int64, string, error) {
	return m.goalOwner, m.goalBuddy, m.goalStatus, nil
}

func (m *mockRepo) NextSortOrder(_ context.Context, _ int64) (int, error) {
	m.sortCounter++
	return m.sortCounter - 1, nil
}

func TestCreate_OwnerSucceeds(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)

	m, err := svc.Create(context.Background(), users.User{ID: 1}, 10, CreateInput{Title: "Изучить корутины"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Title != "Изучить корутины" {
		t.Errorf("title mismatch: %q", m.Title)
	}
	if m.Status != StatusPending {
		t.Errorf("expected pending, got %q", m.Status)
	}
}

func TestCreate_NotOwner(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), users.User{ID: 2}, 10, CreateInput{Title: "X"})
	if err == nil || !errIs(err, ErrNotGoalOwner) {
		t.Fatalf("expected ErrNotGoalOwner, got %v", err)
	}
}

func TestCreate_GoalNotActive(t *testing.T) {
	repo := newMockRepo(1, 2, "pending_buddy_acceptance")
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), users.User{ID: 1}, 10, CreateInput{Title: "X"})
	if err == nil || !errIs(err, ErrGoalNotActive) {
		t.Fatalf("expected ErrGoalNotActive, got %v", err)
	}
}

func TestCreate_EmptyTitle(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), users.User{ID: 1}, 10, CreateInput{Title: "  "})
	if err == nil || !errIs(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestComplete_BuddyOnly(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)
	owner := users.User{ID: 1}
	buddy := users.User{ID: 2}

	m, _ := svc.Create(context.Background(), owner, 10, CreateInput{Title: "X"})

	// Owner cannot complete
	_, err := svc.Complete(context.Background(), owner, m.ID)
	if err == nil || !errIs(err, ErrNotBuddy) {
		t.Errorf("expected ErrNotBuddy for owner, got %v", err)
	}

	// Buddy can
	completed, err := svc.Complete(context.Background(), buddy, m.ID)
	if err != nil {
		t.Fatalf("buddy complete: %v", err)
	}
	if completed.Status != StatusCompleted {
		t.Errorf("expected completed, got %q", completed.Status)
	}
	if completed.CompletedByUserID == nil || *completed.CompletedByUserID != 2 {
		t.Errorf("expected completed_by user 2, got %v", completed.CompletedByUserID)
	}
}

func TestComplete_NotPending(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)
	owner := users.User{ID: 1}
	buddy := users.User{ID: 2}

	m, _ := svc.Create(context.Background(), owner, 10, CreateInput{Title: "X"})
	_, _ = svc.Complete(context.Background(), buddy, m.ID)

	// Cannot complete twice
	_, err := svc.Complete(context.Background(), buddy, m.ID)
	if err == nil || !errIs(err, ErrMilestoneNotPending) {
		t.Errorf("expected ErrMilestoneNotPending, got %v", err)
	}
}

func TestReopen_BuddyOnly(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)
	owner := users.User{ID: 1}
	buddy := users.User{ID: 2}

	m, _ := svc.Create(context.Background(), owner, 10, CreateInput{Title: "X"})
	_, _ = svc.Complete(context.Background(), buddy, m.ID)

	reopened, err := svc.Reopen(context.Background(), buddy, m.ID)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if reopened.Status != StatusPending {
		t.Errorf("expected pending after reopen, got %q", reopened.Status)
	}
	if reopened.CompletedAt != nil {
		t.Errorf("expected completed_at cleared")
	}
}

func TestUpdate_OwnerWhilePending(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)
	owner := users.User{ID: 1}

	m, _ := svc.Create(context.Background(), owner, 10, CreateInput{Title: "Old"})
	newTitle := "New title"
	updated, err := svc.Update(context.Background(), owner, m.ID, UpdateInput{Title: &newTitle})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Title != "New title" {
		t.Errorf("expected updated title, got %q", updated.Title)
	}
}

func TestDelete_OwnerWhilePending(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)
	owner := users.User{ID: 1}

	m, _ := svc.Create(context.Background(), owner, 10, CreateInput{Title: "X"})
	if err := svc.Delete(context.Background(), owner, m.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err := repo.FindByID(context.Background(), m.ID)
	if err == nil || !errIs(err, ErrMilestoneNotFound) {
		t.Errorf("expected milestone deleted, got %v", err)
	}
}

func TestListForGoal_OnlyParticipants(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)

	_, err := svc.ListForGoal(context.Background(), users.User{ID: 99}, 10)
	if err == nil {
		t.Fatal("expected error for non-participant")
	}
}

func errIs(err, target error) bool {
	return errors.Is(err, target)
}
