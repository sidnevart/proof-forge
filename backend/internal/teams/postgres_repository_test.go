package teams_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sidnevart/proof-forge/backend/internal/teams"
	"github.com/sidnevart/proof-forge/backend/testutil"
)

// insertTestUser inserts a minimal users row and returns the new id.
// Tests do not need email-uniqueness across runs because the harness
// TRUNCATEs everything in OpenIntegrationPool.
func insertTestUser(t *testing.T, pool *pgxpool.Pool, email string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (email, display_name) VALUES ($1, $2) RETURNING id`,
		email, email,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insertTestUser: %v", err)
	}
	return id
}

func newRepo(t *testing.T) (*teams.PostgresRepository, *pgxpool.Pool) {
	pool := testutil.OpenIntegrationPool(t)
	return teams.NewPostgresRepository(pool), pool
}

func TestPostgres_CreateTeam_LeadMembershipCreatedAtomically(t *testing.T) {
	repo, pool := newRepo(t)
	ctx := context.Background()
	uid := insertTestUser(t, pool, "lead@example.com")

	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	d, err := repo.CreateTeam(ctx, teams.CreateTeamParams{
		LeadUserID:  uid,
		Name:        "Alpha",
		InviteCode:  "AAAAAAAAAAAA",
		MemberLimit: 25,
		AIMode:      teams.AIModeMetadataOnly,
		CreatedAt:   now,
	})
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if d.MyMembership.Role != teams.RoleLead {
		t.Errorf("Role = %v, want lead", d.MyMembership.Role)
	}

	var leadCount int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM team_memberships
		   WHERE team_id = $1 AND role = 'lead' AND status = 'active'`,
		d.Team.ID,
	).Scan(&leadCount); err != nil {
		t.Fatal(err)
	}
	if leadCount != 1 {
		t.Errorf("active lead count = %d, want 1", leadCount)
	}
}

