# ProofForge Teams — Phase 0 Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Spec:** `docs/superpowers/specs/2026-05-07-proofforge-teams-habit-loop-design.md` (sections 5–7 cover this phase).

**Goal:** Заложить фундамент корпоративного режима ProofForge: новая сущность `team` с тремя ролями (`lead`, `trusted_approver`, `member`), team-membership с per-team timezone и ai_consent, расширение `goals`/`check_ins` через nullable `team_id` с XOR-инвариантом против `circle_id`, минимальный REST API и web-skeleton страниц `/teams` и `/teams/:id`.

**Phase 0 не включает:** daily-log, streak, weekly proof в команде, AI-персонализация, аналитика. Это фазы 1–4.

**Architecture:** Новый Go-пакет `internal/teams/` идёт по той же структуре, что `internal/circles/` — `domain.go` / `ports.go` / `service.go` / `postgres_repository.go` / `http_handler.go`. Расширения существующих `goals` и `check_ins` — точечные миграции с CHECK-constraints, никаких inline-изменений в их доменной логике на этой фазе. Web получает skeleton-страницы и `lib/api.ts` функции; реальные взаимодействия (создание команды, присоединение по invite-code) уже работают.

**Tech Stack:** Go 1.22+, pgx/v5, chi/v5, goose migrations, Next.js 15, React 18, Vitest, ESLint.

**Phase exit criteria** (из спеки §9):
- Тимлид создаёт команду через web и приглашает по invite-code.
- Sotrudnik присоединяется по invite-code и видит команду в списке своих.
- Тесты team authz (lead/trusted/member) — зелёные.
- Migrations up/down/up чистые в CI.
- XOR `circle_id` vs `team_id` тестами защищён.

---

## File Structure

### Migrations
- Create: `backend/migrations/00011_teams.sql`
  Таблицы `teams`, `team_memberships`, partial unique index «один lead на команду».
- Create: `backend/migrations/00012_team_extensions.sql`
  Колонки `goals.team_id`, `check_ins.team_id`, `check_ins.approver_role`, XOR-CHECK на `goals(circle_id, team_id)`.

### Backend domain — new `teams` package
- Create: `backend/internal/teams/domain.go`
  Типы `Team`, `Membership`, `Role`, `AIMode`, `Detail`, типизированные ошибки.
- Create: `backend/internal/teams/ports.go`
  Интерфейс `Repository` и param-структуры.
- Create: `backend/internal/teams/service.go`
  Use-cases: `CreateTeam`, `GetMyTeams`, `GetTeamForUser`, `JoinTeam`, `ChangeRole`, `RemoveMember`, `LeaveTeam`, `RegenerateInvite`, `ArchiveTeam`, `SetAIConsent`.
- Create: `backend/internal/teams/service_test.go`
  TDD на authz, XOR, инварианты «ровно один lead», invite-code-flow.
- Create: `backend/internal/teams/postgres_repository.go`
  pgx-implementация Repository.
- Create: `backend/internal/teams/postgres_repository_test.go`
  Integration tests на CHECK-constraints и unique-index.
- Create: `backend/internal/teams/http_handler.go`
  REST `/teams` endpoints, dto-shapes, error mapping.
- Create: `backend/internal/teams/http_test.go`
  Handler tests на 200/201/400/403/404/422.
- Create: `backend/internal/teams/authz.go`
  Чистые функции проверки прав без зависимости от БД.
- Create: `backend/internal/teams/authz_test.go`
  Полная таблица авторизации из спеки §1.

### Backend wiring
- Modify: `backend/internal/platform/app/api.go`
  Регистрация teams-handler в `/v1/teams` под auth-middleware.
- Modify: `backend/testutil/integration_db.go`
  TRUNCATE новых таблиц `teams`, `team_memberships` в test setup.

### Backend goals/check_ins — минимальные правки
- Modify: `backend/internal/goals/domain.go`
  Добавить optional `TeamID *int64` в `Goal` struct (read-only для phase 0).
- Modify: `backend/internal/goals/postgres_repository.go`
  Читать колонку `team_id` в `SELECT` (write — phase 1).
- Modify: `backend/internal/goals/service_test.go`
  Тест что существующие goal'ы остаются с `team_id = nil` после миграции.

### Frontend
- Modify: `web/lib/types.ts`
  Типы `Team`, `TeamMembership`, `TeamRole`, `TeamDetail`, `TeamAIMode`.
- Modify: `web/lib/api.ts`
  Функции: `createTeam`, `listMyTeams`, `getTeam`, `joinTeam`, `changeMemberRole`, `removeMember`, `leaveTeam`, `regenerateTeamInvite`, `archiveTeam`, `setMyAIConsent`.
- Create: `web/app/(product)/teams/page.tsx`
  Список команд пользователя + CTA «Создать команду».
- Create: `web/app/(product)/teams/page.module.css`
  Brutalist-стиль consistent с `/me`.
- Create: `web/app/(product)/teams/new/page.tsx`
  Форма создания команды (name + ai_mode select).
- Create: `web/app/(product)/teams/new/page.module.css`
- Create: `web/app/(product)/teams/[id]/page.tsx`
  Detail view: список members, invite-code copy, leave/archive.
- Create: `web/app/(product)/teams/[id]/page.module.css`
- Create: `web/app/(product)/teams/join/page.tsx`
  Страница `?code=…` для приёма invite-code.
- Create: `web/app/(product)/teams/join/page.module.css`
- Create: `web/components/product/teams-list.tsx`
  Reusable список с member_count, my_role badge.
- Create: `web/components/product/teams-list.test.tsx`
  Vitest на render и navigation.
- Modify: `web/components/product/product-nav.tsx`
  Добавить ссылку «КОМАНДЫ» рядом с «КРУГ» / «ЛЕНТА».

---

## Notes for executor

1. **TDD-first для всего, что в `service.go` / `authz.go` / `domain.go`.** Это требование CLAUDE.md. Web-композиция компонентов — без TDD, но любой нетривиальный hook/state-handler — с тестом.
2. **Миграции пишутся goose-style:** `-- +goose Up` / `-- +goose Down`. Down должен быть зеркальным.
3. **Никаких изменений в `circles` package.** Они продолжают работать параллельно.
4. **Все user-facing UI-тексты — на русском** (CLAUDE.md).
5. **Не трогаем существующий `web/lib/api.ts` auth-flow** (refresh-token, ApiError) — только добавляем новые функции в стиле существующих.

---

### Task 1: Migration `00011_teams.sql`

**Files:**
- Create: `backend/migrations/00011_teams.sql`
- Test: ручная проверка `goose up && goose down && goose up` в integration harness.

- [ ] **Step 1: Write the migration SQL**

