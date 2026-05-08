# Спек 1.2 — Схема Community Space и расширение Circle

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 1 · Данные  
**Зависимости:** Спек 1.1 (workspaces + teamspaces должны существовать)  
**Сложность:** M  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

**Community Space** — пространство внешнего или внутреннего сообщества: Telegram-клуб по ML, закрытый клуб backend-разработчиков, спортивное сообщество. В отличие от teamspace, community space ориентирован на добровольное участие, сезоны и соревновательные челленджи.

Circle (уже существует в БД) нужно научить быть привязанным к одному из трёх контекстов: workspace/teamspace/community space. Сейчас circle — автономная сущность без пространства.

**Кто создаёт community space:**
- workspace_owner или platform_admin
- Или независимо, без workspace (standalone community)

**Кто управляет:**
- community_leader — лидер сообщества

---

## Что реализовать

### Backend
1. SQL миграция `00019_community_spaces.sql`
2. Пакет `backend/internal/communityspaces/` — domain, repository, service, handler
3. Расширение таблицы `circles` — добавить nullable FK на teamspace/community_space
4. Роуты в `api.go`

### Не реализовывать в этом спеке
- UI страниц community space (спеки 5.x и 6.x)
- Аналитика сообщества (спек 3.4)
- Лидерборды community (спеки 4.x)

---

## Схема базы данных

```sql
-- +goose Up

CREATE TABLE community_spaces (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id    UUID REFERENCES workspaces(id) ON DELETE SET NULL, -- nullable: standalone community
  name            TEXT NOT NULL,
  slug            TEXT NOT NULL UNIQUE,
  leader_user_id  UUID NOT NULL REFERENCES users(id),
  type            TEXT NOT NULL DEFAULT 'private'
                  CHECK (type IN ('private', 'public', 'paid')),
  description     TEXT,
  avatar_url      TEXT,
  invite_code     TEXT UNIQUE,
  is_active       BOOLEAN NOT NULL DEFAULT TRUE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX community_spaces_slug_idx ON community_spaces(slug);
CREATE INDEX community_spaces_workspace_idx ON community_spaces(workspace_id) WHERE workspace_id IS NOT NULL;

CREATE TABLE community_memberships (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  community_space_id  UUID NOT NULL REFERENCES community_spaces(id) ON DELETE CASCADE,
  user_id             UUID NOT NULL REFERENCES users(id),
  role                TEXT NOT NULL DEFAULT 'member'
                      CHECK (role IN ('leader', 'member', 'observer')),
  status              TEXT NOT NULL DEFAULT 'active'
                      CHECK (status IN ('active', 'left', 'removed')),
  joined_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(community_space_id, user_id)
);

CREATE UNIQUE INDEX community_single_leader_idx
  ON community_memberships(community_space_id)
  WHERE role = 'leader' AND status = 'active';

-- Расширение circles: добавить контекст пространства
ALTER TABLE circles
  ADD COLUMN teamspace_id       UUID REFERENCES teamspaces(id) ON DELETE SET NULL,
  ADD COLUMN community_space_id UUID REFERENCES community_spaces(id) ON DELETE SET NULL;

-- Circle может быть привязан только к одному пространству
ALTER TABLE circles
  ADD CONSTRAINT circle_single_space_ctx
  CHECK (
    (teamspace_id IS NULL AND community_space_id IS NULL) OR  -- standalone circle
    (teamspace_id IS NOT NULL AND community_space_id IS NULL) OR
    (teamspace_id IS NULL AND community_space_id IS NOT NULL)
  );

CREATE INDEX circles_teamspace_idx ON circles(teamspace_id) WHERE teamspace_id IS NOT NULL;
CREATE INDEX circles_community_idx ON circles(community_space_id) WHERE community_space_id IS NOT NULL;

-- +goose Down
ALTER TABLE circles DROP COLUMN IF EXISTS teamspace_id;
ALTER TABLE circles DROP COLUMN IF EXISTS community_space_id;
DROP TABLE IF EXISTS community_memberships;
DROP TABLE IF EXISTS community_spaces;
```

---

## Go Domain (`backend/internal/communityspaces/domain.go`)

