package teams

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// ──────────────────────────────────────────────────────────────────────
// fakeRepo — in-memory Repository that mirrors Postgres invariants.
// ──────────────────────────────────────────────────────────────────────

type fakeRepo struct {
	mu          sync.Mutex
	teams       map[int64]Team
	memberships []Membership // flat list keyed by (team_id, user_id) implicitly
	codes       map[string]int64
	nextTeamID  int64
	nextMemID   int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		teams:      map[int64]Team{},
		codes:      map[string]int64{},
		nextTeamID: 1,
		nextMemID:  1,
	}
}

func (r *fakeRepo) findMembership(teamID, userID int64) (int, *Membership) {
	for i := range r.memberships {
		if r.memberships[i].TeamID == teamID && r.memberships[i].UserID == userID {
			return i, &r.memberships[i]
		}
	}
	return -1, nil
}

func (r *fakeRepo) activeMembershipCount(teamID int64) int {
	n := 0
	for _, m := range r.memberships {
		if m.TeamID == teamID && m.Status == MembershipStatusActive {
			n++
		}
	}
	return n
}

func (r *fakeRepo) activeLeadCount(teamID int64, exceptUserID int64) int {
	n := 0
	for _, m := range r.memberships {
		if m.TeamID != teamID || m.Status != MembershipStatusActive || m.Role != RoleLead {
			continue
		}
		if m.UserID == exceptUserID {
			continue
		}
		n++
	}
	return n
}

func (r *fakeRepo) buildDetail(teamID, callerUserID int64) (Detail, error) {
	team, ok := r.teams[teamID]
	if !ok {
		return Detail{}, ErrTeamNotFound
	}
	_, mem := r.findMembership(teamID, callerUserID)
	if mem == nil || mem.Status != MembershipStatusActive {
		return Detail{}, ErrNotMember
	}
	return Detail{
		Team:         team,
		MyMembership: *mem,
		MemberCount:  r.activeMembershipCount(teamID),
	}, nil
}

func (r *fakeRepo) CreateTeam(ctx context.Context, params CreateTeamParams) (Detail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.codes[params.InviteCode]; exists {
		// In real Postgres this is a UNIQUE violation; service generates fresh
		// codes each call so collision is essentially impossible in tests.
		return Detail{}, errors.New("invite_code collision (fake)")
	}

	team := Team{
		ID:          r.nextTeamID,
		LeadUserID:  params.LeadUserID,
		Name:        params.Name,
		InviteCode:  params.InviteCode,
		MemberLimit: params.MemberLimit,
		AIMode:      params.AIMode,
		CreatedAt:   params.CreatedAt,
		UpdatedAt:   params.CreatedAt,
	}
	r.nextTeamID++
	r.teams[team.ID] = team
	r.codes[team.InviteCode] = team.ID

	mem := Membership{
		ID:        r.nextMemID,
		TeamID:    team.ID,
		UserID:    params.LeadUserID,
		Role:      RoleLead,
		Status:    MembershipStatusActive,
		AIConsent: false,
		Timezone:  "Europe/Moscow",
		JoinedAt:  params.CreatedAt,
	}
	r.nextMemID++
	r.memberships = append(r.memberships, mem)

	return Detail{Team: team, MyMembership: mem, MemberCount: 1}, nil
}

func (r *fakeRepo) ListMyTeams(ctx context.Context, userID int64) ([]Detail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := []Detail{}
	for _, m := range r.memberships {
		if m.UserID != userID || m.Status != MembershipStatusActive {
			continue
		}
		team := r.teams[m.TeamID]
		out = append(out, Detail{
			Team:         team,
			MyMembership: m,
			MemberCount:  r.activeMembershipCount(m.TeamID),
		})
	}
	return out, nil
}

func (r *fakeRepo) GetTeamForUser(ctx context.Context, teamID, userID int64) (Detail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buildDetail(teamID, userID)
}