```sql
-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS teams (
    id BIGSERIAL PRIMARY KEY,
    lead_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 80),
    invite_code TEXT NOT NULL UNIQUE,
    member_limit INTEGER NOT NULL DEFAULT 25 CHECK (member_limit BETWEEN 2 AND 100),
    ai_mode TEXT NOT NULL DEFAULT 'metadata-only'
        CHECK (ai_mode IN ('off', 'metadata-only', 'full')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS team_memberships (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('lead', 'trusted_approver', 'member')),
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'left', 'removed')),
    ai_consent BOOLEAN NOT NULL DEFAULT FALSE,
    timezone TEXT NOT NULL DEFAULT 'Europe/Moscow',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    UNIQUE (team_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_teams_lead ON teams(lead_user_id) WHERE archived_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_team_memberships_user ON team_memberships(user_id, status);
CREATE INDEX IF NOT EXISTS idx_team_memberships_team_role ON team_memberships(team_id, role) WHERE status = 'active';

-- Ровно один активный lead на команду.
CREATE UNIQUE INDEX IF NOT EXISTS uq_team_one_lead ON team_memberships(team_id)
    WHERE role = 'lead' AND status = 'active';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS uq_team_one_lead;
DROP INDEX IF EXISTS idx_team_memberships_team_role;
DROP INDEX IF EXISTS idx_team_memberships_user;
DROP INDEX IF EXISTS idx_teams_lead;
DROP TABLE IF EXISTS team_memberships;
DROP TABLE IF EXISTS teams;
-- +goose StatementEnd
```

- [ ] **Step 2: Run `make migrate-up`** в локальной среде (или `goose -dir backend/migrations postgres "$DATABASE_URL" up`).
- [ ] **Step 3: Run `make migrate-down` once** — убедиться, что Down работает чисто.
- [ ] **Step 4: Run migrate-up again** — повторное применение должно быть чистым (idempotent).
- [ ] **Step 5: Verify** через `psql`:
  - `\dt teams`, `\dt team_memberships` — таблицы есть.
  - `\di idx_teams_lead`, `\di uq_team_one_lead` — индексы есть.

---

### Task 2: Migration `00012_team_extensions.sql`

**Files:**
- Create: `backend/migrations/00012_team_extensions.sql`

- [ ] **Step 1: Write the migration SQL**

```sql
-- +goose Up
-- +goose StatementBegin

ALTER TABLE goals
    ADD COLUMN IF NOT EXISTS team_id BIGINT REFERENCES teams(id) ON DELETE SET NULL;

-- XOR: цель либо в круге, либо в команде, либо ни там, ни там — но не в обоих сразу.
ALTER TABLE goals
    ADD CONSTRAINT goal_circle_or_team_xor
    CHECK (
        circle_id IS NULL OR team_id IS NULL
    );

ALTER TABLE check_ins
    ADD COLUMN IF NOT EXISTS team_id BIGINT REFERENCES teams(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS approver_role TEXT
        CHECK (approver_role IS NULL OR approver_role IN ('lead', 'trusted_approver'));

CREATE INDEX IF NOT EXISTS idx_goals_team_owner ON goals(team_id, owner_user_id) WHERE team_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_check_ins_team_status ON check_ins(team_id, status) WHERE team_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_check_ins_team_status;
DROP INDEX IF EXISTS idx_goals_team_owner;

ALTER TABLE check_ins
    DROP COLUMN IF EXISTS approver_role,
    DROP COLUMN IF EXISTS team_id;

ALTER TABLE goals
    DROP CONSTRAINT IF EXISTS goal_circle_or_team_xor,
    DROP COLUMN IF EXISTS team_id;
-- +goose StatementEnd
```

> **Note:** XOR через одно условие `(circle_id IS NULL OR team_id IS NULL)` эквивалентен «не оба не-null». Это разрешает: оба null, только circle, только team. Это и нужно по спеке.

- [ ] **Step 2: Apply, rollback, re-apply** (как Task 1, шаги 2–4).
- [ ] **Step 3: Manual XOR verification** — попытаться вставить goal с обоими `circle_id` и `team_id` через psql, должна упасть на CHECK.

---

### Task 3: Domain types in `teams` package

**Files:**
- Create: `backend/internal/teams/domain.go`
- Create: `backend/internal/teams/domain_test.go`
- Test: `backend/internal/teams/domain_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package teams

import (
	"strings"
	"testing"
)

func TestRoleString(t *testing.T) {
	cases := []struct {
		role Role
		want string
	}{
		{RoleLead, "lead"},
		{RoleTrustedApprover, "trusted_approver"},
		{RoleMember, "member"},
	}
	for _, c := range cases {
		if string(c.role) != c.want {
			t.Errorf("Role %v string = %q, want %q", c.role, string(c.role), c.want)
		}
	}
}

func TestParseRoleAcceptsValidValues(t *testing.T) {
	cases := []struct {
		in   string
		want Role
	}{
		{"lead", RoleLead},
		{"trusted_approver", RoleTrustedApprover},
		{"member", RoleMember},
	}
	for _, c := range cases {
		got, err := ParseRole(c.in)
		if err != nil {
			t.Fatalf("ParseRole(%q) returned error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("ParseRole(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseRoleRejectsUnknown(t *testing.T) {
	if _, err := ParseRole("admin"); err == nil {
		t.Fatal("ParseRole(\"admin\") expected error, got nil")
	}
}

func TestParseAIModeAcceptsValidValues(t *testing.T) {
	cases := []struct {
		in   string
		want AIMode
	}{
		{"off", AIModeOff},
		{"metadata-only", AIModeMetadataOnly},
		{"full", AIModeFull},
	}
	for _, c := range cases {
		got, err := ParseAIMode(c.in)
		if err != nil {
			t.Fatalf("ParseAIMode(%q) returned error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("ParseAIMode(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseAIModeRejectsUnknown(t *testing.T) {
	if _, err := ParseAIMode("hyper"); err == nil {
		t.Fatal("ParseAIMode(\"hyper\") expected error, got nil")
	}
}

func TestValidateTeamNameRejectsEmpty(t *testing.T) {
	if err := ValidateTeamName(""); err == nil {
		t.Fatal("ValidateTeamName(\"\") expected error")
	}
	if err := ValidateTeamName("   "); err == nil {
		t.Fatal("ValidateTeamName whitespace expected error")
	}
}

func TestValidateTeamNameRejectsTooLong(t *testing.T) {
	long := strings.Repeat("я", 81)
	if err := ValidateTeamName(long); err == nil {
		t.Fatal("ValidateTeamName(81 chars) expected error")
	}
}

func TestValidateTeamNameAcceptsBoundary(t *testing.T) {
	if err := ValidateTeamName("a"); err != nil {
		t.Fatalf("ValidateTeamName(1 char) unexpected error: %v", err)
	}
	long := strings.Repeat("я", 80)
	if err := ValidateTeamName(long); err != nil {
		t.Fatalf("ValidateTeamName(80 chars) unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Make the tests pass**

Реализация в `domain.go`:

```go
package teams

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Role enumerates the three membership roles inside a team.
//
// Lead is the team owner — exactly one per team is enforced via partial
// unique index in migration 00011. TrustedApprover has approval rights
// delegated by the lead. Member can submit proofs and comment but cannot
// approve.
type Role string

const (
	RoleLead            Role = "lead"
	RoleTrustedApprover Role = "trusted_approver"
	RoleMember          Role = "member"
)

// MembershipStatus mirrors the BD CHECK constraint on team_memberships.status.
type MembershipStatus string

const (
	MembershipStatusActive  MembershipStatus = "active"
	MembershipStatusLeft    MembershipStatus = "left"
	MembershipStatusRemoved MembershipStatus = "removed"
)

