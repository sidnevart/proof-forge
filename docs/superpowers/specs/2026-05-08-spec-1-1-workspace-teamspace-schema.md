# Спек 1.1 — Схема Workspace и Teamspace

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 1 · Данные  
**Зависимости:** нет (первая миграция серии)  
**Сложность:** M  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

ProofForge вводит иерархию пространств вместо плоской модели "команды".

- **Workspace** — верхний уровень: организация (T-Bank AI Stream) или крупное сообщество
- **Teamspace** — пространство конкретной команды внутри workspace (команда персонализации)

Без этих сущностей нельзя разграничивать аналитику, роли и видимость данных между командами одной организации.

**Кто создаёт:**
- Workspace создаёт platform_admin или workspace_owner (через admin panel)
- Teamspace создаёт workspace_owner или назначенный teamspace_lead

**Кто использует:**
- Teamspace_lead — управляет пространством команды
- Member — участник, видит цели и proof'ы своего teamspace
- Buddy — может быть из другого teamspace того же workspace

---

## Что реализовать

### Backend
1. SQL миграция `00018_workspaces.sql` — таблицы `workspaces` и `teamspaces`
2. Go пакет `backend/internal/workspaces/` с domain, repository, service, http_handler
3. Роуты в `backend/internal/platform/app/api.go`

### Не реализовывать в этом спеке
- Community space (спек 1.2)
- Workspace admin panel UI (спек 6.4)
- Visibility rules (спек 2.2)

---

## Схема базы данных

```sql
-- +goose Up

CREATE TABLE workspaces (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name            TEXT NOT NULL,
  slug            TEXT NOT NULL UNIQUE,          -- url-friendly id: "t-bank-ai"
  type            TEXT NOT NULL DEFAULT 'org'    -- 'org' | 'community'
                  CHECK (type IN ('org', 'community')),
  owner_user_id   UUID NOT NULL REFERENCES users(id),
  description     TEXT,
  avatar_url      TEXT,
  is_active       BOOLEAN NOT NULL DEFAULT TRUE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX workspaces_slug_idx ON workspaces(slug);
CREATE INDEX workspaces_owner_idx ON workspaces(owner_user_id);

CREATE TABLE teamspaces (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  name            TEXT NOT NULL,
  slug            TEXT NOT NULL,                 -- уникален внутри workspace
  lead_user_id    UUID NOT NULL REFERENCES users(id),
  description     TEXT,
  is_active       BOOLEAN NOT NULL DEFAULT TRUE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(workspace_id, slug)
);

CREATE INDEX teamspaces_workspace_idx ON teamspaces(workspace_id);
CREATE INDEX teamspaces_lead_idx ON teamspaces(lead_user_id);

CREATE TABLE teamspace_memberships (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  teamspace_id    UUID NOT NULL REFERENCES teamspaces(id) ON DELETE CASCADE,
  user_id         UUID NOT NULL REFERENCES users(id),
  role            TEXT NOT NULL DEFAULT 'member'
                  CHECK (role IN ('lead', 'trusted_approver', 'member')),
  status          TEXT NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active', 'left', 'removed')),
  joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(teamspace_id, user_id)
);

-- Ровно один lead на teamspace (аналогично существующему teams)
CREATE UNIQUE INDEX teamspace_single_lead_idx
  ON teamspace_memberships(teamspace_id)
  WHERE role = 'lead' AND status = 'active';

-- +goose Down
DROP TABLE IF EXISTS teamspace_memberships;
DROP TABLE IF EXISTS teamspaces;
DROP TABLE IF EXISTS workspaces;
```

---

## Go Domain (`backend/internal/workspaces/domain.go`)

```go
package workspaces

import (
    "time"
    "github.com/google/uuid"
)

type WorkspaceType string

const (
    WorkspaceTypeOrg       WorkspaceType = "org"
    WorkspaceTypeCommunity WorkspaceType = "community"
)

type Workspace struct {
    ID          uuid.UUID
    Name        string
    Slug        string
    Type        WorkspaceType
    OwnerUserID uuid.UUID
    Description string
    AvatarURL   string
    IsActive    bool
    CreatedAt   time.Time
}

type TeamspaceRole string

const (
    TeamspaceRoleLead            TeamspaceRole = "lead"
    TeamspaceRoleTrustedApprover TeamspaceRole = "trusted_approver"
    TeamspaceRoleMember          TeamspaceRole = "member"
)

type MembershipStatus string

const (
    MembershipStatusActive  MembershipStatus = "active"
    MembershipStatusLeft    MembershipStatus = "left"
    MembershipStatusRemoved MembershipStatus = "removed"
)

type Teamspace struct {
    ID          uuid.UUID
    WorkspaceID uuid.UUID
    Name        string
    Slug        string
    LeadUserID  uuid.UUID
    Description string
    IsActive    bool
    CreatedAt   time.Time
}

type TeamspaceMembership struct {
    ID          uuid.UUID
    TeamspaceID uuid.UUID
    UserID      uuid.UUID
    Role        TeamspaceRole
    Status      MembershipStatus
    JoinedAt    time.Time
}

// Authz helpers
func CanManageTeamspace(role TeamspaceRole) bool {
    return role == TeamspaceRoleLead
}

func CanApproveProofs(role TeamspaceRole) bool {
    return role == TeamspaceRoleLead || role == TeamspaceRoleTrustedApprover
}
```

---

## API Contract

### Workspaces

