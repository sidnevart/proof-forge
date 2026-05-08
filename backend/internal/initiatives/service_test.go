package initiatives

import (
	"context"
	"errors"
	"testing"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// ── stubs ────────────────────────────────────────────────────────────────────

type stubRepo struct {
	initiative    Initiative
	findErr       error
	joinResult    JoinResult
	joinErr       error
	isParticipant bool
	authorID      int64
	approveErr    error
	pendingProofs []PendingProof
}

func (r *stubRepo) Create(_ context.Context, _ CreateInput, _ int64) (Initiative, error) {
	return r.initiative, nil
}
func (r *stubRepo) FindByID(_ context.Context, _ int64) (Initiative, error) {
	return r.initiative, r.findErr
}
func (r *stubRepo) ListBySpace(_ context.Context, _ string, _ int64) ([]Initiative, error) {
	return nil, nil
}
func (r *stubRepo) JoinOrGet(_ context.Context, _, _ int64) (JoinResult, error) {
	return r.joinResult, r.joinErr
}
func (r *stubRepo) PendingProofs(_ context.Context, _, _ int64) ([]PendingProof, error) {
	return r.pendingProofs, nil
}
func (r *stubRepo) Approve(_ context.Context, _, _ int64, _ string) error {
	return r.approveErr
}
func (r *stubRepo) CheckinAuthorID(_ context.Context, _ int64) (int64, error) {
	return r.authorID, nil
}
func (r *stubRepo) IsParticipant(_ context.Context, _, _ int64) (bool, error) {
	return r.isParticipant, nil
}
func (r *stubRepo) Participants(_ context.Context, _ int64) ([]ParticipantProgress, error) {
	return nil, nil
}
func (r *stubRepo) Archive(_ context.Context, _ int64) error { return nil }

type stubMembers struct{ member bool }

func (m *stubMembers) IsTeamspaceMember(_ context.Context, _, _ int64) (bool, error) {
	return m.member, nil
}
func (m *stubMembers) IsCommunityMember(_ context.Context, _, _ int64) (bool, error) {
	return m.member, nil
}

func actor(id int64) users.User { return users.User{ID: id, Email: "u@test.com"} }

// ── tests ────────────────────────────────────────────────────────────────────

func TestCreate_NotSpaceMember(t *testing.T) {
	svc := NewService(&stubRepo{}, &stubMembers{member: false})
	_, err := svc.Create(context.Background(), actor(1), CreateInput{
		SpaceType:     "teamspace",
		SpaceID:       10,
		Title:         "Run 5km weekly",
		ProofCriteria: "Screenshot from fitness app",
	})
	if !errors.Is(err, ErrNotSpaceMember) {
		t.Fatalf("expected ErrNotSpaceMember, got %v", err)
	}
}

func TestCreate_InvalidInput(t *testing.T) {
	svc := NewService(&stubRepo{}, &stubMembers{member: true})
	_, err := svc.Create(context.Background(), actor(1), CreateInput{
		SpaceType:     "teamspace",
		SpaceID:       10,
		Title:         "Hi", // too short
		ProofCriteria: "Screenshot from fitness app",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestJoin_ArchivedInitiative(t *testing.T) {
	repo := &stubRepo{initiative: Initiative{Status: StatusArchived, SpaceType: "teamspace"}}
	teamID := int64(10)
	repo.initiative.TeamspaceID = &teamID
	svc := NewService(repo, &stubMembers{member: true})
	_, err := svc.Join(context.Background(), actor(1), 99)
	if !errors.Is(err, ErrInitiativeArchived) {
		t.Fatalf("expected ErrInitiativeArchived, got %v", err)
	}
}

func TestJoin_Idempotent(t *testing.T) {
	teamID := int64(10)
	repo := &stubRepo{
		initiative: Initiative{Status: StatusActive, SpaceType: "teamspace", TeamspaceID: &teamID},
		joinResult: JoinResult{GoalID: 42, Created: false},
	}
	svc := NewService(repo, &stubMembers{member: true})
	result, err := svc.Join(context.Background(), actor(1), 99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.GoalID != 42 {
		t.Fatalf("expected GoalID 42, got %d", result.GoalID)
	}
	if result.Created {
		t.Fatal("expected Created=false for idempotent join")
	}
}

func TestApprove_CannotApproveSelf(t *testing.T) {
	repo := &stubRepo{isParticipant: true, authorID: 7}
	svc := NewService(repo, &stubMembers{member: true})
	err := svc.Approve(context.Background(), actor(7), 1, 100, "")
	if !errors.Is(err, ErrCannotApproveSelf) {
		t.Fatalf("expected ErrCannotApproveSelf, got %v", err)
	}
}

func TestApprove_NotParticipant(t *testing.T) {
	repo := &stubRepo{isParticipant: false}
	svc := NewService(repo, &stubMembers{member: true})
	err := svc.Approve(context.Background(), actor(5), 1, 100, "nice proof")
	if !errors.Is(err, ErrNotInitiativeMember) {
		t.Fatalf("expected ErrNotInitiativeMember, got %v", err)
	}
}

func TestPendingProofs_NotParticipant(t *testing.T) {
	repo := &stubRepo{isParticipant: false}
	svc := NewService(repo, &stubMembers{member: true})
	_, err := svc.PendingProofs(context.Background(), actor(5), 1)
	if !errors.Is(err, ErrNotInitiativeMember) {
		t.Fatalf("expected ErrNotInitiativeMember, got %v", err)
	}
}