// AIMode defines per-team privacy posture for personalization layer.
//
// Off — никаких AI-вызовов, только шаблоны.
// MetadataOnly (default) — в LLM идут алиасы и заголовки, но не содержимое.
// Full — содержимое заметок и proof'ов передаётся в LLM (требует ai_consent
// от каждого user'а индивидуально).
type AIMode string

const (
	AIModeOff          AIMode = "off"
	AIModeMetadataOnly AIMode = "metadata-only"
	AIModeFull         AIMode = "full"
)

const (
	DefaultMemberLimit = 25
	MaxTeamNameLength  = 80
)

// Domain errors. Map to HTTP via http_handler.go.
var (
	ErrTeamNotFound                = errors.New("team not found")
	ErrTeamArchived                = errors.New("team archived")
	ErrTeamFull                    = errors.New("team full")
	ErrAlreadyMember               = errors.New("already a member")
	ErrNotMember                   = errors.New("not a member of this team")
	ErrNotLead                     = errors.New("only the lead can do this")
	ErrMustHaveLead                = errors.New("team must have a lead")
	ErrCannotDemoteSelf            = errors.New("lead cannot demote themselves")
	ErrCannotRemoveSelfAsLead      = errors.New("lead cannot remove themselves; transfer first")
	ErrCannotLeaveAsOnlyLead       = errors.New("lead cannot leave the only-lead team; transfer first")
	ErrCannotChangeOthersAIConsent = errors.New("only the user themselves can change ai_consent")
	ErrInvalidInviteCode           = errors.New("invalid invite code")
	ErrInvalidTeamName             = errors.New("invalid team name")
	ErrInvalidRole                 = errors.New("invalid role")
	ErrInvalidAIMode               = errors.New("invalid ai_mode")
	ErrInvalidTimezone             = errors.New("invalid timezone")
)

// Team is the persisted aggregate.
type Team struct {
	ID          int64
	LeadUserID  int64
	Name        string
	InviteCode  string
	MemberLimit int
	AIMode      AIMode
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ArchivedAt  *time.Time
}

// Membership is the link between a user and a team with role/state.
type Membership struct {
	ID        int64
	TeamID    int64
	UserID    int64
	Role      Role
	Status    MembershipStatus
	AIConsent bool
	Timezone  string
	JoinedAt  time.Time
	LeftAt    *time.Time
}

// Detail aggregates a team with the caller's own membership and a member count.
// Used as the canonical read-model for /teams/:id and items in /teams.
type Detail struct {
	Team         Team
	MyMembership Membership
	MemberCount  int
}

func ParseRole(s string) (Role, error) {
	switch s {
	case string(RoleLead), string(RoleTrustedApprover), string(RoleMember):
		return Role(s), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidRole, s)
	}
}

func ParseAIMode(s string) (AIMode, error) {
	switch s {
	case string(AIModeOff), string(AIModeMetadataOnly), string(AIModeFull):
		return AIMode(s), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidAIMode, s)
	}
}

// ValidateTeamName enforces the length CHECK from migration 00011.
// Trims whitespace before counting runes to reject all-space names.
func ValidateTeamName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("%w: empty", ErrInvalidTeamName)
	}
	if utf8.RuneCountInString(trimmed) > MaxTeamNameLength {
		return fmt.Errorf("%w: longer than %d chars", ErrInvalidTeamName, MaxTeamNameLength)
	}
	return nil
}
```

- [ ] **Step 3: Run `go test ./backend/internal/teams/...`** — all green.

---

### Task 4: Authorization rules in `teams/authz.go`

This task is the heart of phase 0 — every endpoint defers to these pure functions.

**Files:**
- Create: `backend/internal/teams/authz.go`
- Create: `backend/internal/teams/authz_test.go`
- Test: `backend/internal/teams/authz_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package teams

import "testing"

func ms(role Role) Membership {
	return Membership{Role: role, Status: MembershipStatusActive}
}

func TestCanApproveProof(t *testing.T) {
	cases := []struct {
		name string
		mem  Membership
		want bool
	}{
		{"lead can approve", ms(RoleLead), true},
		{"trusted can approve", ms(RoleTrustedApprover), true},
		{"member cannot approve", ms(RoleMember), false},
		{"left lead cannot approve", Membership{Role: RoleLead, Status: MembershipStatusLeft}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CanApproveProof(c.mem); got != c.want {
				t.Errorf("CanApproveProof(%v) = %v, want %v", c.mem, got, c.want)
			}
		})
	}
}

func TestCanApproveProofRejectsSelf(t *testing.T) {
	mem := Membership{UserID: 42, Role: RoleLead, Status: MembershipStatusActive}
	if CanApproveProofForOwner(mem, 42) {
		t.Error("CanApproveProofForOwner(self) = true, want false (cannot approve own)")
	}
	if !CanApproveProofForOwner(mem, 7) {
		t.Error("CanApproveProofForOwner(other) = false, want true")
	}
}

func TestCanManageTeam(t *testing.T) {
	if !CanManageTeam(ms(RoleLead)) {
		t.Error("lead should be able to manage team")
	}
	if CanManageTeam(ms(RoleTrustedApprover)) {
		t.Error("trusted should NOT manage team")
	}
	if CanManageTeam(ms(RoleMember)) {
		t.Error("member should NOT manage team")
	}
}

func TestCanReadFullMemberMetrics(t *testing.T) {
	if !CanReadFullMemberMetrics(ms(RoleLead)) {
		t.Error("lead should read full member metrics")
	}
	if !CanReadFullMemberMetrics(ms(RoleTrustedApprover)) {
		t.Error("trusted should read full member metrics")
	}
	if CanReadFullMemberMetrics(ms(RoleMember)) {
		t.Error("member should NOT read others' full metrics")
	}
}

func TestCanReadPeerMetrics(t *testing.T) {
	for _, r := range []Role{RoleLead, RoleTrustedApprover, RoleMember} {
		if !CanReadPeerMetrics(ms(r)) {
			t.Errorf("active %v should read peer metrics", r)
		}
	}
	if CanReadPeerMetrics(Membership{Role: RoleMember, Status: MembershipStatusLeft}) {
		t.Error("left member should NOT read peer metrics")
	}
}

func TestEnsureCanChangeAIConsentOnlySelf(t *testing.T) {
	if err := EnsureCanChangeAIConsent(7, 7); err != nil {
		t.Errorf("self should be allowed, got %v", err)
	}
	if err := EnsureCanChangeAIConsent(7, 8); err == nil {
		t.Error("changing other's consent should fail")
	}
}
```

- [ ] **Step 2: Make the tests pass**

```go
package teams

// CanApproveProof returns true if the membership has approval rights and is
// active. This is the ONLY function that should be used to gate approval
// actions — callers must NOT inline `if mem.Role == RoleLead` checks.
func CanApproveProof(mem Membership) bool {
	if mem.Status != MembershipStatusActive {
		return false
	}
	return mem.Role == RoleLead || mem.Role == RoleTrustedApprover
}

// CanApproveProofForOwner additionally rejects approving one's own proof.
// Caller must pass the owner_user_id of the proof being approved.
func CanApproveProofForOwner(approver Membership, ownerUserID int64) bool {
	if approver.UserID == ownerUserID {
		return false
	}
	return CanApproveProof(approver)
}

