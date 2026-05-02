package stakes

import (
	"context"
	"testing"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

type mockRepo struct {
	stakes       map[int64]Stake
	forfeitures  []StakeForfeiture
	goalOwner    int64
	goalBuddy    int64
	goalStatus   string
	nextStakeID  int64
	nextForfID   int64
	insertErr    error
}

func newMockRepo(ownerID, buddyID int64, goalStatus string) *mockRepo {
	return &mockRepo{
		stakes:      make(map[int64]Stake),
		goalOwner:   ownerID,
		goalBuddy:   buddyID,
		goalStatus:  goalStatus,
		nextStakeID: 1,
		nextForfID:  1,
	}
}

func (m *mockRepo) Insert(_ context.Context, goalID, ownerUserID int64, description string) (Stake, error) {
	if m.insertErr != nil {
		return Stake{}, m.insertErr
	}
	now := time.Now().UTC()
	s := Stake{
		ID:          m.nextStakeID,
		GoalID:      goalID,
		OwnerUserID: ownerUserID,
		Description: description,
		Status:      StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	m.stakes[s.ID] = s
	m.nextStakeID++
	return s, nil
}

func (m *mockRepo) FindByGoal(_ context.Context, _ int64) ([]StakeView, error) {
	var views []StakeView
	for _, s := range m.stakes {
		views = append(views, StakeView{Stake: s})
	}
	return views, nil
}

func (m *mockRepo) FindByID(_ context.Context, stakeID int64) (Stake, error) {
	s, ok := m.stakes[stakeID]
	if !ok {
		return Stake{}, ErrStakeNotFound
	}
	return s, nil
}

func (m *mockRepo) UpdateStatus(_ context.Context, stakeID int64, status StakeStatus, now time.Time) error {
	s, ok := m.stakes[stakeID]
	if !ok {
		return ErrStakeNotFound
	}
	s.Status = status
	s.UpdatedAt = now
	switch status {
	case StatusForfeited:
		s.ForfeitedAt = &now
	case StatusCompleted:
		s.CompletedAt = &now
	case StatusCancelled:
		s.CancelledAt = &now
	}
	m.stakes[stakeID] = s
	return nil
}

func (m *mockRepo) InsertForfeiture(_ context.Context, stakeID, declaredByUserID int64, reason string) (StakeForfeiture, error) {
	f := StakeForfeiture{
		ID:               m.nextForfID,
		StakeID:          stakeID,
		DeclaredByUserID: declaredByUserID,
		Reason:           reason,
		CreatedAt:        time.Now().UTC(),
	}
	m.forfeitures = append(m.forfeitures, f)
	m.nextForfID++
	return f, nil
}

func (m *mockRepo) FindGoalParticipants(_ context.Context, _ int64) (int64, int64, string, error) {
	return m.goalOwner, m.goalBuddy, m.goalStatus, nil
}

func TestCreate_Success(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)
	actor := users.User{ID: 1}

	view, err := svc.Create(context.Background(), actor, 10, CreateInput{Description: "5000₽ на благотворительность"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Stake.Description != "5000₽ на благотворительность" {
		t.Errorf("expected description to match, got %q", view.Stake.Description)
	}
	if view.Stake.Status != StatusActive {
		t.Errorf("expected status active, got %q", view.Stake.Status)
	}
}

func TestCreate_EmptyDescription(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)
	actor := users.User{ID: 1}

	_, err := svc.Create(context.Background(), actor, 10, CreateInput{Description: "   "})
	if err == nil {
		t.Fatal("expected error for empty description")
	}
}

func TestCreate_NotOwner(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)
	actor := users.User{ID: 2} // buddy, not owner

	_, err := svc.Create(context.Background(), actor, 10, CreateInput{Description: "test"})
	if err == nil {
		t.Fatal("expected error for non-owner")
	}
	if !isErr(err, ErrNotGoalOwner) {
		t.Errorf("expected ErrNotGoalOwner, got %v", err)
	}
}

func TestCreate_GoalNotActive(t *testing.T) {
	repo := newMockRepo(1, 2, "pending_buddy_acceptance")
	svc := NewService(repo)
	actor := users.User{ID: 1}

	_, err := svc.Create(context.Background(), actor, 10, CreateInput{Description: "test"})
	if err == nil {
		t.Fatal("expected error for inactive goal")
	}
	if !isErr(err, ErrGoalNotActive) {
		t.Errorf("expected ErrGoalNotActive, got %v", err)
	}
}

func TestCancel_Success(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)
	actor := users.User{ID: 1}

	view, _ := svc.Create(context.Background(), actor, 10, CreateInput{Description: "test"})

	err := svc.Cancel(context.Background(), actor, view.Stake.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := repo.stakes[view.Stake.ID]
	if s.Status != StatusCancelled {
		t.Errorf("expected cancelled, got %q", s.Status)
	}
}

func TestCancel_NotOwner(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)
	owner := users.User{ID: 1}
	buddy := users.User{ID: 2}

	view, _ := svc.Create(context.Background(), owner, 10, CreateInput{Description: "test"})

	err := svc.Cancel(context.Background(), buddy, view.Stake.ID)
	if err == nil {
		t.Fatal("expected error for non-owner cancel")
	}
	if !isErr(err, ErrNotGoalOwner) {
		t.Errorf("expected ErrNotGoalOwner, got %v", err)
	}
}

