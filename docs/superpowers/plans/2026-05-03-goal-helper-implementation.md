# Goal Helper Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить Goal Helper на `/goals/new`: fake-by-default refinement API с cache и rate-limit, drawer с 3 SMART-вариантами, и сохранение `proof_examples`/`category` в `goals`.

**Architecture:** Backend вводит новый `ai.RefineProvider` seam и хранит orchestration в `goals`: validate input, rate limit, cache, provider call, persistence. Frontend получает новый `GoalRefineSheet`, который вызывает `POST /v1/goals/refine`, подставляет выбранный вариант в форму и сохраняет `proof_examples`/`category` при `createGoal`. Реальный OpenAI потом подключается только через backend wiring по флагу, без изменения публичного API.

**Tech Stack:** Go, pgx/PostgreSQL, goose migrations, Next.js 15, React 18, Vitest, ESLint

---

## File Structure

- Create: `backend/internal/ai/llm_provider.go`
  Общий продуктовый AI seam и fake provider для refinement.
- Create: `backend/internal/ai/llm_provider_test.go`
  Unit tests на deterministic fake provider.
- Modify: `backend/internal/goals/domain.go`
  Новые типы refinement, поля `proof_examples` и `category`.
- Modify: `backend/internal/goals/ports.go`
  Новые repository methods для cache/rate-limit и расширенный create payload.
- Modify: `backend/internal/goals/postgres_repository.go`
  Persist `proof_examples`/`category`, cache table, request log table access.
- Modify: `backend/internal/goals/service.go`
  Основная orchestration логика `RefineGoal` и прокидывание новых полей в `CreateGoal`.
- Modify: `backend/internal/goals/service_test.go`
  TDD на cache miss/hit, rate limit, invalid input, persistence payload.
- Modify: `backend/internal/goals/http_handler.go`
  Новый endpoint `POST /goals/refine`.
- Modify: `backend/internal/goals/http_test.go`
  Handler tests на `400`, `429`, `200`.
- Modify: `backend/internal/platform/app/api.go`
  Wiring fake provider в goals service и route registration.
- Create: `backend/migrations/00003_goal_refine.sql`
  `goals.proof_examples`, `goals.category`, `goal_refine_cache`, `goal_refine_requests`.
- Modify: `backend/testutil/integration_db.go`
  TRUNCATE новых refine таблиц для integration harness.
- Create: `web/components/product/goal-refine-sheet.tsx`
  Drawer UI с loading/error/loaded states.
- Create: `web/components/product/goal-refine-sheet.test.tsx`
  Компонентные тесты на drawer и variant selection.
- Modify: `web/components/product/goal-setup-screen.tsx`
  Кнопка refine, client state, submit новых полей.
- Modify: `web/components/product/goal-setup-screen.module.css`
  Layout для кнопки, proof block, drawer trigger.
- Create: `web/components/product/goal-setup-screen.test.tsx`
  TDD на disabled trigger, successful refine selection, submit payload.
- Modify: `web/lib/api.ts`
  `refineGoal()` и расширение `createGoal()`.
- Modify: `web/lib/types.ts`
  Типы refinement payload/response.

---

### Task 1: Add The AI Seam And Deterministic Fake Provider

**Files:**
- Create: `backend/internal/ai/llm_provider.go`
- Create: `backend/internal/ai/llm_provider_test.go`
- Test: `backend/internal/ai/llm_provider_test.go`

- [ ] **Step 1: Write the failing test**