// CanManageTeam — only the lead can rename, archive, change AI mode,
// regenerate invite, change other members' roles, remove members.
func CanManageTeam(mem Membership) bool {
	return mem.Status == MembershipStatusActive && mem.Role == RoleLead
}

// CanReadFullMemberMetrics — lead and trusted may read /members/:userId/full.
// Member can only read /full for self (handled separately by EnsureSelf).
func CanReadFullMemberMetrics(mem Membership) bool {
	if mem.Status != MembershipStatusActive {
		return false
	}
	return mem.Role == RoleLead || mem.Role == RoleTrustedApprover
}

// CanReadPeerMetrics — every active member of the team may see peer cards.
func CanReadPeerMetrics(mem Membership) bool {
	return mem.Status == MembershipStatusActive
}

// EnsureCanChangeAIConsent — only the user themselves can flip their consent.
func EnsureCanChangeAIConsent(callerUserID, targetUserID int64) error {
	if callerUserID == targetUserID {
		return nil
	}
	return ErrCannotChangeOthersAIConsent
}
```

- [ ] **Step 3: Run `go test ./backend/internal/teams/ -run Authz -v`** — все 6+ кейсов зелёные.

---

### Task 5: Repository interface in `teams/ports.go`

**Files:**
- Create: `backend/internal/teams/ports.go`

- [ ] **Step 1: Define the interface and param structs**

```go
package teams

import (
	"context"
	"time"
)

// Repository is the persistence boundary for the teams package. It is
// implemented by postgres_repository.go. Tests use an in-memory fake
// (defined in service_test.go).
type Repository interface {
	// CreateTeam inserts the team row AND the lead's membership row in a
	// single transaction. Returns the freshly-created Detail with my_role=lead.
	CreateTeam(ctx context.Context, params CreateTeamParams) (Detail, error)

	// ListMyTeams returns all teams where the user has an ACTIVE membership.
	ListMyTeams(ctx context.Context, userID int64) ([]Detail, error)

	// GetTeamForUser returns the team detail for a user who must already be
	// an active member. Returns ErrNotMember if the user has no active
	// membership in the team.
	GetTeamForUser(ctx context.Context, teamID, userID int64) (Detail, error)

	// JoinTeam atomically: looks up the team by invite_code (must not be
	// archived), checks capacity, inserts a 'member' row. Returns ErrTeamFull,
	// ErrTeamArchived, ErrAlreadyMember, ErrInvalidInviteCode appropriately.
	JoinTeam(ctx context.Context, params JoinTeamParams) (Detail, error)

	// ChangeMemberRole — used only by the lead. Service layer guards calls
	// against demoting self / leaving without a lead. Repo just performs the
	// UPDATE and refuses if the partial unique index is violated.
	ChangeMemberRole(ctx context.Context, params ChangeMemberRoleParams) error

	// RemoveMember marks the membership as 'removed' and stamps left_at.
	RemoveMember(ctx context.Context, params RemoveMemberParams) error

	// LeaveTeam marks the membership as 'left' and stamps left_at. Service
	// guards against the only-lead leaving.
	LeaveTeam(ctx context.Context, teamID, userID int64, at time.Time) error

	// SetAIConsent flips ai_consent on the user's own membership.
	SetAIConsent(ctx context.Context, teamID, userID int64, consent bool) error

	// RegenerateInviteCode replaces the invite code with a new value.
	RegenerateInviteCode(ctx context.Context, teamID int64, newCode string) error

	// ArchiveTeam stamps archived_at on the team row.
	ArchiveTeam(ctx context.Context, teamID int64, at time.Time) error

	// GetMembership is the building block for authz checks. Returns
	// ErrNotMember when status != 'active'.
	GetMembership(ctx context.Context, teamID, userID int64) (Membership, error)
}

type CreateTeamParams struct {
	LeadUserID  int64
	Name        string
	InviteCode  string
	MemberLimit int
	AIMode      AIMode
	CreatedAt   time.Time
}

type JoinTeamParams struct {
	UserID     int64
	InviteCode string
	JoinedAt   time.Time
}

type ChangeMemberRoleParams struct {
	TeamID  int64
	UserID  int64
	NewRole Role
}

type RemoveMemberParams struct {
	TeamID int64
	UserID int64
	At     time.Time
}
```

- [ ] **Step 2:** `go build ./backend/internal/teams/...` — должно собраться (без impl в repo, но с interface + params).

---

### Task 6: Service layer with TDD against in-memory fake

The service is where invariants live ("ровно один lead", "lead не может уйти один и т.п."). All TDD'ed against a fake repo, real Postgres tested separately in Task 7.

**Files:**
- Create: `backend/internal/teams/service.go`
- Create: `backend/internal/teams/service_test.go`
- Test: `backend/internal/teams/service_test.go`

- [ ] **Step 1: Write the in-memory fake repo helper at top of service_test.go**

```go
package teams

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeRepo is an in-memory implementation of Repository for service tests.
// Real Postgres is tested in postgres_repository_test.go.
type fakeRepo struct {
	mu          sync.Mutex
	teams       map[int64]Team
	memberships map[int64][]Membership // keyed by team_id
	nextTeamID  int64
	nextMemID   int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		teams:       map[int64]Team{},
		memberships: map[int64][]Membership{},
		nextTeamID:  1,
		nextMemID:   1,
	}
}

// ... CreateTeam, ListMyTeams, JoinTeam, ChangeMemberRole, RemoveMember,
//     LeaveTeam, SetAIConsent, GetTeamForUser, GetMembership,
//     RegenerateInviteCode, ArchiveTeam — implement enough to satisfy
//     Repository interface and the test cases below.
```

> **Implementation detail:** Each fake method must respect the same invariants as Postgres (unique invite_code, member_limit, exactly-one-lead, status=active filter). When in doubt, follow the SQL CHECKs from migration 00011.

- [ ] **Step 2: Write the failing test cases**

The service must pass at minimum these test scenarios — one `t.Run` each, table-driven where natural:

```go
func TestCreateTeamHappyPath(t *testing.T) {
	// Service.CreateTeam returns Detail with my_role=lead, member_count=1.
	// Invite code is generated, non-empty, unique.
	// Default ai_mode is 'metadata-only' if not specified.
}

func TestCreateTeamRejectsEmptyName(t *testing.T) {
	// Returns ErrInvalidTeamName.
}

func TestJoinTeamByInviteCodeHappyPath(t *testing.T) {
	// User2 joins lead's team via invite_code. Detail.MyMembership.Role == member.
	// MemberCount == 2.
}

func TestJoinTeamRejectsBadCode(t *testing.T) {
	// ErrInvalidInviteCode.
}

func TestJoinTeamRejectsArchived(t *testing.T) {
	// ErrTeamArchived.
}

func TestJoinTeamRejectsWhenFull(t *testing.T) {
	// member_limit=2 → 1 lead + 1 member → 3rd join → ErrTeamFull.
}

func TestJoinTeamIdempotentForExistingMember(t *testing.T) {
	// Joining twice with the same code does NOT create a duplicate row.
	// Second call returns the existing membership detail.
}

func TestChangeMemberRolePromoteToTrusted(t *testing.T) {
	// Lead promotes member → trusted_approver. CanApproveProof becomes true.
}

