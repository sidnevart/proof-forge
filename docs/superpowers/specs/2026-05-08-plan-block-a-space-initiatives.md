# Plan: Space Initiatives — Block A

**Spec:** `2026-05-08-spec-block-a-space-initiatives.md`  
**Date:** 2026-05-08

---

## Шаги реализации

### Шаг 1 — Миграция БД

**Файл:** `backend/migrations/00026_space_initiatives.sql`

Создать три изменения:
1. Новая таблица `initiatives` с XOR-констрейнтом на `teamspace_id`/`community_space_id`
2. `ALTER TABLE goals ADD COLUMN initiative_id BIGINT REFERENCES initiatives(id) ON DELETE SET NULL`
3. Новая таблица `initiative_approvals` с UNIQUE на `checkin_id`

Индексы: `idx_initiatives_teamspace`, `idx_initiatives_community`, `idx_goals_initiative`.

---

### Шаг 2 — Go домен: `backend/internal/initiatives/`

**domain.go** — типы и ошибки:
```
Initiative, InitiativeStatus, CreateInitiativeInput, JoinResult
InitiativeDetail (initiative + participants с кол-вом пруфов)
PendingProof (checkin_id, author display_name, content, submitted_at)
ApproveInput (comment string)

ErrInitiativeNotFound, ErrNotSpaceMember, ErrAlreadyJoined,
ErrCannotApproveSelf, ErrNotInitiativeMember, ErrInitiativeArchived
```

**ports.go** — интерфейсы:
```go
type Repository interface {
    Create(ctx, input CreateInitiativeInput) (Initiative, error)
    FindByID(ctx, id int64) (Initiative, error)
    ListBySpace(ctx, spaceType string, spaceID int64) ([]Initiative, error)
    JoinOrGet(ctx, initiativeID, userID int64) (goalID int64, created bool, err error)
    PendingProofs(ctx, initiativeID, viewerID int64) ([]PendingProof, error)
    Approve(ctx, checkinID, approverID int64, comment string) error
    Participants(ctx, initiativeID int64) ([]ParticipantProgress, error)
}

type SpaceMemberChecker interface {
    IsTeamspaceMember(ctx, teamID, userID int64) (bool, error)
    IsCommunityMember(ctx, communityID, userID int64) (bool, error)
}
```

**service.go** — бизнес-логика:
- `Create`: валидация input, проверка членства, вставка
- `Join`: проверка членства в пространстве → `repo.JoinOrGet` (идемпотентен: если goal уже есть — возвращает его)
- `ApprovePendingProof`: проверка `approver ≠ автор` + участник инициативы → `repo.Approve`
- `ListPendingProofs`, `GetDetail`

**service_test.go** — unit-тесты с mock-репозиторием на все инварианты (cannot approve own, not member, already joined → 200).

**postgres_repository.go** — SQL-запросы:
- `JoinOrGet`: INSERT INTO goals ... ON CONFLICT DO NOTHING, затем SELECT
- `PendingProofs`: JOIN check_ins → goals WHERE initiative_id = $1 AND owner_user_id ≠ $2 AND status = 'submitted'
- `Approve`: BEGIN → INSERT initiative_approvals → UPDATE check_ins SET status='approved' → COMMIT

**http_handler.go** — 6 эндпоинтов из спека. Паттерн как у `internal/community/http_handler.go`.

---

### Шаг 3 — Патч checkins service

**Файл:** `backend/internal/checkins/service.go`

В методе создания check_in добавить ветку перед buddy-guard:

```go
// Initiative goals: no buddy required, submit directly
if goal.InitiativeID != nil {
    // skip buddy validation, status = "submitted"
}
```

**Файл:** `backend/internal/goals/domain.go` — добавить поле:
```go
InitiativeID *int64 `json:"initiative_id,omitempty" db:"initiative_id"`
```

**Файл:** `backend/internal/goals/postgres_repository.go` — добавить `initiative_id` в SELECT при чтении goal.

---

### Шаг 4 — Регистрация роутов