```go
package ai

import (
	"context"
	"testing"
)

func TestFakeGoalRefineProviderReturnsThreeDeterministicVariants(t *testing.T) {
	provider := NewFakeGoalRefineProvider()

	first, err := provider.RefineGoal(context.Background(), "учить англ для собеседований")
	if err != nil {
		t.Fatalf("RefineGoal returned error: %v", err)
	}
	second, err := provider.RefineGoal(context.Background(), "учить англ для собеседований")
	if err != nil {
		t.Fatalf("RefineGoal second call returned error: %v", err)
	}

	if first.Category != "учёба" {
		t.Fatalf("expected category учёба, got %q", first.Category)
	}
	if len(first.Variants) != 3 {
		t.Fatalf("expected 3 variants, got %d", len(first.Variants))
	}
	for i, variant := range first.Variants {
		if variant.Title == "" {
			t.Fatalf("variant %d title is empty", i)
		}
		if variant.Smart == "" {
			t.Fatalf("variant %d smart is empty", i)
		}
		if len(variant.ProofExamples) != 3 {
			t.Fatalf("variant %d expected 3 proof examples, got %d", i, len(variant.ProofExamples))
		}
	}
	if first != second {
		t.Fatalf("expected deterministic result, got %#v and %#v", first, second)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai -run TestFakeGoalRefineProviderReturnsThreeDeterministicVariants -v`
Expected: FAIL with missing package/file/symbols.

- [ ] **Step 3: Write minimal implementation**

```go
package ai

import (
	"context"
	"strings"
)

type GoalRefineVariant struct {
	Title         string   `json:"title"`
	Smart         string   `json:"smart"`
	ProofExamples []string `json:"proof_examples"`
}

type GoalRefineResult struct {
	Category string              `json:"category"`
	Variants []GoalRefineVariant `json:"variants"`
}

type RefineProvider interface {
	RefineGoal(ctx context.Context, draftText string) (GoalRefineResult, error)
}

type FakeGoalRefineProvider struct{}

func NewFakeGoalRefineProvider() *FakeGoalRefineProvider {
	return &FakeGoalRefineProvider{}
}

func (p *FakeGoalRefineProvider) RefineGoal(_ context.Context, draftText string) (GoalRefineResult, error) {
	normalized := strings.TrimSpace(strings.ToLower(draftText))
	category := classifyCategory(normalized)

	return GoalRefineResult{
		Category: category,
		Variants: []GoalRefineVariant{
			buildOutcomeVariant(category, draftText),
			buildCadenceVariant(category, draftText),
			buildEvidenceVariant(category, draftText),
		},
	}, nil
}
```

Add helpers in the same file for:
- `classifyCategory()`
- `buildOutcomeVariant()`
- `buildCadenceVariant()`
- `buildEvidenceVariant()`

