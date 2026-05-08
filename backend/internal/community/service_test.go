package community_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sidnevart/proof-forge/backend/internal/community"
	"github.com/sidnevart/proof-forge/backend/testutil"
)

func newTestService(t *testing.T, pool *pgxpool.Pool) *community.Service {
	t.Helper()
	return community.NewService(community.NewPostgresRepository(pool))
}

func seedUser(t *testing.T, pool *pgxpool.Pool, email string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (email, display_name) VALUES ($1, $2) RETURNING id`,
		email, "Test User",
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id
}

func TestCreateCommunitySpace(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	ownerID := seedUser(t, pool, "cs-owner@test.com")

	cs, err := svc.Create(ctx, community.CreateInput{
		OwnerUserID: ownerID,
		Name:        "ML Community",
		Slug:        "ml-community",
		IsPublic:    true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if cs.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if cs.InviteCode == "" {
		t.Fatal("expected non-empty invite code")
	}
}

func TestJoinCommunitySpace(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	ownerID := seedUser(t, pool, "cs-owner2@test.com")
	memberID := seedUser(t, pool, "cs-member@test.com")

	cs, err := svc.Create(ctx, community.CreateInput{
		OwnerUserID: ownerID,
		Name:        "Backend Club",
		Slug:        "backend-club",
		IsPublic:    true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	m, err := svc.Join(ctx, memberID, cs.InviteCode)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if m.UserID != memberID {
		t.Fatalf("membership user_id: got %d want %d", m.UserID, memberID)
	}

	// Double join should fail.
	_, err = svc.Join(ctx, memberID, cs.InviteCode)
	if err == nil {
		t.Fatal("expected error on double join")
	}
}

func TestJoinInvalidCode(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	userID := seedUser(t, pool, "bad-join@test.com")
	_, err := svc.Join(ctx, userID, "INVALID_CODE")
	if err == nil {
		t.Fatal("expected invalid invite code error")
	}
}