**Файл:** `backend/internal/platform/app/api.go`

```go
initiativesRepo := initiatives.NewPostgresRepository(pool)
initiativesSvc  := initiatives.NewService(initiativesRepo, spaceMemberChecker)
initiativesHndl := initiatives.NewHandler(initiativesSvc)

// в protected group:
initiativesHndl.RegisterRoutes(r)
```

`spaceMemberChecker` реализуется как адаптер над существующими `teams.Service` и `community.Service`.

---

### Шаг 5 — Frontend: типы и API-клиент

**Файл:** `web/lib/types.ts` — добавить:
```ts
export interface Initiative {
  id: number; space_type: string; title: string;
  proof_criteria: string; status: string;
  participant_count: number; created_at: string;
  joined?: boolean; my_proof_count?: number; pending_for_me?: number;
}
export interface PendingProof {
  checkin_id: number; author_name: string;
  content: string; submitted_at: string; attachment_url?: string;
}
export interface ParticipantProgress {
  user_id: number; display_name: string; proof_count: number;
}
```

**Файл:** `web/lib/api.ts` — добавить:
```ts
listInitiatives(spaceType, spaceId): Promise<Initiative[]>
createInitiative(spaceType, spaceId, input): Promise<Initiative>
joinInitiative(id): Promise<{ goal_id: number }>
getPendingProofs(id): Promise<PendingProof[]>
approveProof(initiativeId, checkinId, comment?): Promise<void>
getInitiativeDetail(id): Promise<{ initiative: Initiative, participants: ParticipantProgress[] }>
```

---

### Шаг 6 — Frontend: компоненты

**`web/components/product/initiative-list.tsx`**  
`InitiativeList({ spaceType, spaceId })` — карточки инициатив.  
Каждая карточка: заголовок, proof_criteria (collapsed), N участников, активность, pending count.  
Состояния: `not-joined` (кнопка ПРИСОЕДИНИТЬСЯ) / `joined` (badge + серия + pending count).  
Кнопка «+ СОЗДАТЬ ИНИЦИАТИВУ» — открывает форму создания (inline expand или modal).

**`web/components/product/initiative-detail.tsx`**  
`InitiativeDetail({ initiativeId })` — страница инициативы.  
Два раздела: peer-review очередь (pending proofs) + прогресс участников (бары).  
Pending section скрыт если очередь пуста.

**`web/components/product/initiative-peer-review.tsx`**  
`InitiativePeerReview({ initiativeId })` — изолированный компонент очереди.  
На каждый proof: имя автора, время, текст, опциональное вложение, кнопки ОДОБРИТЬ / УТОЧНИТЬ.  
УТОЧНИТЬ раскрывает textarea (как у buddy) — отправка блокируется без текста.

---

### Шаг 7 — Frontend: страница пространства

**`web/app/(product)/spaces/[type]/[id]/page.tsx`**  
Новая страница для `/spaces/teamspace/123` и `/spaces/community/456`.  
Вкладки: УЧАСТНИКИ | КРУГИ | ИНИЦИАТИВЫ | АНАЛИТИКА.  
Активная вкладка через `searchParams.tab`.  
При `tab=initiatives` рендерит `<InitiativeList spaceType={type} spaceId={id} />`.

---

## Порядок и зависимости

```
Шаг 1 (миграция)
    ↓
Шаг 3a (goals domain — поле initiative_id)
    ↓
Шаг 2 (initiatives package — domain + ports + service + repo + handler)
    ↓
Шаг 3b (checkins service — bypass buddy guard)
    ↓
Шаг 4 (api.go — wire up)
    ↓
Шаг 5 (frontend types + api client)
    ↓
Шаг 6 (компоненты)
    ↓
Шаг 7 (страница пространства)
```

---

## Файлы которые не трогать

- Существующие миграции 00001–00025
- `backend/internal/goals/service.go` — не менять бизнес-логику, только domain.go + repo
- `web/lib/types.ts` — только добавлять, не переименовывать существующие типы