func TestChangeMemberRoleNonLeadRejected(t *testing.T) {
	// trusted_approver tries to promote → ErrNotLead.
}

func TestChangeMemberRoleCannotDemoteSelf(t *testing.T) {
	// Lead tries to set self to member → ErrCannotDemoteSelf.
}

func TestRemoveMemberHappyPath(t *testing.T) {
	// Lead removes a member. Status=removed, left_at is set.
	// Subsequent GetMembership returns ErrNotMember.
}

func TestRemoveMemberCannotRemoveSelfAsLead(t *testing.T) {
	// Lead tries to remove self → ErrCannotRemoveSelfAsLead.
}

func TestLeaveTeamMember(t *testing.T) {
	// Member leaves. Status=left, left_at set.
}

func TestLeaveTeamCannotLeaveAsOnlyLead(t *testing.T) {
	// Lead tries to leave a team where they are the only lead → ErrCannotLeaveAsOnlyLead.
}

func TestSetAIConsentSelfOnly(t *testing.T) {
	// User can flip own consent. Trying to flip another user's consent → ErrCannotChangeOthersAIConsent.
}

func TestRegenerateInviteCodeOnlyLead(t *testing.T) {
	// Lead regenerates → new code differs. Member tries → ErrNotLead.
}

func TestArchiveTeamOnlyLead(t *testing.T) {
	// Lead archives. archived_at is set. Joining via invite_code now returns ErrTeamArchived.
	// Member tries to archive → ErrNotLead.
}

func TestListMyTeamsExcludesNonActiveMemberships(t *testing.T) {
	// User left team A, removed from B. ListMyTeams returns only active.
}
```

- [ ] **Step 3: Implement Service in `service.go` to make tests pass**

Key signatures:

```go
package teams

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"strings"
	"time"
)

type Service struct {
	repo  Repository
	clock func() time.Time
}

type ServiceOption func(*Service)

func WithClock(f func() time.Time) ServiceOption {
	return func(s *Service) { s.clock = f }
}

