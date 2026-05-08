package workspaces_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sidnevart/proof-forge/backend/internal/workspaces"
	"github.com/sidnevart/proof-forge/backend/testutil"
)

func newTestService(t *testing.T, pool *pgxpool.Pool) *workspaces.Service {
	t.Helper()
	return workspaces.NewService(workspaces.NewPostgresRepository(pool))
}

func TestCreateWorkspace_HappyPath(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	ownerID := seedUser(t, pool, "owner@test.com")

	ws, err := svc.Create(ctx, workspaces.CreateInput{
		OwnerUserID: ownerID,
		Name:        "T-Bank AI Stream",
		Slug:        "tbank-ai",
		Type:        "organization",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if ws.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if ws.Slug != "tbank-ai" {
		t.Fatalf("slug: got %q want %q", ws.Slug, "tbank-ai")
	}
	if ws.Type != workspaces.WorkspaceTypeOrganization {
		t.Fatalf("type: got %q", ws.Type)
	}
	if !ws.IsActive {
		t.Fatal("expected is_active=true")
	}
}

func TestCreateWorkspace_SlugConflict(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	ownerID := seedUser(t, pool, "o2@test.com")

	_, err := svc.Create(ctx, workspaces.CreateInput{
		OwnerUserID: ownerID,
		Name:        "First",
		Slug:        "my-workspace",
		Type:        "community",
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err = svc.Create(ctx, workspaces.CreateInput{
		OwnerUserID: ownerID,
		Name:        "Second",
		Slug:        "my-workspace",
		Type:        "organization",
	})
	if err == nil {
		t.Fatal("expected slug conflict error")
	}
}

func TestCreateWorkspace_InvalidSlug(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	ownerID := seedUser(t, pool, "o3@test.com")

	cases := []string{"AB", "-bad", "bad-", "a", "has space"}
	for _, slug := range cases {
		_, err := svc.Create(ctx, workspaces.CreateInput{
			OwnerUserID: ownerID,
			Name:        "Test",
			Slug:        slug,
			Type:        "organization",
		})
		if err == nil {
			t.Errorf("slug %q should fail validation", slug)
		}
	}
}

func TestFreezeUnfreeze(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	ownerID := seedUser(t, pool, "freeze@test.com")
	ws, _ := svc.Create(ctx, workspaces.CreateInput{
		OwnerUserID: ownerID,
		Name:        "Freeze Test",
		Slug:        "freeze-test",
		Type:        "organization",
	})

	if err := svc.Freeze(ctx, ws.ID, "pilot ended"); err != nil {
		t.Fatalf("freeze: %v", err)
	}

	got, _ := svc.GetByID(ctx, ws.ID)
	if !got.IsFrozen() {
		t.Fatal("expected frozen")
	}
	if got.IsActive {
		t.Fatal("expected is_active=false after freeze")
	}

	if err := svc.Unfreeze(ctx, ws.ID); err != nil {
		t.Fatalf("unfreeze: %v", err)
	}
	got, _ = svc.GetByID(ctx, ws.ID)
	if got.IsFrozen() {
		t.Fatal("expected not frozen after unfreeze")
	}
}

func TestCheckSlugAvailable(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	ownerID := seedUser(t, pool, "slug@test.com")
	_, _ = svc.Create(ctx, workspaces.CreateInput{
		OwnerUserID: ownerID,
		Name:        "Taken",
		Slug:        "taken-slug",
		Type:        "community",
	})

	avail, _ := svc.CheckSlugAvailable(ctx, "taken-slug")
	if avail {
		t.Fatal("should not be available")
	}

	avail, _ = svc.CheckSlugAvailable(ctx, "free-slug")
	if !avail {
		t.Fatal("should be available")
	}
}

// seedUser inserts a minimal user row and returns its ID.
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