func (r *fakeRepo) JoinTeam(ctx context.Context, params JoinTeamParams) (Detail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	teamID, ok := r.codes[params.InviteCode]
	if !ok {
		return Detail{}, ErrInvalidInviteCode
	}
	team, ok := r.teams[teamID]
	if !ok || team.ArchivedAt != nil {
		return Detail{}, ErrTeamArchived
	}

	if _, existing := r.findMembership(teamID, params.UserID); existing != nil {
		if existing.Status == MembershipStatusActive {
			// Idempotent — return current detail without inserting.
			return Detail{
				Team:         team,
				MyMembership: *existing,
				MemberCount:  r.activeMembershipCount(teamID),
			}, nil
		}
		if existing.Status == MembershipStatusRemoved {
			return Detail{}, ErrNotMember
		}
		// Re-activate previously left/removed membership as a member.
		existing.Status = MembershipStatusActive
		existing.Role = RoleMember
		existing.JoinedAt = params.JoinedAt
		existing.LeftAt = nil
		// Fall through to capacity check via re-counting.
		if r.activeMembershipCount(teamID) > team.MemberLimit {
			existing.Status = MembershipStatusLeft // revert
			return Detail{}, ErrTeamFull
		}
		return Detail{
			Team:         team,
			MyMembership: *existing,
			MemberCount:  r.activeMembershipCount(teamID),
		}, nil
	}

	if r.activeMembershipCount(teamID) >= team.MemberLimit {
		return Detail{}, ErrTeamFull
	}

	mem := Membership{
		ID:        r.nextMemID,
		TeamID:    teamID,
		UserID:    params.UserID,
		Role:      RoleMember,
		Status:    MembershipStatusActive,
		AIConsent: false,
		Timezone:  "Europe/Moscow",
		JoinedAt:  params.JoinedAt,
	}
	r.nextMemID++
	r.memberships = append(r.memberships, mem)

	return Detail{
		Team:         team,
		MyMembership: mem,
		MemberCount:  r.activeMembershipCount(teamID),
	}, nil
}

func (r *fakeRepo) ChangeMemberRole(ctx context.Context, params ChangeMemberRoleParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, mem := r.findMembership(params.TeamID, params.UserID)
	if mem == nil || mem.Status != MembershipStatusActive {
		return ErrNotMember
	}
	// Promotion to lead would create a second active lead — refuse (mirrors
	// the partial unique index uq_team_one_lead).
	if params.NewRole == RoleLead && mem.Role != RoleLead {
		if r.activeLeadCount(params.TeamID, mem.UserID) >= 1 {
			return ErrMustHaveLead
		}
	}
	// Demotion of the only lead would orphan the team — refuse.
	if mem.Role == RoleLead && params.NewRole != RoleLead {
		if r.activeLeadCount(params.TeamID, mem.UserID) == 0 {
			return ErrMustHaveLead
		}
	}
	mem.Role = params.NewRole
	return nil
}

func (r *fakeRepo) RemoveMember(ctx context.Context, params RemoveMemberParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, mem := r.findMembership(params.TeamID, params.UserID)
	if mem == nil || mem.Status != MembershipStatusActive {
		return ErrNotMember
	}
	mem.Status = MembershipStatusRemoved
	mem.LeftAt = &params.At
	return nil
}

func (r *fakeRepo) LeaveTeam(ctx context.Context, teamID, userID int64, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, mem := r.findMembership(teamID, userID)
	if mem == nil || mem.Status != MembershipStatusActive {
		return ErrNotMember
	}
	if mem.Role == RoleLead && r.activeLeadCount(teamID, userID) == 0 {
		return ErrCannotLeaveAsOnlyLead
	}
	mem.Status = MembershipStatusLeft
	mem.LeftAt = &at
	return nil
}

func (r *fakeRepo) SetAIConsent(ctx context.Context, teamID, userID int64, consent bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, mem := r.findMembership(teamID, userID)
	if mem == nil || mem.Status != MembershipStatusActive {
		return ErrNotMember
	}
	mem.AIConsent = consent
	return nil
}

func (r *fakeRepo) RegenerateInviteCode(ctx context.Context, teamID int64, newCode string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	team, ok := r.teams[teamID]
	if !ok {
		return ErrTeamNotFound
	}
	if _, exists := r.codes[newCode]; exists {
		return errors.New("invite_code collision (fake)")
	}
	delete(r.codes, team.InviteCode)
	team.InviteCode = newCode
	r.teams[teamID] = team
	r.codes[newCode] = teamID
	return nil
}