Rules for helpers:
- return deterministic strings only
- always return 3 proof examples
- map keywords to `учёба`, `фитнес`, `работа`, `творчество`, else `развитие`

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ai -run TestFakeGoalRefineProviderReturnsThreeDeterministicVariants -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/ai/llm_provider.go backend/internal/ai/llm_provider_test.go
git commit -m "feat: add fake goal refine provider"
```

### Task 2: Add Goal Refine Storage And Extend Goal Persistence

**Files:**
- Create: `backend/migrations/00003_goal_refine.sql`
- Modify: `backend/internal/goals/domain.go`
- Modify: `backend/internal/goals/ports.go`
- Modify: `backend/internal/goals/postgres_repository.go`
- Modify: `backend/testutil/integration_db.go`
- Test: `backend/internal/goals/service_test.go`

- [ ] **Step 1: Write the failing test**

Add to `backend/internal/goals/service_test.go`:

```go
func TestCreateGoalPersistsProofExamplesAndCategory(t *testing.T) {
	var captured CreateGoalParams

	stub := newTestStub()
	stub.createGoal = func(_ context.Context, params CreateGoalParams) (GoalView, error) {
		captured = params
		return GoalView{}, nil
	}

	service := NewService(stub, noopEmailSender{}, "http://localhost:3000", nil, 7*24*time.Hour)
	service.tokenGenerate = func() (string, error) { return "token-123", nil }
	owner := users.User{ID: 1, Email: "owner@example.com", DisplayName: "Owner"}

	_, err := service.CreateGoal(context.Background(), owner, CreateInput{
		Title:         "Уточнить лендинг",
		Description:   "Сделать релиз измеримым",
		BuddyName:     "Peer",
		BuddyEmail:    "peer@example.com",
		ProofExamples: "- Ссылка\n- Скриншот\n- PR",
		Category:      "работа",
	})
	if err != nil {
		t.Fatalf("CreateGoal returned error: %v", err)
	}

	if captured.ProofExamples != "- Ссылка\n- Скриншот\n- PR" {
		t.Fatalf("expected proof examples to be forwarded, got %q", captured.ProofExamples)
	}
	if captured.Category != "работа" {
		t.Fatalf("expected category to be forwarded, got %q", captured.Category)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/goals -run TestCreateGoalPersistsProofExamplesAndCategory -v`
Expected: FAIL because `CreateInput` / `CreateGoalParams` do not yet contain these fields.

- [ ] **Step 3: Write minimal implementation**

Update `backend/internal/goals/domain.go`:

```go
type CreateInput struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	BuddyName     string `json:"buddy_name"`
	BuddyEmail    string `json:"buddy_email"`
	CircleID      int64  `json:"circle_id,omitempty"`
	ProofExamples string `json:"proof_examples,omitempty"`
	Category      string `json:"category,omitempty"`
}
```

Update `Normalize()` to trim `ProofExamples` and `Category`.

Update `backend/internal/goals/ports.go`:

```go
type CreateGoalParams struct {
	OwnerID         int64
	OwnerEmail      string
	Title           string
	Description     string
	BuddyName       string
	BuddyEmail      string
	CircleID        *int64
	ProofExamples   string
	Category        string
	GoalStatus      GoalStatus
	PactStatus      PactStatus
	InviteStatus    InviteStatus
	ProgressHealth  ProgressHealth
	InviteTokenHash string
	InviteExpiresAt time.Time
}
```

Update `backend/internal/goals/service.go`:

```go
params := CreateGoalParams{
	OwnerID:         owner.ID,
	OwnerEmail:      owner.Email,
	Title:           input.Title,
	Description:     input.Description,
	BuddyName:       input.BuddyName,
	BuddyEmail:      input.BuddyEmail,
	CircleID:        nullableInt64(input.CircleID),
	ProofExamples:   input.ProofExamples,
	Category:        input.Category,
	GoalStatus:      GoalStatusPendingBuddyAcceptance,
	PactStatus:      PactStatusInvited,
	InviteStatus:    InviteStatusPending,
	ProgressHealth:  ProgressHealthUnknown,
	InviteTokenHash: hashToken(rawToken),
	InviteExpiresAt: expiresAt,
}
```

Update `backend/internal/goals/postgres_repository.go` insert statement to persist:
- `proof_examples`
- `category`

Create migration `backend/migrations/00003_goal_refine.sql`:

```sql
-- +goose Up
ALTER TABLE goals
    ADD COLUMN IF NOT EXISTS proof_examples TEXT,
    ADD COLUMN IF NOT EXISTS category TEXT;

CREATE TABLE IF NOT EXISTS goal_refine_cache (
    hash TEXT PRIMARY KEY,
    response JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS goal_refine_requests (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    draft_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_goal_refine_requests_user_created
    ON goal_refine_requests(user_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_goal_refine_requests_user_created;
DROP TABLE IF EXISTS goal_refine_requests;
DROP TABLE IF EXISTS goal_refine_cache;
ALTER TABLE goals
    DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS proof_examples;
```

Update `backend/testutil/integration_db.go` truncate list to include:
- `goal_refine_requests`
- `goal_refine_cache`

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/goals -run TestCreateGoalPersistsProofExamplesAndCategory -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/migrations/00003_goal_refine.sql backend/internal/goals/domain.go backend/internal/goals/ports.go backend/internal/goals/postgres_repository.go backend/internal/goals/service.go backend/internal/goals/service_test.go backend/testutil/integration_db.go
git commit -m "feat: persist goal refine metadata"
```

### Task 3: Add Refine Endpoint With Cache And Rate Limit

**Files:**
- Modify: `backend/internal/goals/domain.go`
- Modify: `backend/internal/goals/ports.go`
- Modify: `backend/internal/goals/postgres_repository.go`
- Modify: `backend/internal/goals/service.go`
- Modify: `backend/internal/goals/service_test.go`
- Modify: `backend/internal/goals/http_handler.go`
- Modify: `backend/internal/goals/http_test.go`
- Modify: `backend/internal/platform/app/api.go`
- Test: `backend/internal/goals/service_test.go`
- Test: `backend/internal/goals/http_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `backend/internal/goals/service_test.go`:

```go
func TestRefineGoalReturnsCachedValueBeforeCallingProvider(t *testing.T) {
	stub := newTestStub()
	stub.findGoalRefineCache = func(_ context.Context, hash string, maxAge time.Duration) (GoalRefineResponse, bool, error) {
		return GoalRefineResponse{
			Category: "работа",
			Variants: []GoalRefineVariant{
				{Title: "one", Smart: "one", ProofExamples: []string{"a", "b", "c"}},
				{Title: "two", Smart: "two", ProofExamples: []string{"a", "b", "c"}},
				{Title: "three", Smart: "three", ProofExamples: []string{"a", "b", "c"}},
			},
		}, true, nil
	}

	providerCalled := false
	service := NewService(stub, noopEmailSender{}, "http://localhost:3000", nil, 7*24*time.Hour)
	service.refineProvider = refineProviderFunc(func(context.Context, string) (ai.GoalRefineResult, error) {
		providerCalled = true
		return ai.GoalRefineResult{}, nil
	})

	result, err := service.RefineGoal(context.Background(), users.User{ID: 1}, RefineInput{DraftText: "запустить лендинг"})
	if err != nil {
		t.Fatalf("RefineGoal returned error: %v", err)
	}
	if providerCalled {
		t.Fatal("expected provider not to be called on cache hit")
	}
	if result.Category != "работа" {
		t.Fatalf("expected cached category, got %q", result.Category)
	}
}

func TestRefineGoalReturnsRateLimitErrorAfterFiveRequests(t *testing.T) {
	stub := newTestStub()
	stub.countGoalRefineRequests = func(_ context.Context, userID int64, since time.Time) (int, error) {
		return 5, nil
	}

	service := NewService(stub, noopEmailSender{}, "http://localhost:3000", nil, 7*24*time.Hour)
	_, err := service.RefineGoal(context.Background(), users.User{ID: 1}, RefineInput{DraftText: "учить англ"})
	if !errors.Is(err, ErrGoalRefineRateLimited) {
		t.Fatalf("expected ErrGoalRefineRateLimited, got %v", err)
	}
}
```

Add to `backend/internal/goals/http_test.go`:

```go
func TestHandleRefineGoalRejectsInvalidInput(t *testing.T) {
	service := &serviceStub{
		refineGoal: func(context.Context, users.User, RefineInput) (GoalRefineResponse, error) {
			return GoalRefineResponse{}, ErrInvalidGoalRefineInput
		},
	}

	handler := NewHandler(nil, service)
	req := httptest.NewRequest(http.MethodPost, "/v1/goals/refine", strings.NewReader(`{"draft_text":"  "}`))
	req = req.WithContext(users.WithCurrentUser(req.Context(), users.User{ID: 1, Email: "owner@example.com"}))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.handleRefineGoal(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/goals -run 'TestRefineGoal|TestHandleRefineGoal' -v`
Expected: FAIL with missing types/methods/errors.

- [ ] **Step 3: Write minimal implementation**

Add to `backend/internal/goals/domain.go`:

```go
var (
	ErrInvalidGoalRefineInput = errors.New("invalid goal refine input")
	ErrGoalRefineRateLimited  = errors.New("goal refine rate limited")
)

type RefineInput struct {
	DraftText string `json:"draft_text"`
}

type GoalRefineVariant struct {
	Title         string   `json:"title"`
	Smart         string   `json:"smart"`
	ProofExamples []string `json:"proof_examples"`
}

type GoalRefineResponse struct {
	Category string              `json:"category"`
	Variants []GoalRefineVariant `json:"variants"`
}
```

Add repository methods in `backend/internal/goals/ports.go`:

```go
FindGoalRefineCache(ctx context.Context, hash string, maxAge time.Duration) (GoalRefineResponse, bool, error)
SaveGoalRefineCache(ctx context.Context, hash string, response GoalRefineResponse) error
CountGoalRefineRequests(ctx context.Context, userID int64, since time.Time) (int, error)
InsertGoalRefineRequest(ctx context.Context, userID int64, draftHash string, createdAt time.Time) error
```

Implement service method in `backend/internal/goals/service.go`:

```go
func (s *Service) RefineGoal(ctx context.Context, actor users.User, input RefineInput) (GoalRefineResponse, error) {
	normalized := strings.TrimSpace(input.DraftText)
	if len([]rune(normalized)) < 3 || len([]rune(normalized)) > 240 {
		return GoalRefineResponse{}, ErrInvalidGoalRefineInput
	}

	hash := refineHash(normalized)
	now := s.clock().UTC()

	count, err := s.repo.CountGoalRefineRequests(ctx, actor.ID, now.Add(-24*time.Hour))
	if err != nil {
		return GoalRefineResponse{}, fmt.Errorf("count refine requests: %w", err)
	}
	if count >= 5 {
		return GoalRefineResponse{}, ErrGoalRefineRateLimited
	}
	if err := s.repo.InsertGoalRefineRequest(ctx, actor.ID, hash, now); err != nil {
		return GoalRefineResponse{}, fmt.Errorf("insert refine request: %w", err)
	}

	if cached, ok, err := s.repo.FindGoalRefineCache(ctx, hash, 30*24*time.Hour); err != nil {
		return GoalRefineResponse{}, fmt.Errorf("find refine cache: %w", err)
	} else if ok {
		return cached, nil
	}

	result, err := s.refineProvider.RefineGoal(ctx, normalized)
	if err != nil {
		return GoalRefineResponse{}, fmt.Errorf("provider refine goal: %w", err)
	}
	response, err := normalizeGoalRefineResult(result)
	if err != nil {
		return GoalRefineResponse{}, err
	}
	if err := s.repo.SaveGoalRefineCache(ctx, hash, response); err != nil {
		return GoalRefineResponse{}, fmt.Errorf("save refine cache: %w", err)
	}
	return response, nil
}
```

Add HTTP route in `backend/internal/goals/http_handler.go`:

```go
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/goals", h.handleCreateGoal)
	r.Post("/goals/refine", h.handleRefineGoal)
	r.Get("/goals", h.handleListGoals)
	r.Get("/dashboard", h.handleDashboard)
	r.Post("/invites/{token}/accept", h.handleAcceptInvite)
}
```

Add `handleRefineGoal()` with:
- auth required
- decode JSON
- call `service.RefineGoal`
- map:
  - `ErrInvalidGoalRefineInput` -> `400 invalid_input`
  - `ErrGoalRefineRateLimited` -> `429 refine_rate_limited`
  - success -> `200` with `{ category, variants }`

Update `backend/internal/platform/app/api.go` wiring:
- construct fake provider now
- pass it into goals service constructor

Constructor shape to introduce in `service.go`:

```go
func NewService(repo Repository, emails email.BuddyInviteSender, webOrigin string, log *slog.Logger, inviteTTL time.Duration, options ...Option) *Service
```

With option:

```go
func WithRefineProvider(provider ai.RefineProvider) Option
```

Default provider:
- `ai.NewFakeGoalRefineProvider()`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/goals -run 'TestRefineGoal|TestHandleRefineGoal' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/goals/domain.go backend/internal/goals/ports.go backend/internal/goals/postgres_repository.go backend/internal/goals/service.go backend/internal/goals/service_test.go backend/internal/goals/http_handler.go backend/internal/goals/http_test.go backend/internal/platform/app/api.go
git commit -m "feat: add goal refine endpoint with cache and rate limit"
```

### Task 4: Add Goal Helper UI And Submit Refine Metadata

**Files:**
- Create: `web/components/product/goal-refine-sheet.tsx`
- Create: `web/components/product/goal-refine-sheet.test.tsx`
- Create: `web/components/product/goal-setup-screen.test.tsx`
- Modify: `web/components/product/goal-setup-screen.tsx`
- Modify: `web/components/product/goal-setup-screen.module.css`
- Modify: `web/lib/api.ts`
- Modify: `web/lib/types.ts`
- Test: `web/components/product/goal-refine-sheet.test.tsx`
- Test: `web/components/product/goal-setup-screen.test.tsx`

- [ ] **Step 1: Write the failing frontend tests**

Create `web/components/product/goal-refine-sheet.test.tsx`:

```tsx
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { GoalRefineSheet } from "./goal-refine-sheet";

describe("GoalRefineSheet", () => {
  it("renders three variants and returns the selected one", () => {
    const onPick = vi.fn();

    render(
      <GoalRefineSheet
        open
        loading={false}
        error={null}
        result={{
          category: "работа",
          variants: [
            { title: "A", smart: "A smart", proof_examples: ["1", "2", "3"] },
            { title: "B", smart: "B smart", proof_examples: ["1", "2", "3"] },
            { title: "C", smart: "C smart", proof_examples: ["1", "2", "3"] },
          ],
        }}
        onClose={() => {}}
        onPick={onPick}
      />,
    );

    fireEvent.click(screen.getAllByRole("button", { name: "ВЗЯТЬ" })[0]);
    expect(onPick).toHaveBeenCalledWith({
      title: "A",
      smart: "A smart",
      proof_examples: ["1", "2", "3"],
      category: "работа",
    });
  });
});
```

Create `web/components/product/goal-setup-screen.test.tsx`:

```tsx
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  return {
    ...actual,
    getDashboard: vi.fn(),
    listCircles: vi.fn(),
    refineGoal: vi.fn(),
    createGoal: vi.fn(),
  };
});

it("disables refine button until title has at least three characters", async () => {
  render(<GoalSetupScreen />);
  const button = await screen.findByRole("button", { name: /уточнить цель/i });
  expect(button).toBeDisabled();
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm test -- goal-refine-sheet goal-setup-screen`
Expected: FAIL because files/components/api are missing.

- [ ] **Step 3: Write minimal implementation**

Update `web/lib/types.ts`:

```ts
export type GoalRefineVariant = {
  title: string;
  smart: string;
  proof_examples: string[];
};

export type GoalRefineResponse = {
  category: string;
  variants: GoalRefineVariant[];
};
```

Update `web/lib/api.ts`:

```ts
export type RefineGoalInput = {
  draft_text: string;
};

export async function refineGoal(input: RefineGoalInput): Promise<GoalRefineResponse> {
  return request<GoalRefineResponse>("/v1/goals/refine", {
    method: "POST",
    body: JSON.stringify(input),
  });
}
```

Also extend `CreateGoalInput`:

```ts
proof_examples?: string;
category?: string;
```

Create `web/components/product/goal-refine-sheet.tsx` with props:

```tsx
type PickedVariant = {
  title: string;
  smart: string;
  proof_examples: string[];
  category: string;
};

type GoalRefineSheetProps = {
  open: boolean;
  loading: boolean;
  error: string | null;
  result: GoalRefineResponse | null;
  onClose: () => void;
  onPick: (variant: PickedVariant) => void;
};
```

Behavior:
- if `open === false`, render `null`
- show loading copy when `loading`
- show error block when `error`
- render 3 variant cards with buttons `ВЗЯТЬ` and `ИЗМЕНИТЬ`
- both buttons call `onPick()`

Update `web/components/product/goal-setup-screen.tsx`:
- add local state:
  - `draftTitle`
  - `isRefineOpen`
  - `isRefining`
  - `refineError`
  - `refineResult`
  - `selectedProofExamples`
  - `selectedCategory`
- make title input controlled
- add button below title field:

```tsx
<Button
  type="button"
  variant="secondary"
  disabled={draftTitle.trim().length < 3 || isRefining}
  onClick={handleRefineOpen}
>
  {isRefining ? "[⚡] УТОЧНЯЕМ..." : "[⚡] УТОЧНИТЬ ЦЕЛЬ"}
</Button>
```

- implement `handleRefineOpen()`:
  - set drawer open
  - call `refineGoal({ draft_text: draftTitle })`
  - set states from response
- implement `handlePickVariant()`:
  - set `draftTitle`
  - set description state
  - set `selectedProofExamples`
  - set `selectedCategory`
  - close drawer
- render read-only proof block:

```tsx
{selectedProofExamples.length > 0 ? (
  <div className={styles.proofBlock}>
    <strong>Что считается пруфом</strong>
    <ul>
      {selectedProofExamples.map((item) => (
        <li key={item}>{item}</li>
      ))}
    </ul>
  </div>
) : null}
```

- on submit, send:

```ts
proof_examples: selectedProofExamples.length > 0 ? selectedProofExamples.map((item) => `- ${item}`).join("\n") : undefined,
category: selectedCategory || undefined,
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `npm test -- goal-refine-sheet goal-setup-screen`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/components/product/goal-refine-sheet.tsx web/components/product/goal-refine-sheet.test.tsx web/components/product/goal-setup-screen.tsx web/components/product/goal-setup-screen.module.css web/components/product/goal-setup-screen.test.tsx web/lib/api.ts web/lib/types.ts
git commit -m "feat: add goal helper drawer to goal setup"
```

### Task 5: Run Full Verification For Goal Helper

**Files:**
- Test: `backend/internal/ai/llm_provider_test.go`
- Test: `backend/internal/goals/service_test.go`
- Test: `backend/internal/goals/http_test.go`
- Test: `web/components/product/goal-refine-sheet.test.tsx`
- Test: `web/components/product/goal-setup-screen.test.tsx`

- [ ] **Step 1: Run focused backend tests**

Run: `go test ./internal/ai ./internal/goals ./internal/platform/app -v`
Expected: PASS

- [ ] **Step 2: Run focused frontend tests**

Run: `npm test -- goal-refine-sheet goal-setup-screen`
Expected: PASS

- [ ] **Step 3: Run project frontend quality gates**

Run: `npm run lint && npm run build`
Expected: PASS

- [ ] **Step 4: Run full backend suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/ai backend/internal/goals backend/internal/platform/app backend/migrations/00003_goal_refine.sql backend/testutil web/components/product/goal-refine-sheet.tsx web/components/product/goal-refine-sheet.test.tsx web/components/product/goal-setup-screen.tsx web/components/product/goal-setup-screen.test.tsx web/lib/api.ts web/lib/types.ts
git commit -m "feat: ship fake-backed goal helper"
```

## Self-Review

- Spec coverage: covered seam, fake-by-default wiring, `POST /v1/goals/refine`, cache, rate limit, schema changes, drawer UI, form integration, and verification commands.
- Placeholder scan: no `TODO`/`TBD`; every task names exact files and commands.
- Type consistency: plan uses `GoalRefineResult` in `ai`, `GoalRefineResponse` in `goals`, and `GoalRefineResponse` in frontend API/types; `proof_examples` is transported as string list in refine API and flattened to `text` only at `createGoal` persistence boundary.

Plan complete and saved to `docs/superpowers/plans/2026-05-03-goal-helper-implementation.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
