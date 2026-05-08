package contracts_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sidnevart/proof-forge/backend/internal/contracts"
	"github.com/sidnevart/proof-forge/backend/testutil"
)

func newTestService(t *testing.T, pool *pgxpool.Pool) *contracts.Service {
	t.Helper()
	return contracts.NewService(contracts.NewPostgresRepository(pool))
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

func seedGoal(t *testing.T, pool *pgxpool.Pool, userID int64) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO goals (user_id, title) VALUES ($1, $2) RETURNING id`,
		userID, "Test Goal",
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed goal: %v", err)
	}
	return id
}

func futureTime(d time.Duration) time.Time {
	return time.Now().Add(d)
}

func TestCreateContract_NoBuddy_AutoActivates(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	userID := seedUser(t, pool, "solo@test.com")
	goalID := seedGoal(t, pool, userID)

	c, err := svc.Create(ctx, contracts.CreateInput{
		GoalID:      goalID,
		UserID:      userID,
		WhatToProve: "Finish chapter 3",
		HowToProve:  "Screenshot of completed exercises",
		DueAt:       futureTime(48 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if c.Status != contracts.StatusActive {
		t.Fatalf("status: got %q want active (no buddy → auto-active)", c.Status)
	}
}

func TestCreateContract_WithBuddy_IsPending(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	userID := seedUser(t, pool, "owner@test.com")
	buddyID := seedUser(t, pool, "buddy@test.com")
	goalID := seedGoal(t, pool, userID)

	c, err := svc.Create(ctx, contracts.CreateInput{
		GoalID:      goalID,
		UserID:      userID,
		BuddyUserID: &buddyID,
		WhatToProve: "Ship feature X",
		HowToProve:  "PR link",
		DueAt:       futureTime(72 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.Status != contracts.StatusPending {
		t.Fatalf("status: got %q want pending (has buddy → needs activation)", c.Status)
	}
}

func TestCreateContract_PastDueAt_Fails(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	userID := seedUser(t, pool, "past@test.com")
	goalID := seedGoal(t, pool, userID)

	_, err := svc.Create(ctx, contracts.CreateInput{
		GoalID:      goalID,
		UserID:      userID,
		WhatToProve: "Something",
		DueAt:       time.Now().Add(-time.Hour),
	})
	if err == nil {
		t.Fatal("expected error for past due_at")
	}
}

func TestCreateContract_EmptyWhatToProve_Fails(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	userID := seedUser(t, pool, "empty@test.com")
	goalID := seedGoal(t, pool, userID)

	_, err := svc.Create(ctx, contracts.CreateInput{
		GoalID:      goalID,
		UserID:      userID,
		WhatToProve: "   ",
		DueAt:       futureTime(24 * time.Hour),
	})
	if err == nil {
		t.Fatal("expected error for empty what_to_prove")
	}
}

func TestActivate_ByBuddy_Succeeds(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	userID := seedUser(t, pool, "act-owner@test.com")
	buddyID := seedUser(t, pool, "act-buddy@test.com")
	goalID := seedGoal(t, pool, userID)

	c, _ := svc.Create(ctx, contracts.CreateInput{
		GoalID:      goalID,
		UserID:      userID,
		BuddyUserID: &buddyID,
		WhatToProve: "Deploy to prod",
		DueAt:       futureTime(48 * time.Hour),
	})

	if err := svc.Activate(ctx, c.ID, buddyID); err != nil {
		t.Fatalf("activate: %v", err)
	}

	got, _ := svc.GetByID(ctx, c.ID)
	if got.Status != contracts.StatusActive {
		t.Fatalf("status: got %q want active", got.Status)
	}
}

func TestActivate_ByNonBuddy_Forbidden(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	userID := seedUser(t, pool, "forbid-owner@test.com")
	buddyID := seedUser(t, pool, "forbid-buddy@test.com")
	strangerID := seedUser(t, pool, "forbid-stranger@test.com")
	goalID := seedGoal(t, pool, userID)

	c, _ := svc.Create(ctx, contracts.CreateInput{
		GoalID:      goalID,
		UserID:      userID,
		BuddyUserID: &buddyID,
		WhatToProve: "Something",
		DueAt:       futureTime(48 * time.Hour),
	})

	if err := svc.Activate(ctx, c.ID, strangerID); err == nil {
		t.Fatal("expected forbidden error")
	}
}

func TestFulfill_ByOwner_Succeeds(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	userID := seedUser(t, pool, "fulfill-owner@test.com")
	goalID := seedGoal(t, pool, userID)

	c, _ := svc.Create(ctx, contracts.CreateInput{
		GoalID:      goalID,
		UserID:      userID,
		WhatToProve: "Write tests",
		DueAt:       futureTime(24 * time.Hour),
	})

	if err := svc.Fulfill(ctx, c.ID, userID); err != nil {
		t.Fatalf("fulfill: %v", err)
	}

	got, _ := svc.GetByID(ctx, c.ID)
	if got.Status != contracts.StatusFulfilled {
		t.Fatalf("status: got %q want fulfilled", got.Status)
	}
	if got.FulfilledAt == nil {
		t.Fatal("expected fulfilled_at to be set")
	}
}

func TestCancel_ByOwner_Succeeds(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	ctx := context.Background()

	userID := seedUser(t, pool, "cancel-owner@test.com")
	goalID := seedGoal(t, pool, userID)

	c, _ := svc.Create(ctx, contracts.CreateInput{
		GoalID:      goalID,
		UserID:      userID,
		WhatToProve: "Ship v2",
		DueAt:       futureTime(24 * time.Hour),
	})

	if err := svc.Cancel(ctx, c.ID, userID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	got, _ := svc.GetByID(ctx, c.ID)
	if got.Status != contracts.StatusCancelled {
		t.Fatalf("status: got %q want cancelled", got.Status)
	}
}

func TestMarkBrokenOverdue(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	svc := newTestService(t, pool)
	repo := contracts.NewPostgresRepository(pool)
	ctx := context.Background()

	userID := seedUser(t, pool, "overdue@test.com")
	goalID := seedGoal(t, pool, userID)

	// Create a contract with a past due_at by inserting directly.
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO proof_contracts (goal_id, user_id, what_to_prove, due_at, status)
		VALUES ($1, $2, $3, $4, 'active') RETURNING id`,
		goalID, userID, "Overdue proof", time.Now().Add(-time.Hour),
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed overdue contract: %v", err)
	}

	n, err := repo.MarkBrokenOverdue(ctx)
	if err != nil {
		t.Fatalf("mark broken: %v", err)
	}
	if n == 0 {
		t.Fatal("expected at least one broken contract")
	}

	got, _ := svc.GetByID(ctx, id)
	if got.Status != contracts.StatusBroken {
		t.Fatalf("status: got %q want broken", got.Status)
	}
}
