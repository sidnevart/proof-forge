# Спек 3.3 — Аналитика для лидера Teamspace

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 3 · Backend метрики  
**Зависимости:** Спек 1.1, 2.1, 2.2  
**Сложность:** M  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Тимлид видит **энергию команды**, но не получает инструмент давления. Аналитика отвечает на вопросы:
- Кто активно двигается?
- Какие темы вызывают интерес?
- Где нужна помощь? (агрегированно, без имён)
- Какие пруфы пригодны для ИПР?

**Принцип DNA П5:** риски показываем агрегированно, без публичного ранжирования участников снизу.

---

## API

### GET /v1/teamspaces/:id/analytics

```
GET /v1/teamspaces/:id/analytics?period=last_4_weeks
Authorization: Bearer {token}
Требует роль: teamspace_lead или trusted_approver
```

**Query params:**
- `period`: `last_week` | `last_4_weeks` (default) | `last_12_weeks`

**Response 200:**
```json
{
  "data": {
    "period": "last_4_weeks",
    "teamspace_id": "uuid",
    "teamspace_name": "Команда персонализации",
    "summary": {
      "active_members": 7,
      "total_members": 9,
      "total_proofs_submitted": 23,
      "total_proofs_approved": 19,
      "avg_approval_time_hours": 18.4
    },
    "activity_trend": [
      { "week": "2026-W17", "proofs_count": 5, "active_members": 6 },
      { "week": "2026-W18", "proofs_count": 7, "active_members": 7 },
      { "week": "2026-W19", "proofs_count": 6, "active_members": 5 },
      { "week": "2026-W20", "proofs_count": 5, "active_members": 6 }
    ],
    "popular_topics": [
      { "tag": "Kotlin", "proof_count": 8, "member_count": 4 },
      { "tag": "AI-инструменты", "proof_count": 6, "member_count": 5 },
      { "tag": "System Design", "proof_count": 5, "member_count": 3 }
    ],
    "ipr_eligible_proofs": 12,
    "attention_needed": {
      "count": 2,
      "description": "2 участника не сдавали пруф более 2 недель"
    },
    "best_proofs_preview": [
      {
        "check_in_id": "uuid",
        "goal_title": "Kotlin Coroutines",
        "proof_preview": "Демо SupervisorJob — показал как...",
        "submitted_at": "2026-05-06T10:00:00Z",
        "user_display_name": "Артём Козлов"
      }
    ]
  }
}
```

**Важно:** `attention_needed` содержит только число и описание, **без имён и user_id** (DNA П5 — риски приватны).

`best_proofs_preview` — топ-3 пруфа по сроку (последние), имена показываем только если пользователь согласился на visibility (по умолчанию показываем).

---

### GET /v1/teamspaces/:id/analytics/members

Детальная активность **по конкретному участнику** — доступна только если сам участник дал согласие на sharing с лидером (consent в team_memberships).

```
GET /v1/teamspaces/:id/analytics/members
Authorization: Bearer {token}
Требует роль: teamspace_lead
```

**Response 200:**
```json
{
  "data": [
    {
      "user_id": "uuid",
      "display_name": "Артём Козлов",
      "avatar_url": "...",
      "proofs_count": 6,
      "active_weeks": 4,
      "goals_count": 2,
      "has_ipr_goal": true,
      "consent_shared": true
    },
    {
      "user_id": "uuid",
      "display_name": "Мария Иванова",
      "proofs_count": 0,
      "active_weeks": 0,
      "goals_count": 1,
      "has_ipr_goal": false,
      "consent_shared": false
    }
  ]
}
```

Для участников с `consent_shared: false` — показываем только `user_id`, `display_name`, `consent_shared: false`. Остальные поля — null/скрыты.

---

## Структура файлов

```
backend/internal/analytics/
  teamspace.go     ← GetTeamspaceAnalytics(), GetMemberActivity()
  queries.go       ← SQL
  http_handler.go  ← endpoints выше
```

### Ключевой SQL — popular_topics

```sql
-- Популярные темы по skill tags из check-ins
SELECT 
    unnest(string_to_array(ci.skill_tags, ',')) as tag,
    COUNT(ci.id) as proof_count,
    COUNT(DISTINCT g.user_id) as member_count
FROM check_ins ci
JOIN goals g ON g.id = ci.goal_id
JOIN teamspace_memberships tm ON tm.user_id = g.user_id 
    AND tm.teamspace_id = $1
    AND tm.status = 'active'
WHERE ci.created_at > NOW() - INTERVAL '1 week' * $2
  AND ci.status = 'approved'
GROUP BY tag
ORDER BY proof_count DESC
LIMIT 10
```

*Примечание: поле `skill_tags` добавить к check-ins в отдельном спеке если его ещё нет. Пока использовать `title` из goal как заменитель.*

---

## Acceptance Criteria

- [ ] 403 для не-лидеров teamspace
- [ ] `attention_needed.count` корректен (без имён)
- [ ] `popular_topics` сортированы по proof_count DESC
- [ ] `activity_trend` возвращает данные по ISO-неделям
- [ ] `/members` показывает только пользователей с `consent_shared: true` в полном объёме
- [ ] `period=last_week` возвращает только данные за текущую неделю
- [ ] Пустой teamspace (0 members) — корректные нули, не ошибка

---

## Что нельзя делать

- Не показывать имена участников в `attention_needed` — только агрегат
- Не давать trusted_approver доступ к `/members` — только teamspace_lead
- Не вычислять "рейтинг" участников — только активность в абсолютных числах
