# Спек 2.1 — Расширенная система ролей

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 2 · Роли и приватность  
**Зависимости:** Спек 1.1 (workspaces + teamspaces)  
**Сложность:** M  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Существующие роли (`lead`, `trusted_approver`, `member`) недостаточны для v2. Нужны:

| Роль | Scope | Что может |
|------|-------|-----------|
| `platform_admin` | Вся платформа | Всё: создавать workspace, выдавать роли, смотреть аналитику, freeze |
| `workspace_owner` | Один workspace | Создавать teamspace/community, назначать leads, смотреть агрегированную аналитику |
| `community_leader` | Один community space | Управлять сообществом, видеть аналитику, публиковать витрину |
| `teamspace_lead` | Один teamspace | Управлять командой, видеть аналитику, приватный борд "нужна помощь" |
| `trusted_approver` | Один teamspace | Проверять пруфы участников |
| `buddy` | Конкретный пакт | Проверять пруфы своего подопечного |
| `member` | Space | Участник, создаёт цели и сдаёт пруфы |

**Важно:** роли не исключают друг друга. Пользователь может быть `workspace_owner` в одном workspace и `member` в другом. `platform_admin` — это глобальная метка на пользователе.

---

## Что реализовать

### Backend
1. SQL миграция `00022_platform_admins.sql` — добавить флаг platform_admin на users
2. Обновить `backend/internal/auth/` — включать роли в JWT claims
3. Создать `backend/internal/authz/` — centralized authorization service
4. Обновить middleware в `backend/internal/platform/`

### Не реализовывать
- Admin panel UI (спек 6.4)
- Workspace management UI (спек 5.1)

---

## Схема базы данных

```sql
-- +goose Up

-- Platform admin — глобальный флаг на пользователе
ALTER TABLE users
  ADD COLUMN is_platform_admin BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX users_platform_admin_idx ON users(id) WHERE is_platform_admin = TRUE;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS is_platform_admin;
```

*workspace_owner определяется через workspaces.owner_user_id (спек 1.1)*  
*teamspace_lead определяется через teamspace_memberships.role = 'lead' (спек 1.1)*  
*community_leader определяется через community_memberships.role = 'leader' (спек 1.2)*

---

## JWT Claims

Обновить `backend/internal/auth/` чтобы при выдаче токена включать глобальные роли:

```go
// Структура JWT claims (добавить к существующим):
type Claims struct {
    UserID          string   `json:"sub"`
    IsPlatformAdmin bool     `json:"is_platform_admin"`
    // Роли в конкретных пространствах — НЕ включать в JWT
    // (слишком динамичны, проверяются per-request из БД)
}
```

**Роли в пространствах (teamspace, community, workspace) НЕ кешируются в JWT** — они проверяются из БД при каждом запросе, так как могут измениться в любой момент.

---

## Centralized Authz (`backend/internal/authz/authz.go`)

```go
package authz

import (
    "context"
    "github.com/google/uuid"
)

type Authorizer struct {
    db Database // interface для запросов
}

// Проверить что пользователь является platform admin
func (a *Authorizer) IsPlatformAdmin(ctx context.Context, userID uuid.UUID) (bool, error)

// Проверить роль в workspace
func (a *Authorizer) WorkspaceRole(ctx context.Context, userID, workspaceID uuid.UUID) (WorkspaceRole, error)
// WorkspaceRole: "owner" | "member" | "none"

// Проверить роль в teamspace
func (a *Authorizer) TeamspaceRole(ctx context.Context, userID, teamspaceID uuid.UUID) (teamspaces.TeamspaceRole, error)
// Возвращает "" если не участник

// Проверить роль в community
func (a *Authorizer) CommunityRole(ctx context.Context, userID, communityID uuid.UUID) (communityspaces.CommunityRole, error)

// Удобные helpers:
func (a *Authorizer) CanManageTeamspace(ctx context.Context, userID, teamspaceID uuid.UUID) (bool, error)
func (a *Authorizer) CanApproveProofs(ctx context.Context, userID, teamspaceID uuid.UUID) (bool, error)
func (a *Authorizer) CanViewTeamAnalytics(ctx context.Context, userID, teamspaceID uuid.UUID) (bool, error)
```

---

## Middleware

**`backend/internal/platform/middleware/`** — добавить:

```go
// RequirePlatformAdmin — 403 если не platform_admin
func RequirePlatformAdmin(authz *authz.Authorizer) func(http.Handler) http.Handler

// RequireWorkspaceOwner — 403 если не owner конкретного workspace
// Берёт workspace_id из URL параметра
func RequireWorkspaceOwner(authz *authz.Authorizer) func(http.Handler) http.Handler

// RequireTeamspaceLead — 403 если не lead конкретного teamspace
func RequireTeamspaceLead(authz *authz.Authorizer) func(http.Handler) http.Handler
```

### Применение middleware в роутах

```go
// admin routes
r.Route("/v1/admin", func(r chi.Router) {
    r.Use(authMiddleware.RequireAuth)
    r.Use(adminMiddleware.RequirePlatformAdmin(authz))
    // ... admin handlers
})

// workspace management
r.Patch("/v1/workspaces/{id}", 
    authMiddleware.RequireAuth,
    wsMiddleware.RequireWorkspaceOwner(authz),
    workspaceHandler.Update,
)
```

---

## API: назначить platform admin

Только существующий platform_admin может назначить другого:

```
POST /v1/admin/users/:user_id/make-admin
DELETE /v1/admin/users/:user_id/make-admin
```

**Request:** пустое тело (действие идемпотентно)  
**Response 200:** `{ "data": { "user_id": "uuid", "is_platform_admin": true } }`

---

## Acceptance Criteria

- [ ] Таблица `users` имеет поле `is_platform_admin`
- [ ] JWT включает `is_platform_admin` claim
- [ ] `RequirePlatformAdmin` middleware возвращает 403 для не-admin пользователей
- [ ] `RequireTeamspaceLead` проверяет роль из БД, а не из JWT
- [ ] `POST /v1/admin/users/:id/make-admin` работает только для platform_admin → 403 иначе
- [ ] Все существующие `/v1/teams` endpoints продолжают работать (не сломаны)
- [ ] Authz helpers покрыты unit-тестами (мок БД)

---

## Что нельзя делать

- Не кешировать space-роли в JWT (они меняются динамически)
- Не удалять существующую авторизационную логику из `teams/authz.go` — перенести или обернуть
- Не добавлять роли напрямую в JWT кроме `is_platform_admin`