func TestCancel_NotActive(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)
	actor := users.User{ID: 1}

	view, _ := svc.Create(context.Background(), actor, 10, CreateInput{Description: "test"})
	_ = svc.Cancel(context.Background(), actor, view.Stake.ID)

	err := svc.Cancel(context.Background(), actor, view.Stake.ID)
	if err == nil {
		t.Fatal("expected error for already cancelled")
	}
	if !isErr(err, ErrStakeNotActive) {
		t.Errorf("expected ErrStakeNotActive, got %v", err)
	}
}

func TestForfeit_Success(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)
	owner := users.User{ID: 1}
	buddy := users.User{ID: 2}

	created, _ := svc.Create(context.Background(), owner, 10, CreateInput{Description: "test"})

	view, err := svc.Forfeit(context.Background(), buddy, created.Stake.ID, ForfeitInput{Reason: "не сделал"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Stake.Status != StatusForfeited {
		t.Errorf("expected forfeited, got %q", view.Stake.Status)
	}
	if view.Forfeiture == nil {
		t.Fatal("expected forfeiture record")
	}
	if view.Forfeiture.Reason != "не сделал" {
		t.Errorf("expected reason, got %q", view.Forfeiture.Reason)
	}
}

func TestForfeit_NotBuddy(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)
	owner := users.User{ID: 1}

	created, _ := svc.Create(context.Background(), owner, 10, CreateInput{Description: "test"})

	_, err := svc.Forfeit(context.Background(), owner, created.Stake.ID, ForfeitInput{})
	if err == nil {
		t.Fatal("expected error for non-buddy forfeit")
	}
	if !isErr(err, ErrNotBuddy) {
		t.Errorf("expected ErrNotBuddy, got %v", err)
	}
}

func TestForfeit_NotActive(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)
	owner := users.User{ID: 1}
	buddy := users.User{ID: 2}

	created, _ := svc.Create(context.Background(), owner, 10, CreateInput{Description: "test"})
	_ = svc.Cancel(context.Background(), owner, created.Stake.ID)

	_, err := svc.Forfeit(context.Background(), buddy, created.Stake.ID, ForfeitInput{})
	if err == nil {
		t.Fatal("expected error for inactive stake")
	}
	if !isErr(err, ErrStakeNotActive) {
		t.Errorf("expected ErrStakeNotActive, got %v", err)
	}
}

func TestListForGoal_OnlyParticipants(t *testing.T) {
	repo := newMockRepo(1, 2, "active")
	svc := NewService(repo)
	stranger := users.User{ID: 99}

	_, err := svc.ListForGoal(context.Background(), stranger, 10)
	if err == nil {
		t.Fatal("expected error for non-participant")
	}
}

func isErr(err, target error) bool {
	for err != nil {
		if err == target {
			return true
		}
		err = unwrap(err)
	}
	return false
}

func unwrap(err error) error {
	u, ok := err.(interface{ Unwrap() error })
	if !ok {
		return nil
	}
	return u.Unwrap()
}
