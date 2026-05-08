# Спек 3.5 — Platform Admin: метрики и управление

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 3 · Backend метрики  
**Зависимости:** Спек 2.1 (platform_admin роль), 1.1, 1.2  
**Сложность:** M  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Platform admin — единственный кто видит бизнес-метрики всей платформы: сколько workspace активно, где риск churn, техническое здоровье. Также управляет workspace, назначает роли, замораживает пространства.

---

## API

### GET /v1/admin/stats

```
GET /v1/admin/stats
Authorization: Bearer {token}
Требует: is_platform_admin = true
```

**Response 200:**
```json
{
  "data": {
    "platform": {
      "total_users": 1240,
      "active_users_last_30d": 387,
      "total_workspaces": 8,
      "active_workspaces": 6,
      "total_teamspaces": 23,
      "total_community_spaces": 12,
      "total_proofs_last_30d": 2341,
      "avg_proofs_per_active_user": 6.0
    },
    "health": [
      {
        "workspace_id": "uuid",
        "workspace_name": "T-Bank AI Stream",
        "health_score": 82,
        "active_members": 67,
        "total_members": 80,
        "proofs_last_30d": 312,
        "churn_risk": "low"
      },
      {
        "workspace_id": "uuid",
        "workspace_name": "ML Community",
        "health_score": 41,
        "active_members": 12,
        "total_members": 45,
        "proofs_last_30d": 23,
        "churn_risk": "high"
      }
    ],
    "revenue": {
      "paid_workspaces_count": 3,
      "trial_workspaces_count": 2,
      "free_workspaces_count": 3
    }
  }
}
```

**health_score формула:**
```
health_score = (active_members / total_members * 50) + 
               (min(proofs_last_30d / total_members, 10) / 10 * 50)
```
Значение 0–100. `churn_risk`: `"low"` (≥70), `"medium"` (40–69), `"high"` (<40).

---

### GET /v1/admin/workspaces

Список всех workspace с управляющими действиями.

```
GET /v1/admin/workspaces?status=active
```

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "T-Bank AI Stream",
      "slug": "t-bank-ai",
      "type": "org",
      "owner": { "user_id": "uuid", "display_name": "Иван Петров", "email": "i@..." },
      "is_active": true,
      "teamspaces_count": 5,
      "members_count": 80,
      "created_at": "2026-01-15T00:00:00Z",
      "last_activity_at": "2026-05-07T18:00:00Z",
      "health_score": 82,
      "churn_risk": "low"
    }
  ],
  "meta": { "total": 8 }
}
```

---

### POST /v1/admin/workspaces/:id/freeze

Заморозить workspace — пользователи не могут создавать новые цели и пруфы, но данные сохранены.

```
POST /v1/admin/workspaces/:id/freeze
Body: { "reason": "неоплаченный счёт" }
```

**Response 200:** `{ "data": { "id": "uuid", "is_active": false, "frozen_reason": "..." } }`

**Поведение:** добавить поля `frozen_at TIMESTAMPTZ` и `frozen_reason TEXT` в таблицу workspaces (добавить в миграцию 00018 или отдельную 00023).

---

### POST /v1/admin/workspaces/:id/unfreeze

```
POST /v1/admin/workspaces/:id/unfreeze
```

---

### GET /v1/admin/users

Список пользователей с поиском.

```
GET /v1/admin/users?q=иван&limit=20&offset=0
```

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "display_name": "Иван Петров",
      "email": "ivan@...",
      "is_platform_admin": false,
      "created_at": "2026-01-10T00:00:00Z",
      "last_active_at": "2026-05-07T10:00:00Z",
      "goals_count": 3,
      "proofs_count": 28
    }
  ],
  "meta": { "total": 1240, "limit": 20, "offset": 0 }
}
```

---

### POST /v1/admin/users/:id/make-admin / DELETE /v1/admin/users/:id/make-admin

Описан в спеке 2.1.

---

## Схема расширений для заморозки

```sql
-- Добавить в миграцию 00023_workspace_freeze.sql
ALTER TABLE workspaces
  ADD COLUMN frozen_at     TIMESTAMPTZ,
  ADD COLUMN frozen_reason TEXT;
```

---

## Структура файлов

```
backend/internal/admin/
  stats.go         ← GetPlatformStats()
  workspaces.go    ← ListWorkspaces(), Freeze(), Unfreeze()
  users.go         ← ListUsers(), MakeAdmin()
  http_handler.go  ← все admin endpoints
```

Все admin handlers за `RequirePlatformAdmin` middleware (спек 2.1).

---

## Acceptance Criteria

- [ ] 403 для не-platform_admin на всех `/v1/admin/*`
- [ ] `health_score` вычисляется по формуле, значение 0–100
- [ ] `churn_risk` корректно классифицируется
- [ ] Freeze: `is_active = false`, данные не удаляются
- [ ] После freeze пользователи workspace получают 403 при попытке создать goal/check-in
- [ ] Unfreeze восстанавливает доступ
- [ ] `/v1/admin/users?q=` ищет по `display_name` и `email` (ILIKE)
- [ ] Все list endpoints поддерживают `limit` (max 100) и `offset`

---

## Что нельзя делать

- Не давать platform_admin изменять личные данные пользователей (только роли)
- Не удалять данные при freeze — только блокировать создание нового контента
- Не показывать пароли или токены сессий в admin API