```
POST   /v1/workspaces          — создать workspace
GET    /v1/workspaces          — мои workspaces
GET    /v1/workspaces/:slug    — workspace по slug
PATCH  /v1/workspaces/:id      — обновить (только owner)
DELETE /v1/workspaces/:id      — архивировать (только platform_admin)
```

**POST /v1/workspaces — Request:**
```json
{
  "name": "T-Bank AI Stream",
  "slug": "t-bank-ai",
  "type": "org",
  "description": "AI-инициативы и исследования"
}
```

**POST /v1/workspaces — Response 201:**
```json
{
  "data": {
    "id": "uuid",
    "name": "T-Bank AI Stream",
    "slug": "t-bank-ai",
    "type": "org",
    "owner_user_id": "uuid",
    "is_active": true,
    "created_at": "2026-05-08T10:00:00Z"
  }
}
```

**GET /v1/workspaces — Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "T-Bank AI Stream",
      "slug": "t-bank-ai",
      "type": "org",
      "my_role": "owner"
    }
  ]
}
```

### Teamspaces

```
POST   /v1/workspaces/:workspace_id/teamspaces       — создать teamspace
GET    /v1/workspaces/:workspace_id/teamspaces       — список teamspaces
GET    /v1/teamspaces/:id                            — teamspace детали + участники
POST   /v1/teamspaces/:id/members                   — добавить участника
PATCH  /v1/teamspaces/:id/members/:user_id          — изменить роль/статус
```

**POST /v1/workspaces/:workspace_id/teamspaces — Request:**
```json
{
  "name": "Команда персонализации",
  "slug": "personalization",
  "description": "Алгоритмы рекомендаций и A/B-тесты"
}
```

**GET /v1/teamspaces/:id — Response 200:**
```json
{
  "data": {
    "id": "uuid",
    "workspace_id": "uuid",
    "name": "Команда персонализации",
    "slug": "personalization",
    "lead_user_id": "uuid",
    "is_active": true,
    "my_membership": {
      "role": "lead",
      "status": "active"
    },
    "members": [
      {
        "user_id": "uuid",
        "display_name": "Артём Козлов",
        "avatar_url": "...",
        "role": "member",
        "status": "active"
      }
    ]
  }
}
```

---

## Структура файлов

```
backend/internal/workspaces/
  domain.go                    ← типы, authz helpers
  postgres_repository.go       ← CRUD для workspaces + teamspaces
  service.go                   ← бизнес-логика
  http_handler.go              ← HTTP handlers + validation
  
backend/migrations/
  00018_workspaces.sql         ← миграция выше
```

### Регистрация роутов (`api.go`)

```go
// В функции setupRoutes:
workspaceSvc := workspaces.NewService(workspaceRepo)
workspaceHandler := workspaces.NewHTTPHandler(workspaceSvc)

r.Route("/v1/workspaces", func(r chi.Router) {
    r.Use(authMiddleware.RequireAuth)
    r.Post("/", workspaceHandler.Create)
    r.Get("/", workspaceHandler.List)
    r.Get("/{slug}", workspaceHandler.GetBySlug)
    r.Patch("/{id}", workspaceHandler.Update)
    r.Delete("/{id}", workspaceHandler.Archive)
    r.Post("/{workspaceID}/teamspaces", workspaceHandler.CreateTeamspace)
    r.Get("/{workspaceID}/teamspaces", workspaceHandler.ListTeamspaces)
})
r.Route("/v1/teamspaces", func(r chi.Router) {
    r.Use(authMiddleware.RequireAuth)
    r.Get("/{id}", workspaceHandler.GetTeamspace)
    r.Post("/{id}/members", workspaceHandler.AddMember)
    r.Patch("/{id}/members/{userID}", workspaceHandler.UpdateMember)
})
```

---

## Ошибки и валидация

| Ситуация | HTTP | Код ошибки |
|----------|------|------------|
| slug уже занят | 409 | `slug_taken` |
| slug невалидный (не a-z0-9-) | 422 | `invalid_slug` |
| не owner/admin для создания workspace | 403 | `forbidden` |
| teamspace не найден | 404 | `teamspace_not_found` |
| попытка добавить второго lead | 409 | `lead_already_exists` |

**Формат slug:** только `[a-z0-9-]`, длина 2–50 символов. Валидация на бэкенде regexp `^[a-z0-9-]{2,50}$`.

---

## Acceptance Criteria

- [ ] `POST /v1/workspaces` создаёт workspace, возвращает 201 с телом
- [ ] `GET /v1/workspaces` возвращает только workspaces где пользователь является owner или member
- [ ] Slug уникален глобально для workspaces, уникален внутри workspace для teamspaces
- [ ] `POST /v1/workspaces/:id/teamspaces` создаёт teamspace, создатель автоматически получает роль lead
- [ ] Попытка создать второго lead в teamspace → 409 `lead_already_exists`
- [ ] `GET /v1/teamspaces/:id` возвращает `my_membership` с ролью текущего пользователя
- [ ] Миграция `goose up` и `goose down` выполняются без ошибок
- [ ] Все endpoints требуют аутентификации (401 без токена)

---

## Что нельзя делать

- Не смешивать существующую таблицу `teams` с новой `teamspaces` — это разные сущности
- Не создавать workspace_admin роль в этом спеке (будет в спеке 2.1)
- Не добавлять frontend в этом спеке
- Не реализовывать invite-flow для teamspace (отдельная задача)