func (r *fakeRepo) ArchiveTeam(ctx context.Context, teamID int64, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	team, ok := r.teams[teamID]
	if !ok {
		return ErrTeamNotFound
	}
	team.ArchivedAt = &at
	r.teams[teamID] = team
	return nil
}

func (r *fakeRepo) GetMembership(ctx context.Context, teamID, userID int64) (Membership, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, mem := r.findMembership(teamID, userID)
	if mem == nil || mem.Status != MembershipStatusActive {
		return Membership{}, ErrNotMember
	}
	return *mem, nil
}

// ──────────────────────────────────────────────────────────────────────
// Test helpers
// ──────────────────────────────────────────────────────────────────────

func newTestService(t *testing.T) (*Service, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	svc := NewService(repo, WithClock(func() time.Time {
		return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	}))
	return svc, repo
}

// makeTeam — convenience wrapper that always returns a usable Detail.
func makeTeam(t *testing.T, svc *Service, leadUserID int64, name string) Detail {
	t.Helper()
	d, err := svc.CreateTeam(context.Background(), leadUserID, name, "")
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	return d
}

// ──────────────────────────────────────────────────────────────────────
// Test cases
// ──────────────────────────────────────────────────────────────────────

func TestCreateTeamHappyPath(t *testing.T) {
	svc, _ := newTestService(t)

	d, err := svc.CreateTeam(context.Background(), 1, "ML team", "")
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if d.Team.Name != "ML team" {
		t.Errorf("Name = %q, want %q", d.Team.Name, "ML team")
	}
	if d.MyMembership.Role != RoleLead {
		t.Errorf("MyMembership.Role = %v, want lead", d.MyMembership.Role)
	}
	if d.MemberCount != 1 {
		t.Errorf("MemberCount = %d, want 1", d.MemberCount)
	}
	if d.Team.InviteCode == "" {
		t.Error("InviteCode must be generated")
	}
	if d.Team.AIMode != AIModeMetadataOnly {
		t.Errorf("default AIMode = %v, want metadata-only", d.Team.AIMode)
	}
	if d.Team.MemberLimit != DefaultMemberLimit {
		t.Errorf("MemberLimit = %d, want %d", d.Team.MemberLimit, DefaultMemberLimit)
	}
}

func TestCreateTeamRejectsEmptyName(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.CreateTeam(context.Background(), 1, "  ", "")
	if !errors.Is(err, ErrInvalidTeamName) {
		t.Fatalf("expected ErrInvalidTeamName, got %v", err)
	}
}

func TestCreateTeamRejectsBadAIMode(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.CreateTeam(context.Background(), 1, "x", "hyper")
	if !errors.Is(err, ErrInvalidAIMode) {
		t.Fatalf("expected ErrInvalidAIMode, got %v", err)
	}
}

func TestJoinTeamByInviteCodeHappyPath(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")

	d, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode)
	if err != nil {
		t.Fatalf("JoinTeam: %v", err)
	}
	if d.MyMembership.Role != RoleMember {
		t.Errorf("Role = %v, want member", d.MyMembership.Role)
	}
	if d.MemberCount != 2 {
		t.Errorf("MemberCount = %d, want 2", d.MemberCount)
	}
}

func TestJoinTeamRejectsBadCode(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.JoinTeam(context.Background(), 2, "NOPE")
	if !errors.Is(err, ErrInvalidInviteCode) {
		t.Fatalf("expected ErrInvalidInviteCode, got %v", err)
	}
}

func TestJoinTeamRejectsEmptyCode(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.JoinTeam(context.Background(), 2, "  ")
	if !errors.Is(err, ErrInvalidInviteCode) {
		t.Fatalf("expected ErrInvalidInviteCode, got %v", err)
	}
}

func TestJoinTeamRejectsArchived(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")
	if err := svc.ArchiveTeam(context.Background(), 1, t1.Team.ID); err != nil {
		t.Fatalf("ArchiveTeam: %v", err)
	}
	_, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode)
	if !errors.Is(err, ErrTeamArchived) {
		t.Fatalf("expected ErrTeamArchived, got %v", err)
	}
}