func NewService(repo Repository, opts ...ServiceOption) *Service {
	s := &Service{repo: repo, clock: time.Now}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// CreateTeam — every authenticated user can create their own team.
// Returns Detail with my_role=lead, member_count=1.
func (s *Service) CreateTeam(ctx context.Context, leadUserID int64, name string, mode AIMode) (Detail, error) {
	if err := ValidateTeamName(name); err != nil {
		return Detail{}, err
	}
	if mode == "" {
		mode = AIModeMetadataOnly
	}
	if _, err := ParseAIMode(string(mode)); err != nil {
		return Detail{}, err
	}
	code, err := generateInviteCode()
	if err != nil {
		return Detail{}, err
	}
	return s.repo.CreateTeam(ctx, CreateTeamParams{
		LeadUserID:  leadUserID,
		Name:        strings.TrimSpace(name),
		InviteCode:  code,
		MemberLimit: DefaultMemberLimit,
		AIMode:      mode,
		CreatedAt:   s.clock(),
	})
}

// ListMyTeams returns all teams the user is currently active in.
func (s *Service) ListMyTeams(ctx context.Context, userID int64) ([]Detail, error) {
	return s.repo.ListMyTeams(ctx, userID)
}

// GetTeamForUser is the canonical /teams/:id read.
func (s *Service) GetTeamForUser(ctx context.Context, teamID, userID int64) (Detail, error) {
	return s.repo.GetTeamForUser(ctx, teamID, userID)
}

// JoinTeam — user joins by invite_code. Idempotent for existing members.
func (s *Service) JoinTeam(ctx context.Context, userID int64, inviteCode string) (Detail, error) {
	code := strings.TrimSpace(inviteCode)
	if code == "" {
		return Detail{}, ErrInvalidInviteCode
	}
	return s.repo.JoinTeam(ctx, JoinTeamParams{
		UserID:     userID,
		InviteCode: code,
		JoinedAt:   s.clock(),
	})
}

// ChangeMemberRole — only lead. Cannot demote self.
func (s *Service) ChangeMemberRole(ctx context.Context, callerID, teamID, targetUserID int64, newRole Role) error {
	mem, err := s.repo.GetMembership(ctx, teamID, callerID)
	if err != nil {
		return err
	}
	if !CanManageTeam(mem) {
		return ErrNotLead
	}
	if callerID == targetUserID && newRole != RoleLead {
		return ErrCannotDemoteSelf
	}
	return s.repo.ChangeMemberRole(ctx, ChangeMemberRoleParams{
		TeamID:  teamID,
		UserID:  targetUserID,
		NewRole: newRole,
	})
}

// RemoveMember — lead only. Cannot remove self (must transfer or archive).
func (s *Service) RemoveMember(ctx context.Context, callerID, teamID, targetUserID int64) error {
	mem, err := s.repo.GetMembership(ctx, teamID, callerID)
	if err != nil {
		return err
	}
	if !CanManageTeam(mem) {
		return ErrNotLead
	}
	if callerID == targetUserID {
		return ErrCannotRemoveSelfAsLead
	}
	return s.repo.RemoveMember(ctx, RemoveMemberParams{
		TeamID: teamID,
		UserID: targetUserID,
		At:     s.clock(),
	})
}

// LeaveTeam — anyone can leave, except the only lead.
func (s *Service) LeaveTeam(ctx context.Context, callerID, teamID int64) error {
	mem, err := s.repo.GetMembership(ctx, teamID, callerID)
	if err != nil {
		return err
	}
	if mem.Role == RoleLead {
		// Only-lead check is enforced in repo via partial unique index;
		// repo returns ErrCannotLeaveAsOnlyLead when the constraint would
		// be violated (no other active lead exists).
		return ErrCannotLeaveAsOnlyLead
	}
	return s.repo.LeaveTeam(ctx, teamID, callerID, s.clock())
}

// SetMyAIConsent — caller is always the target. The handler enforces that
// userID == auth.user_id, but we double-check here.
func (s *Service) SetMyAIConsent(ctx context.Context, callerID, teamID, targetUserID int64, consent bool) error {
	if err := EnsureCanChangeAIConsent(callerID, targetUserID); err != nil {
		return err
	}
	return s.repo.SetAIConsent(ctx, teamID, targetUserID, consent)
}

// RegenerateInvite — lead only.
func (s *Service) RegenerateInvite(ctx context.Context, callerID, teamID int64) (string, error) {
	mem, err := s.repo.GetMembership(ctx, teamID, callerID)
	if err != nil {
		return "", err
	}
	if !CanManageTeam(mem) {
		return "", ErrNotLead
	}
	code, err := generateInviteCode()
	if err != nil {
		return "", err
	}
	if err := s.repo.RegenerateInviteCode(ctx, teamID, code); err != nil {
		return "", err
	}
	return code, nil
}

// ArchiveTeam — lead only. Joining via invite_code returns ErrTeamArchived.
func (s *Service) ArchiveTeam(ctx context.Context, callerID, teamID int64) error {
	mem, err := s.repo.GetMembership(ctx, teamID, callerID)
	if err != nil {
		return err
	}
	if !CanManageTeam(mem) {
		return ErrNotLead
	}
	return s.repo.ArchiveTeam(ctx, teamID, s.clock())
}

// generateInviteCode — 12-char base32 (no padding, no ambiguous chars).
func generateInviteCode() (string, error) {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	out := strings.ToUpper(enc.EncodeToString(b[:]))
	// 16 chars from 10 bytes, take first 12 — collision space ~10^15.
	return out[:12], nil
}
```

- [ ] **Step 4: Run `go test ./backend/internal/teams/... -v`** — все service-тесты зелёные.

> **Hint for executor:** when implementing the fakeRepo, the only-lead invariant is the trickiest. Mirror what Postgres' partial unique index does: keep a per-team `activeLeadCount` and refuse INSERT/UPDATE that would push it >1, and refuse leave/role-change that would push it to 0.

---

### Task 7: Postgres repository implementation

**Files:**
- Create: `backend/internal/teams/postgres_repository.go`
- Create: `backend/internal/teams/postgres_repository_test.go`
- Test: `backend/internal/teams/postgres_repository_test.go`

- [ ] **Step 1: Write integration tests against real Postgres**

Use existing `testutil/integration_db.go` harness. Add TRUNCATE for new tables there (Task 13).

```go
// +build integration

package teams_test

import (
	"context"
	"testing"
	"time"

	"github.com/proofforge/backend/internal/teams"
	"github.com/proofforge/backend/testutil"
)

func TestPostgres_CreateTeam_LeadMembershipCreatedAtomically(t *testing.T) {
	// Setup: integration DB + a user.
	// Action: CreateTeam.
	// Assert: SELECT 1 FROM team_memberships WHERE team_id=? AND role='lead' returns 1 row.
	// Assert: invite_code is unique across multiple CreateTeam calls.
}

func TestPostgres_PartialUniqueIndex_RejectsSecondLead(t *testing.T) {
	// Setup: team with one lead.
	// Action: try to INSERT another row with role='lead', status='active', same team_id.
	// Assert: returns SQL unique_violation; repo translates it into a meaningful error.
}

func TestPostgres_JoinTeam_RaceCondition_HonorsLimit(t *testing.T) {
	// Setup: team with member_limit=2 (lead + 1 slot).
	// Action: launch 5 concurrent JoinTeam calls.
	// Assert: exactly 1 succeeds, 4 return ErrTeamFull.
}

func TestPostgres_GoalCircleOrTeamXOR_Enforced(t *testing.T) {
	// Setup: a team and a circle.
	// Action: try to INSERT INTO goals(circle_id, team_id, ...) VALUES (?, ?, ...).
	// Assert: CHECK constraint goal_circle_or_team_xor fires.
}

func TestPostgres_ArchiveTeam_HidesFromInviteCodeJoin(t *testing.T) {
	// Setup: team, then ArchiveTeam.
	// Action: JoinTeam with the team's invite_code.
	// Assert: ErrTeamArchived.
}

func TestPostgres_LeaveTeam_OnlyLead_PartialUniqueAllowsLeave_RepoReturnsErr(t *testing.T) {
	// Setup: team with only the lead.
	// Action: LeaveTeam(lead).
	// Assert: repo detects no other active lead exists → returns ErrCannotLeaveAsOnlyLead.
	//         membership row is unchanged.
}
```

- [ ] **Step 2: Implement `postgres_repository.go`**

Follow conventions from `circles/postgres_repository.go`. Key SQL fragments:

- `CreateTeam` — `BEGIN; INSERT teams; INSERT team_memberships(role='lead'); COMMIT;` — single transaction.
- `JoinTeam` — `BEGIN; SELECT team WHERE invite_code=? AND archived_at IS NULL FOR UPDATE; check capacity via COUNT; INSERT membership; COMMIT;` — `FOR UPDATE` is what makes the race-condition test pass.
- Unique-violation translation: catch `pgconn.PgError.Code == "23505"`, inspect constraint name, return `ErrTeamFull` / `ErrInvalidInviteCode` accordingly.
- `LeaveTeam` for lead: pre-check `SELECT COUNT(*) FROM team_memberships WHERE team_id=? AND role='lead' AND status='active' AND user_id != ?` — if 0, return `ErrCannotLeaveAsOnlyLead` without touching the row.

- [ ] **Step 3: Run `go test -tags=integration ./backend/internal/teams/...`** — все tests зелёные.

---

### Task 8: HTTP handler in `teams/http_handler.go`

**Files:**
- Create: `backend/internal/teams/http_handler.go`
- Create: `backend/internal/teams/http_test.go`
- Test: `backend/internal/teams/http_test.go`

- [ ] **Step 1: Write the failing handler tests**

Use existing helpers from `users/http_handler_test.go` (or whatever pattern is established). Each test mounts a chi router with the auth middleware mock and a stub `Service`.

```go
func TestHandler_CreateTeam_201(t *testing.T) {
	// POST /teams body {"name":"ML team","ai_mode":"metadata-only"} as user 7.
	// Assert: 201, response.data.team.name == "ML team", my_role == "lead", member_count == 1.
}

func TestHandler_CreateTeam_ValidationFailure_400(t *testing.T) {
	// POST with empty name. Expect 400, error.code == "validation.field_invalid".
}

func TestHandler_JoinTeam_201(t *testing.T) {
	// POST /teams/join {"invite_code": "..."} as user 8 — returns 201 with membership.
}

func TestHandler_JoinTeam_BadCode_404(t *testing.T) {
	// expect 404, code == "invite.invalid".
}

func TestHandler_JoinTeam_Archived_409(t *testing.T) {
	// expect 409, code == "team.archived".
}

func TestHandler_GetTeam_200(t *testing.T) {
	// GET /teams/:id as a member — returns Detail.
}

func TestHandler_GetTeam_NonMember_403(t *testing.T) {
	// GET /teams/:id as a stranger — 403, code == "team.not_member".
}

func TestHandler_ChangeRole_NotLead_403(t *testing.T) {
	// POST /teams/:id/members/:userId/role as member — 403.
}

func TestHandler_ArchiveTeam_NotLead_403(t *testing.T) {
	// POST /teams/:id/archive as member — 403.
}

func TestHandler_AICotsentChangeOthers_403(t *testing.T) {
	// POST /teams/:id/members/:userId/ai-consent for someone other than auth user — 403.
}

func TestHandler_LeaveTeam_OnlyLead_409(t *testing.T) {
	// POST /teams/:id/leave as the only lead — 409, code == "team.cannot_leave_as_only_lead".
}

func TestHandler_RegenerateInvite_NewCodeReturned(t *testing.T) {
	// POST /teams/:id/regenerate-invite as lead — 200 with new code.
}
```

- [ ] **Step 2: Implement `http_handler.go`**

Routes (mounted under `/v1` by api.go):

```
POST   /teams
GET    /teams
GET    /teams/:id
POST   /teams/:id/archive
POST   /teams/:id/regenerate-invite
POST   /teams/join
POST   /teams/:id/members/:userId/role
DELETE /teams/:id/members/:userId
POST   /teams/:id/leave
POST   /teams/:id/members/:userId/ai-consent
```

Handler responsibilities:
1. Read auth user from context (`users.CurrentUser(r.Context())`).
2. Decode + validate JSON body.
3. Call service.
4. Map domain errors → http status + error.code (closed list from spec §7).
5. Encode response in `{ "data": ..., "meta": ... }` shape.

Error mapping (`http_errors.go` or inline):

```go
var errMap = map[error]struct {
	status int
	code   string
}{
	ErrTeamNotFound:                {http.StatusNotFound, "team.not_found"},
	ErrTeamArchived:                {http.StatusConflict, "team.archived"},
	ErrTeamFull:                    {http.StatusConflict, "team.full"},
	ErrAlreadyMember:               {http.StatusConflict, "team.already_member"},
	ErrNotMember:                   {http.StatusForbidden, "team.not_member"},
	ErrNotLead:                     {http.StatusForbidden, "team.not_lead"},
	ErrMustHaveLead:                {http.StatusConflict, "team.must_have_lead"},
	ErrCannotDemoteSelf:            {http.StatusConflict, "team.cannot_demote_self"},
	ErrCannotRemoveSelfAsLead:      {http.StatusConflict, "team.cannot_remove_self_as_lead"},
	ErrCannotLeaveAsOnlyLead:       {http.StatusConflict, "team.cannot_leave_as_only_lead"},
	ErrCannotChangeOthersAIConsent: {http.StatusForbidden, "team.cannot_change_others_consent"},
	ErrInvalidInviteCode:           {http.StatusNotFound, "invite.invalid"},
	ErrInvalidTeamName:             {http.StatusBadRequest, "validation.field_invalid"},
	ErrInvalidRole:                 {http.StatusBadRequest, "validation.field_invalid"},
	ErrInvalidAIMode:               {http.StatusBadRequest, "validation.field_invalid"},
}
```

- [ ] **Step 3: Run `go test ./backend/internal/teams/... -run Handler -v`** — все handler-тесты зелёные.

---

### Task 9: Wire teams router into `api.go`

**Files:**
- Modify: `backend/internal/platform/app/api.go`

- [ ] **Step 1: Add import + repo + service + handler construction in app initialization**

```go
// Inside the app builder where other repos/services live:
teamsRepo := teams.NewPostgresRepository(db)
teamsService := teams.NewService(teamsRepo)
teamsHandler := teams.NewHandler(teamsService, log)
```

- [ ] **Step 2: Mount under `/v1` with auth middleware**

```go
router.Route("/v1", func(r chi.Router) {
    // ... existing routes ...
    r.Group(func(r chi.Router) {
        r.Use(authMiddleware) // same as goals
        teamsHandler.Mount(r) // adds /teams/* routes
    })
})
```

- [ ] **Step 3: Run `go build ./...`** — clean.
- [ ] **Step 4: Smoke test locally:**
  - `curl -X POST localhost:8080/v1/teams -d '{"name":"Test"}' -H "Cookie: <auth>"` → 201.
  - `curl localhost:8080/v1/teams -H "Cookie: <auth>"` → 200 with one team.

---

### Task 10: Goals package — minimal team_id read support

**Files:**
- Modify: `backend/internal/goals/domain.go`
- Modify: `backend/internal/goals/postgres_repository.go`
- Modify: `backend/internal/goals/service_test.go`

Phase 0 only adds **read** support — write paths still create goals with `team_id = NULL`. Write paths will be updated in Phase 2 when team-bound proofs are introduced.

- [ ] **Step 1: Add `TeamID *int64` to `Goal` struct**

```go
// In goals/domain.go
type Goal struct {
    // ... existing fields ...
    TeamID *int64 // nullable — set only for goals created in a team context
}
```

- [ ] **Step 2: Update `SELECT` queries in postgres_repository.go**

```sql
SELECT g.id, g.owner_user_id, ..., g.circle_id, g.team_id
FROM goals g
WHERE ...
```

Scan into `&goal.TeamID` (use `pgtype.Int8` or sql.NullInt64 → `*int64` conversion, follow project convention).

- [ ] **Step 3: Add a regression test**

```go
func TestExistingGoals_HaveNullTeamID_AfterMigration(t *testing.T) {
    // Setup: insert a goal via existing CreateGoal (no team_id).
    // Assert: GetGoal returns goal with TeamID == nil.
}
```

- [ ] **Step 4: `go test ./backend/internal/goals/...`** — green.

---

### Task 11: Web — types and API client

**Files:**
- Modify: `web/lib/types.ts`
- Modify: `web/lib/api.ts`

- [ ] **Step 1: Add types**

```ts
// web/lib/types.ts

export type TeamRole = "lead" | "trusted_approver" | "member";
export type TeamAIMode = "off" | "metadata-only" | "full";
export type TeamMembershipStatus = "active" | "left" | "removed";

export interface Team {
  id: number;
  lead_user_id: number;
  name: string;
  invite_code: string;
  member_limit: number;
  ai_mode: TeamAIMode;
  created_at: string;
  archived_at: string | null;
}

export interface TeamMembership {
  id: number;
  team_id: number;
  user_id: number;
  role: TeamRole;
  status: TeamMembershipStatus;
  ai_consent: boolean;
  timezone: string;
  joined_at: string;
}

export interface TeamDetail {
  team: Team;
  my_membership: TeamMembership;
  member_count: number;
}
```

- [ ] **Step 2: Add API functions**

```ts
// web/lib/api.ts

export async function createTeam(input: { name: string; ai_mode?: TeamAIMode }): Promise<TeamDetail> {
  return rawFetch<TeamDetail>("/v1/teams", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
}

export async function listMyTeams(): Promise<{ teams: TeamDetail[] }> {
  return rawFetch<{ teams: TeamDetail[] }>("/v1/teams");
}

export async function getTeam(id: number): Promise<TeamDetail> {
  return rawFetch<TeamDetail>(`/v1/teams/${id}`);
}

export async function joinTeam(invite_code: string): Promise<TeamDetail> {
  return rawFetch<TeamDetail>("/v1/teams/join", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ invite_code }),
  });
}