func TestPostgres_PartialUniqueIndex_RejectsSecondLead(t *testing.T) {
	repo, pool := newRepo(t)
	ctx := context.Background()
	uid := insertTestUser(t, pool, "lead@example.com")
	uid2 := insertTestUser(t, pool, "second@example.com")

	d, err := repo.CreateTeam(ctx, teams.CreateTeamParams{
		LeadUserID: uid, Name: "Alpha", InviteCode: "BBBBBBBBBBBB",
		MemberLimit: 25, AIMode: teams.AIModeMetadataOnly,
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Add a second member, then try to "promote" them to lead — repo should
	// refuse via translateRoleViolation → ErrMustHaveLead.
	_, err = repo.JoinTeam(ctx, teams.JoinTeamParams{
		UserID: uid2, InviteCode: d.Team.InviteCode, JoinedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	err = repo.ChangeMemberRole(ctx, teams.ChangeMemberRoleParams{
		TeamID: d.Team.ID, UserID: uid2, NewRole: teams.RoleLead,
	})
	if err == nil {
		t.Fatal("expected error promoting to lead while another lead exists, got nil")
	}
}

func TestPostgres_JoinTeam_RaceCondition_HonorsLimit(t *testing.T) {
	repo, pool := newRepo(t)
	ctx := context.Background()
	leadID := insertTestUser(t, pool, "lead@example.com")

	// Create team with small member_limit to trigger contention quickly.
	// member_limit must be >= 2 (CHECK in migration), so use 2 → 1 lead + 1 slot.
	d, err := repo.CreateTeam(ctx, teams.CreateTeamParams{
		LeadUserID: leadID, Name: "Race", InviteCode: "CCCCCCCCCCCC",
		MemberLimit: 2, AIMode: teams.AIModeMetadataOnly,
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Five concurrent joiners.
	users := make([]int64, 5)
	for i := range users {
		users[i] = insertTestUser(t, pool, makeEmail(i))
	}

	var (
		wg          sync.WaitGroup
		successes   int32
		fullErrors  int32
		otherErrors int32
	)
	for _, u := range users {
		u := u
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.JoinTeam(ctx, teams.JoinTeamParams{
				UserID: u, InviteCode: d.Team.InviteCode, JoinedAt: time.Now(),
			})
			switch {
			case err == nil:
				atomic.AddInt32(&successes, 1)
			case err == teams.ErrTeamFull:
				atomic.AddInt32(&fullErrors, 1)
			default:
				atomic.AddInt32(&otherErrors, 1)
				t.Logf("unexpected join error: %v", err)
			}
		}()
	}
	wg.Wait()

	if successes != 1 {
		t.Errorf("successful joins = %d, want exactly 1 (limit=2 with 1 lead already)", successes)
	}
	if fullErrors+otherErrors != 4 {
		t.Errorf("rejected joins = %d, want 4", fullErrors+otherErrors)
	}
}

func TestPostgres_JoinTeam_ReactivatesLeftMembership(t *testing.T) {
	repo, pool := newRepo(t)
	ctx := context.Background()
	leadID := insertTestUser(t, pool, "lead@example.com")
	memberID := insertTestUser(t, pool, "member@example.com")

	d, err := repo.CreateTeam(ctx, teams.CreateTeamParams{
		LeadUserID: leadID, Name: "Return", InviteCode: "RETURN000000",
		MemberLimit: 25, AIMode: teams.AIModeMetadataOnly,
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	firstJoin, err := repo.JoinTeam(ctx, teams.JoinTeamParams{
		UserID: memberID, InviteCode: d.Team.InviteCode, JoinedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("first JoinTeam: %v", err)
	}
	if err := repo.LeaveTeam(ctx, d.Team.ID, memberID, time.Now()); err != nil {
		t.Fatalf("LeaveTeam: %v", err)
	}

	returnedAt := time.Now().Add(time.Hour)
	secondJoin, err := repo.JoinTeam(ctx, teams.JoinTeamParams{
		UserID: memberID, InviteCode: d.Team.InviteCode, JoinedAt: returnedAt,
	})
	if err != nil {
		t.Fatalf("second JoinTeam: %v", err)
	}
	if secondJoin.MyMembership.ID != firstJoin.MyMembership.ID {
		t.Fatalf("expected same membership row to reactivate, got %d want %d", secondJoin.MyMembership.ID, firstJoin.MyMembership.ID)
	}
	if secondJoin.MyMembership.Status != teams.MembershipStatusActive {
		t.Fatalf("status = %v, want active", secondJoin.MyMembership.Status)
	}
	if secondJoin.MyMembership.Role != teams.RoleMember {
		t.Fatalf("role = %v, want member", secondJoin.MyMembership.Role)
	}
	if secondJoin.MemberCount != 2 {
		t.Fatalf("member_count = %d, want 2", secondJoin.MemberCount)
	}
}

func TestPostgres_JoinTeam_RemovedMembershipCannotRejoinByCode(t *testing.T) {
	repo, pool := newRepo(t)
	ctx := context.Background()
	leadID := insertTestUser(t, pool, "lead@example.com")
	memberID := insertTestUser(t, pool, "removed@example.com")

	d, err := repo.CreateTeam(ctx, teams.CreateTeamParams{
		LeadUserID: leadID, Name: "Removed", InviteCode: "REMOVED00000",
		MemberLimit: 25, AIMode: teams.AIModeMetadataOnly,
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.JoinTeam(ctx, teams.JoinTeamParams{
		UserID: memberID, InviteCode: d.Team.InviteCode, JoinedAt: time.Now(),
	}); err != nil {
		t.Fatalf("JoinTeam: %v", err)
	}
	if err := repo.RemoveMember(ctx, teams.RemoveMemberParams{
		TeamID: d.Team.ID, UserID: memberID, At: time.Now(),
	}); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	_, err = repo.JoinTeam(ctx, teams.JoinTeamParams{
		UserID: memberID, InviteCode: d.Team.InviteCode, JoinedAt: time.Now(),
	})
	if !errors.Is(err, teams.ErrNotMember) {
		t.Fatalf("expected ErrNotMember for removed rejoin, got %v", err)
	}
}

func TestPostgres_GoalCircleOrTeamXOR_Enforced(t *testing.T) {
	repo, pool := newRepo(t)
	ctx := context.Background()
	uid := insertTestUser(t, pool, "owner@example.com")
	buddy := insertTestUser(t, pool, "buddy@example.com")

	// Create a team and a circle for the same user.
	d, err := repo.CreateTeam(ctx, teams.CreateTeamParams{
		LeadUserID: uid, Name: "T", InviteCode: "DDDDDDDDDDDD",
		MemberLimit: 25, AIMode: teams.AIModeMetadataOnly,
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	var circleID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO circles (owner_user_id, name, invite_code, member_limit)
		 VALUES ($1, 'C', 'CIRCLE-EEEE', 8) RETURNING id`,
		uid,
	).Scan(&circleID); err != nil {
		t.Fatalf("insert circle: %v", err)
	}

	// Inserting a goal with both circle_id and team_id must fail on the CHECK.
	_, err = pool.Exec(ctx,
		`INSERT INTO goals (owner_user_id, buddy_user_id, title, status, circle_id, team_id)
		 VALUES ($1, $2, 'g', 'active', $3, $4)`,
		uid, buddy, circleID, d.Team.ID,
	)
	if err == nil {
		t.Fatal("expected XOR CHECK to fire, got nil error")
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO goals (owner_user_id, buddy_user_id, title, status, team_id)
		 VALUES ($1, $2, 'team-only', 'active', $3)`,
		uid, buddy, d.Team.ID,
	)
	if err != nil {
		t.Fatalf("expected team-only goal to satisfy nullable circle_id + XOR, got %v", err)
	}
}

func TestPostgres_ArchiveTeam_HidesFromInviteCodeJoin(t *testing.T) {
	repo, pool := newRepo(t)
	ctx := context.Background()
	leadID := insertTestUser(t, pool, "lead@example.com")
	memberID := insertTestUser(t, pool, "member@example.com")

	d, err := repo.CreateTeam(ctx, teams.CreateTeamParams{
		LeadUserID: leadID, Name: "Z", InviteCode: "EEEEEEEEEEEE",
		MemberLimit: 25, AIMode: teams.AIModeMetadataOnly,
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.ArchiveTeam(ctx, d.Team.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	_, err = repo.JoinTeam(ctx, teams.JoinTeamParams{
		UserID: memberID, InviteCode: d.Team.InviteCode, JoinedAt: time.Now(),
	})
	if err != teams.ErrTeamArchived {
		t.Fatalf("expected ErrTeamArchived, got %v", err)
	}
}

func TestPostgres_LeaveTeam_OnlyLead_RepoReturnsErr(t *testing.T) {
	repo, pool := newRepo(t)
	ctx := context.Background()
	leadID := insertTestUser(t, pool, "lead@example.com")

	d, err := repo.CreateTeam(ctx, teams.CreateTeamParams{
		LeadUserID: leadID, Name: "Solo", InviteCode: "FFFFFFFFFFFF",
		MemberLimit: 25, AIMode: teams.AIModeMetadataOnly,
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	err = repo.LeaveTeam(ctx, d.Team.ID, leadID, time.Now())
	if err != teams.ErrCannotLeaveAsOnlyLead {
		t.Fatalf("expected ErrCannotLeaveAsOnlyLead, got %v", err)
	}
	// Membership row must still be active.
	mem, err := repo.GetMembership(ctx, d.Team.ID, leadID)
	if err != nil {
		t.Fatalf("expected lead still active, got %v", err)
	}
	if mem.Status != teams.MembershipStatusActive {
		t.Errorf("status = %v, want active", mem.Status)
	}
}

func makeEmail(i int) string {
	return string(rune('a'+i)) + "@example.com"
}