func TestJoinTeamRejectsWhenFull(t *testing.T) {
	svc, repo := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")
	// Override member_limit on the fake to trigger capacity quickly.
	team := repo.teams[t1.Team.ID]
	team.MemberLimit = 2
	repo.teams[t1.Team.ID] = team

	if _, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode); err != nil {
		t.Fatalf("first join failed: %v", err)
	}
	_, err := svc.JoinTeam(context.Background(), 3, t1.Team.InviteCode)
	if !errors.Is(err, ErrTeamFull) {
		t.Fatalf("expected ErrTeamFull, got %v", err)
	}
}

func TestJoinTeamIdempotentForExistingMember(t *testing.T) {
	svc, repo := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")
	if _, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode); err != nil {
		t.Fatal(err)
	}
	d, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode)
	if err != nil {
		t.Fatalf("second join: %v", err)
	}
	if d.MemberCount != 2 {
		t.Errorf("MemberCount = %d, want 2 (no duplicate)", d.MemberCount)
	}
	// And confirm there is exactly one membership row for user 2.
	count := 0
	for _, m := range repo.memberships {
		if m.TeamID == t1.Team.ID && m.UserID == 2 {
			count++
		}
	}
	if count != 1 {
		t.Errorf("memberships(team=%d,user=2) rows = %d, want 1", t1.Team.ID, count)
	}
}

func TestJoinTeamAllowsLeftMemberToRejoin(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")
	if _, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode); err != nil {
		t.Fatalf("initial JoinTeam: %v", err)
	}
	if err := svc.LeaveTeam(context.Background(), 2, t1.Team.ID); err != nil {
		t.Fatalf("LeaveTeam: %v", err)
	}

	d, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode)
	if err != nil {
		t.Fatalf("rejoin after leave: %v", err)
	}
	if d.MyMembership.Status != MembershipStatusActive {
		t.Fatalf("status = %v, want active", d.MyMembership.Status)
	}
}

func TestJoinTeamRejectsRemovedMemberRejoin(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")
	if _, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode); err != nil {
		t.Fatalf("initial JoinTeam: %v", err)
	}
	if err := svc.RemoveMember(context.Background(), 1, t1.Team.ID, 2); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	_, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode)
	if !errors.Is(err, ErrNotMember) {
		t.Fatalf("expected ErrNotMember for removed member rejoin, got %v", err)
	}
}

func TestChangeMemberRolePromoteToTrusted(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")
	if _, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode); err != nil {
		t.Fatal(err)
	}
	if err := svc.ChangeMemberRole(context.Background(), 1, t1.Team.ID, 2, RoleTrustedApprover); err != nil {
		t.Fatalf("ChangeMemberRole: %v", err)
	}
	d, _ := svc.GetTeamForUser(context.Background(), t1.Team.ID, 2)
	if d.MyMembership.Role != RoleTrustedApprover {
		t.Errorf("Role after promote = %v, want trusted_approver", d.MyMembership.Role)
	}
	if !CanApproveProof(d.MyMembership) {
		t.Error("trusted should be able to approve")
	}
}

func TestChangeMemberRoleNonLeadRejected(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")
	if _, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode); err != nil {
		t.Fatal(err)
	}
	err := svc.ChangeMemberRole(context.Background(), 2, t1.Team.ID, 1, RoleMember)
	if !errors.Is(err, ErrNotLead) {
		t.Fatalf("expected ErrNotLead, got %v", err)
	}
}

func TestChangeMemberRoleCannotDemoteSelf(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")
	err := svc.ChangeMemberRole(context.Background(), 1, t1.Team.ID, 1, RoleMember)
	if !errors.Is(err, ErrCannotDemoteSelf) {
		t.Fatalf("expected ErrCannotDemoteSelf, got %v", err)
	}
}

func TestRemoveMemberHappyPath(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")
	if _, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode); err != nil {
		t.Fatal(err)
	}
	if err := svc.RemoveMember(context.Background(), 1, t1.Team.ID, 2); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	if _, err := svc.GetTeamForUser(context.Background(), t1.Team.ID, 2); !errors.Is(err, ErrNotMember) {
		t.Fatalf("expected ErrNotMember after remove, got %v", err)
	}
}

