# Спек 4.4 — Борд достижений + Приватный борд «Нужна помощь»

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 4 · Лидерборды Backend  
**Зависимости:** check_ins, proof_contracts, pacts  
**Сложность:** S  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

**Борд достижений** — публичная витрина milestone: первый пруф, 4 недели подряд, стал buddy, завершил сезон. Признание через достижения, а не ранги.

**Борд «Нужна помощь»** — приватный инструмент тимлида/лидера сообщества. Кто застрял, кто ждёт ревью, какие круги затухают. Инструмент поддержки, не наказания.

---

## API

### GET /v1/me/achievements

Достижения текущего пользователя.

```
GET /v1/me/achievements
```

**Response 200:**
```json
{
  "data": {
    "unlocked": [
      {
        "id": "first_proof",
        "title": "Первый пруф",
        "description": "Сдал первое доказательство прогресса",
        "unlocked_at": "2026-01-20T14:00:00Z",
        "icon": "proof"
      },
      {
        "id": "streak_4_weeks",
        "title": "4 недели подряд",
        "description": "Не пропустил ни одной недели",
        "unlocked_at": "2026-02-17T10:00:00Z",
        "icon": "streak"
      }
    ],
    "locked": [
      {
        "id": "streak_8_weeks",
        "title": "8 недель подряд",
        "description": "Сдавай пруфы каждую неделю",
        "progress": 4,
        "target": 8,
        "icon": "streak"
      }
    ],
    "recent_unlock": {
      "id": "became_buddy",
      "title": "Стал buddy",
      "description": "Помог кому-то не слиться",
      "unlocked_at": "2026-05-01T09:00:00Z"
    }
  }
}
```

`recent_unlock` — последнее разблокированное достижение, для показа в confirmation moment (DNA Раздел 4).

### Каталог достижений

| ID | Триггер | Условие |
|----|---------|---------|
| `first_proof` | check-in approved | Первый approved check-in |
| `streak_4_weeks` | weekly cron | 4 consecutive weeks с approved check-in |
| `streak_8_weeks` | weekly cron | 8 consecutive weeks |
| `streak_12_weeks` | weekly cron | 12 consecutive weeks |
| `helped_5_people` | check-in review | Дал ревью ≥5 разным пользователям |
| `became_buddy` | pact created | Стал buddy первый раз |
| `became_mentor` | community_memberships | Назначен как potential_mentor лидером |
| `season_completed` | challenge goal | Все proof_contracts fulfilled для challenge цели |
| `first_public_artifact` | check-in | check-in с proof_type=artifact и approved |
| `ipr_goal_linked` | goal created | Создал цель с movement_mode=regular_rhythm и тегом ИПР |

### SQL: таблица достижений

```sql
-- 00025_achievements.sql
CREATE TABLE user_achievements (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL REFERENCES users(id),
  achievement_id  TEXT NOT NULL,
  unlocked_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  acknowledged    BOOLEAN NOT NULL DEFAULT FALSE,
  UNIQUE(user_id, achievement_id)
);
```

### Воркер разблокировки достижений

Добавить в `backend/cmd/worker/` задачу `checkAchievements`, запускается:
- После каждого approved check-in (event-driven)
- Раз в день для streak-достижений

---

### GET /v1/teamspaces/:id/needs-help

**Приватный** — только teamspace_lead.

```
GET /v1/teamspaces/:id/needs-help
Требует: teamspace_lead
```

**Response 200:**
```json
{
  "data": {
    "summary": {
      "members_needing_attention": 2,
      "pending_reviews_total": 4,
      "broken_contracts_total": 1
    },
    "members": [
      {
        "user_id": "uuid",
        "display_name": "Алексей Соколов",
        "avatar_url": "...",
        "days_since_last_proof": 14,
        "has_broken_contract": true,
        "pending_reviews_count": 0,
        "attention_reason": "no_proof_14_days"
      },
      {
        "user_id": "uuid",
        "display_name": "Мария Иванова",
        "avatar_url": "...",
        "days_since_last_proof": 3,
        "has_broken_contract": false,
        "pending_reviews_count": 2,
        "attention_reason": "waiting_for_review"
      }
    ],
    "at_risk_circles": [
      {
        "circle_id": "uuid",
        "circle_name": "Группа В",
        "active_members": 2,
        "total_members": 6,
        "last_proof_at": "2026-04-22T10:00:00Z"
      }
    ]
  }
}
```

`attention_reason` values: `"no_proof_14_days"` | `"no_proof_7_days"` | `"broken_contract"` | `"waiting_for_review"`

`at_risk_circles`: круги где active_members / total_members < 0.5.

---

### GET /v1/community-spaces/:id/needs-help

Аналогичный endpoint для community_leader. Отличие: нет данных об ИПР, только активность.

```
GET /v1/community-spaces/:id/needs-help
Требует: community_leader
```

Структура ответа аналогична teamspace, но без `has_ipr_goal`.

---

## Структура файлов

```
backend/internal/leaderboards/
  achievements.go  ← GetAchievements(), checkAndUnlock()
  needs_help.go    ← GetNeedsHelp() для teamspace и community
  http_handler.go  ← endpoints

backend/internal/worker/
  achievements.go  ← checkAchievements cron task

backend/migrations/
  00025_achievements.sql
```

---

## Acceptance Criteria

- [ ] `GET /v1/me/achievements` возвращает unlocked + locked достижения
- [ ] `locked` содержит `progress` и `target` для streak достижений
- [ ] `recent_unlock` — только последнее, null если нет нового за последние 24ч
- [ ] Воркер разблокирует `first_proof` при первом approved check-in
- [ ] Воркер разблокирует `streak_4_weeks` корректно (проверять consecutive weeks)
- [ ] `GET /v1/teamspaces/:id/needs-help` → 403 для не-лидеров
- [ ] `needs-help`: участники с ≥7 днями без пруфа попадают в список
- [ ] `at_risk_circles`: только круги с < 50% активности
- [ ] Достижение не разблокируется повторно (UNIQUE constraint)

---

## Что нельзя делать

- Не делать `/needs-help` публичным — только lead/leader
- Не показывать email или токены пользователей в needs-help
- Не создавать достижение "самый слабый" или "последний" — только позитивные