export async function changeMemberRole(teamId: number, userId: number, role: TeamRole): Promise<void> {
  await rawFetch<void>(`/v1/teams/${teamId}/members/${userId}/role`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ role }),
  });
}

export async function removeMember(teamId: number, userId: number): Promise<void> {
  await rawFetch<void>(`/v1/teams/${teamId}/members/${userId}`, { method: "DELETE" });
}

export async function leaveTeam(teamId: number): Promise<void> {
  await rawFetch<void>(`/v1/teams/${teamId}/leave`, { method: "POST" });
}

export async function regenerateTeamInvite(teamId: number): Promise<{ invite_code: string }> {
  return rawFetch<{ invite_code: string }>(`/v1/teams/${teamId}/regenerate-invite`, {
    method: "POST",
  });
}

export async function archiveTeam(teamId: number): Promise<void> {
  await rawFetch<void>(`/v1/teams/${teamId}/archive`, { method: "POST" });
}

export async function setMyAIConsent(teamId: number, userId: number, consent: boolean): Promise<void> {
  await rawFetch<void>(`/v1/teams/${teamId}/members/${userId}/ai-consent`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ consent }),
  });
}
```

- [ ] **Step 3:** `npx tsc --noEmit` — clean.

---

### Task 12: Web — pages skeleton

**Files (all create):**
- `web/app/(product)/teams/page.tsx`
- `web/app/(product)/teams/page.module.css`
- `web/app/(product)/teams/new/page.tsx`
- `web/app/(product)/teams/new/page.module.css`
- `web/app/(product)/teams/[id]/page.tsx`
- `web/app/(product)/teams/[id]/page.module.css`
- `web/app/(product)/teams/join/page.tsx`
- `web/app/(product)/teams/join/page.module.css`
- `web/components/product/teams-list.tsx`
- `web/components/product/teams-list.test.tsx`

- [ ] **Step 1: `/teams` (list)**

Loads `listMyTeams()`. Empty state: «КОМАНД ПОКА НЕТ» + CTA «СОЗДАТЬ КОМАНДУ» → `/teams/new`. Uses `<TeamsList>` component for rendering.

- [ ] **Step 2: `/teams/new` (create form)**

Inputs:
- `name` — text, required, max 80 chars.
- `ai_mode` — radio group: «Без AI», «Только метаданные» (default), «Полный режим». Описание под каждым вариантом из спеки §4.

Submit → `createTeam` → redirect to `/teams/:id`.

- [ ] **Step 3: `/teams/[id]` (detail)**

Loads `getTeam(id)`. Renders:
- Team name, badge with `ai_mode`.
- Invite-code box with copy-button («КОПИРОВАТЬ КОД»).
- Member count.
- For lead: button «ПЕРЕГЕНЕРИРОВАТЬ КОД», «АРХИВИРОВАТЬ КОМАНДУ».
- For non-lead: button «ВЫЙТИ ИЗ КОМАНДЫ» (with confirm).

Phase 0 does NOT yet render the member list with roles — that's a Phase 2 surface (when there's actual content to show alongside members).

- [ ] **Step 4: `/teams/join`**

Reads `?code=...` from query string. If present — auto-submits via `joinTeam`. If absent — shows a single text input + button «ВСТУПИТЬ». On success → `/teams/:id`.

- [ ] **Step 5: `<TeamsList>` component + test**

```tsx
// teams-list.tsx
export function TeamsList({ teams }: { teams: TeamDetail[] }) { ... }