```go
package communityspaces

import (
    "time"
    "github.com/google/uuid"
)

type CommunityType string

const (
    CommunityTypePrivate CommunityType = "private"
    CommunityTypePublic  CommunityType = "public"
    CommunityTypePaid    CommunityType = "paid"
)

type CommunityRole string

const (
    CommunityRoleLeader   CommunityRole = "leader"
    CommunityRoleMember   CommunityRole = "member"
    CommunityRoleObserver CommunityRole = "observer"
)

type CommunitySpace struct {
    ID            uuid.UUID
    WorkspaceID   *uuid.UUID // nil = standalone
    Name          string
    Slug          string
    LeaderUserID  uuid.UUID
    Type          CommunityType
    Description   string
    AvatarURL     string
    InviteCode    string
    IsActive      bool
    CreatedAt     time.Time
}

type CommunityMembership struct {
    ID                UUID
    CommunitySpaceID  uuid.UUID
    UserID            uuid.UUID
    Role              CommunityRole
    Status            MembershipStatus // переиспользуем из workspaces
    JoinedAt          time.Time
}

func CanManageCommunity(role CommunityRole) bool {
    return role == CommunityRoleLeader
}
```

---

## API Contract

```
POST   /v1/community-spaces              — создать
GET    /v1/community-spaces              — мои + публичные
GET    /v1/community-spaces/:slug        — детали по slug
PATCH  /v1/community-spaces/:id         — обновить (leader only)
POST   /v1/community-spaces/:id/join    — вступить по invite_code
POST   /v1/community-spaces/:id/members — добавить напрямую (leader only)

-- В рамках workspace:
GET    /v1/workspaces/:id/community-spaces — community spaces внутри workspace
```

**POST /v1/community-spaces — Request:**
```json
{
  "name": "ML Club",
  "slug": "ml-club",
  "type": "private",
  "description": "Закрытый клуб ML-практиков",
  "workspace_id": "uuid-or-null"
}
```

**Response 201:**
```json
{
  "data": {
    "id": "uuid",
    "name": "ML Club",
    "slug": "ml-club",
    "type": "private",
    "leader_user_id": "uuid",
    "invite_code": "ML-XKQW",
    "is_active": true,
    "my_membership": { "role": "leader", "status": "active" }
  }
}
```

**POST /v1/community-spaces/:id/join — Request:**
```json
{ "invite_code": "ML-XKQW" }
```

**Response 200:**
```json
{
  "data": {
    "community_space_id": "uuid",
    "role": "member",
    "status": "active"
  }
}
```

**GET /v1/circles/:id — существующий endpoint расширяется:**
```json
{
  "data": {
    "id": "uuid",
    "name": "Группа А",
    "teamspace_id": "uuid-or-null",
    "community_space_id": "uuid-or-null",
    "context": "teamspace"  // "teamspace" | "community" | "standalone"
  }
}
```

---

## Структура файлов

```
backend/internal/communityspaces/
  domain.go
  postgres_repository.go
  service.go
  http_handler.go

backend/migrations/
  00019_community_spaces.sql
```

### Регистрация роутов

```go
communityHandler := communityspaces.NewHTTPHandler(communitySvc)

r.Route("/v1/community-spaces", func(r chi.Router) {
    r.Use(authMiddleware.RequireAuth)
    r.Post("/", communityHandler.Create)
    r.Get("/", communityHandler.List)
    r.Get("/{slug}", communityHandler.GetBySlug)
    r.Patch("/{id}", communityHandler.Update)
    r.Post("/{id}/join", communityHandler.Join)
    r.Post("/{id}/members", communityHandler.AddMember)
})
```

---

## Генерация invite_code

```go
// 4 символа: 2 буквы (из slug) + 4 случайных заглавных
func generateInviteCode(slug string) string {
    prefix := strings.ToUpper(slug[:min(2, len(slug))])
    const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
    suffix := make([]byte, 4)
    for i := range suffix {
        suffix[i] = chars[rand.Intn(len(chars))]
    }
    return prefix + "-" + string(suffix)
}
```

---

## Ошибки и валидация

| Ситуация | HTTP | Код |
|----------|------|-----|
| Неверный invite_code | 422 | `invalid_invite_code` |
| Уже является участником | 409 | `already_member` |
| slug занят | 409 | `slug_taken` |
| Пространство неактивно | 410 | `space_inactive` |

---

## Acceptance Criteria

- [ ] Community space создаётся, создатель получает роль leader
- [ ] invite_code генерируется автоматически при создании
- [ ] `POST /v1/community-spaces/:id/join` с правильным кодом — добавляет участником
- [ ] `POST /v1/community-spaces/:id/join` с неверным кодом → 422
- [ ] Circles GET endpoint возвращает `teamspace_id`, `community_space_id`, `context`
- [ ] Ограничение: circle не может иметь одновременно teamspace_id и community_space_id → 422 при попытке создать
- [ ] Миграция `goose up`/`goose down` без ошибок
- [ ] Существующие circles не ломаются (teamspace_id и community_space_id = NULL для всех существующих)

---

## Что нельзя делать

- Не удалять существующие поля из circles
- Не требовать workspace_id для community space (standalone должен работать)
- Не реализовывать paid-логику (subscriptions, платежи) — только поле type='paid'