func TestRemoveMemberCannotRemoveSelfAsLead(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")
	err := svc.RemoveMember(context.Background(), 1, t1.Team.ID, 1)
	if !errors.Is(err, ErrCannotRemoveSelfAsLead) {
		t.Fatalf("expected ErrCannotRemoveSelfAsLead, got %v", err)
	}
}

func TestLeaveTeamMember(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")
	if _, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode); err != nil {
		t.Fatal(err)
	}
	if err := svc.LeaveTeam(context.Background(), 2, t1.Team.ID); err != nil {
		t.Fatalf("LeaveTeam: %v", err)
	}
	if _, err := svc.GetTeamForUser(context.Background(), t1.Team.ID, 2); !errors.Is(err, ErrNotMember) {
		t.Fatalf("expected ErrNotMember after leave, got %v", err)
	}
}

func TestLeaveTeamCannotLeaveAsOnlyLead(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")
	err := svc.LeaveTeam(context.Background(), 1, t1.Team.ID)
	if !errors.Is(err, ErrCannotLeaveAsOnlyLead) {
		t.Fatalf("expected ErrCannotLeaveAsOnlyLead, got %v", err)
	}
}

func TestSetAIConsentSelfOnly(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")
	if _, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetMyAIConsent(context.Background(), 2, t1.Team.ID, 2, true); err != nil {
		t.Fatalf("SetMyAIConsent self: %v", err)
	}
	err := svc.SetMyAIConsent(context.Background(), 1, t1.Team.ID, 2, false)
	if !errors.Is(err, ErrCannotChangeOthersAIConsent) {
		t.Fatalf("expected ErrCannotChangeOthersAIConsent, got %v", err)
	}
}

func TestRegenerateInviteCodeOnlyLead(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")
	if _, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode); err != nil {
		t.Fatal(err)
	}
	newCode, err := svc.RegenerateInvite(context.Background(), 1, t1.Team.ID)
	if err != nil {
		t.Fatalf("RegenerateInvite: %v", err)
	}
	if newCode == t1.Team.InviteCode {
		t.Error("new code must differ from old")
	}
	_, err = svc.RegenerateInvite(context.Background(), 2, t1.Team.ID)
	if !errors.Is(err, ErrNotLead) {
		t.Fatalf("expected ErrNotLead, got %v", err)
	}
}

func TestArchiveTeamOnlyLead(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")
	if _, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode); err != nil {
		t.Fatal(err)
	}
	if err := svc.ArchiveTeam(context.Background(), 2, t1.Team.ID); !errors.Is(err, ErrNotLead) {
		t.Fatalf("expected ErrNotLead from member, got %v", err)
	}
	if err := svc.ArchiveTeam(context.Background(), 1, t1.Team.ID); err != nil {
		t.Fatalf("ArchiveTeam by lead: %v", err)
	}
	_, err := svc.JoinTeam(context.Background(), 3, t1.Team.InviteCode)
	if !errors.Is(err, ErrTeamArchived) {
		t.Fatalf("expected ErrTeamArchived after archive, got %v", err)
	}
}

func TestListMyTeamsExcludesNonActiveMemberships(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := makeTeam(t, svc, 1, "alpha")
	t2 := makeTeam(t, svc, 1, "beta")
	if _, err := svc.JoinTeam(context.Background(), 2, t1.Team.InviteCode); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.JoinTeam(context.Background(), 2, t2.Team.InviteCode); err != nil {
		t.Fatal(err)
	}
	// User 2 leaves alpha.
	if err := svc.LeaveTeam(context.Background(), 2, t1.Team.ID); err != nil {
		t.Fatal(err)
	}
	teams, err := svc.ListMyTeams(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(teams) != 1 {
		t.Fatalf("ListMyTeams = %d, want 1 (only beta active)", len(teams))
	}
	if teams[0].Team.ID != t2.Team.ID {
		t.Errorf("expected beta team, got %d", teams[0].Team.ID)
	}
}

func TestGetTeamForUserNonMember(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := makeTeam(t, svc, 1, "team")
	_, err := svc.GetTeamForUser(context.Background(), t1.Team.ID, 99)
	if !errors.Is(err, ErrNotMember) {
		t.Fatalf("expected ErrNotMember, got %v", err)
	}
}