// teams-list.test.tsx
it("renders empty state", () => { ... });
it("renders team rows with my_role badge", () => { ... });
it("links each row to /teams/:id", () => { ... });
```

- [ ] **Step 6: `npx vitest run`** — green.

---

### Task 13: Test harness updates

**Files:**
- Modify: `backend/testutil/integration_db.go`
- Modify: `backend/testutil/teamfixture.go` *(create)*

- [ ] **Step 1: Add new tables to TRUNCATE list**

```go
// In testutil/integration_db.go, in the cleanup function:
truncateTables := []string{
    // ... existing list ...
    "team_memberships",
    "teams",
}
```

- [ ] **Step 2: Create `TeamFixture` helper**

```go
package testutil

type TeamFixture struct {
    Team    teams.Team
    Lead    users.User
    Members []users.User
    Trusted *users.User
}

// NewTeamFixture creates a team with lead + N members in active state.
// Optionally promotes the first member to trusted_approver.
func NewTeamFixture(t *testing.T, db *sql.DB, opts ...TeamFixtureOpt) *TeamFixture { ... }
```

This fixture will be reused heavily in Phase 1+ integration tests — start it now even if Phase 0 only uses small bits of it.

- [ ] **Step 3: `go test -tags=integration ./backend/...`** — green.

---

### Task 14: Product nav link

**Files:**
- Modify: `web/components/product/product-nav.tsx`

- [ ] **Step 1: Add «КОМАНДЫ» link to the navbar between «КРУГ» and «ЛЕНТА»**

```tsx
const links = [
    { href: "/dashboard", label: "ДАШБОРД" },
    { href: "/goals/new", label: "КРУГ" },
    { href: "/teams", label: "КОМАНДЫ" },
    { href: "/feed", label: "ЛЕНТА" },
];
```

- [ ] **Step 2: `npx vitest run`** — все existing nav-tests still green.

---

### Task 15: End-to-end smoke

**Files:**
- Manual or scripted (no new file required).

- [ ] **Step 1: Boot stack locally** — `make up` или `docker compose up`.
- [ ] **Step 2: Lead creates a team via web**
  1. Login as user A.
  2. Navigate to `/teams` → empty state.
  3. Click «СОЗДАТЬ КОМАНДУ» → fill form → submit.
  4. Land on `/teams/:id` with my_role=lead, member_count=1.
- [ ] **Step 3: Member joins via invite-code**
  1. Copy `invite_code` from `/teams/:id`.
  2. Logout, login as user B.
  3. Navigate to `/teams/join?code=<code>` → automatic redirect to `/teams/:id`.
  4. Verify `member_count=2`, `my_role=member`.
- [ ] **Step 4: Constraint check** — try to demote yourself as lead via curl, expect 409.
- [ ] **Step 5: Archive flow** — lead archives team. Verify joining via invite_code now returns `team.archived`.

---

### Task 16: Phase 0 quality gate

- [ ] **Step 1: Run full backend test suite**
  - `go test ./backend/...` — all unit tests green.
  - `go test -tags=integration ./backend/...` — all integration tests green.
- [ ] **Step 2: Run full frontend test suite**
  - `cd web && npx vitest run` — all green.
  - `cd web && npx tsc --noEmit` — no errors.
  - `cd web && npx eslint .` — no errors.
- [ ] **Step 3: Migration round-trip in CI**
  - `make migrate-up && make migrate-down && make migrate-up` — clean.
- [ ] **Step 4: Coverage check**
  - `go test -cover ./backend/internal/teams/...` — coverage ≥85% for `teams/` package (per spec §8).
- [ ] **Step 5: Manual smoke test** (Task 15) — passes end-to-end.
- [ ] **Step 6: Update spec status**
  - Add a note at the top of the spec file: «Phase 0 implemented in commit `<sha>`».

---

## Out-of-scope reminders (don't accidentally implement)

The following are explicitly Phase 1+ and must NOT be added in Phase 0 even if it feels easy to throw in:

- ❌ `daily_log_entries` table or any Telegram-bot daily-log handling.
- ❌ `user_streak` table.
- ❌ `analytics_events` partitioned table or any event recording.
- ❌ AI personalization layer (`internal/personalization/`).
- ❌ Team feed rendering at `/feed?tab=team`.
- ❌ Member detail page `/teams/:id/members/:userId/full|peer`.
- ❌ Briefing endpoint or any LLM calls.
- ❌ Email-based team invites (use shared invite-code only).
- ❌ Telegram-bot integration changes (Phase 1).
- ❌ Multi-team contextual UI (active team selector) — Phase 2.

If any of these turn out to be needed for Phase 0 to function, **stop and update the spec** before implementing. Don't silently expand scope.

---

## Definition of Done (Phase 0)

- [ ] All 16 tasks above completed.
- [ ] `quality-gate-runner` skill output green.
- [ ] `security-review` skill: no privacy or auth-bypass concerns flagged.
- [ ] Spec note: «Phase 0 done at `<sha>`».
- [ ] Brief handoff note in `docs/superpowers/reviews/2026-05-XX-phase-0-handoff.md` (use `agent-handoff-writer` skill) so Phase 1 can start without re-discovery.
