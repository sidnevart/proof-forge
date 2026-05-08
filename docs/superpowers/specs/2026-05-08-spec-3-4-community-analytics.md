# Спек 3.4 — Аналитика для лидера Community Space

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 3 · Backend метрики  
**Зависимости:** Спек 1.2 (community_spaces), 2.1  
**Сложность:** M  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Лидер сообщества смотрит на продукт через призму **ценности для участников и монетизации**. Его вопросы:
- Сколько людей дошли до конца сезона?
- Какие пруфы можно показать как витрину?
- Кто из участников может стать ментором?
- Готово ли сообщество платить за следующий сезон?

Аналитика должна давать материал для постов, обзоров и продаж следующего сезона.

---

## API

### GET /v1/community-spaces/:id/analytics

```
GET /v1/community-spaces/:id/analytics?period=current_season
Authorization: Bearer {token}
Требует роль: community_leader
```

**Response 200:**
```json
{
  "data": {
    "period": "current_season",
    "community_space_id": "uuid",
    "community_name": "ML Club",
    "summary": {
      "total_members": 42,
      "active_members": 31,
      "retention_pct": 73.8,
      "completion_rate_pct": 61.9,
      "total_proofs_this_season": 187,
      "avg_proofs_per_member": 6.0
    },
    "retention_trend": [
      { "week": "2026-W17", "active_members": 38, "new_members": 3, "left_members": 1 },
      { "week": "2026-W18", "active_members": 36, "new_members": 0, "left_members": 2 },
      { "week": "2026-W19", "active_members": 34, "new_members": 1, "left_members": 3 },
      { "week": "2026-W20", "active_members": 31, "new_members": 0, "left_members": 3 }
    ],
    "circle_engagement": [
      {
        "circle_id": "uuid",
        "circle_name": "Группа А",
        "members_count": 7,
        "proofs_count": 34,
        "completion_pct": 85.7,
        "status": "thriving"
      },
      {
        "circle_id": "uuid",
        "circle_name": "Группа Б",
        "members_count": 6,
        "proofs_count": 8,
        "completion_pct": 22.2,
        "status": "at_risk"
      }
    ],
    "best_proofs_week": [
      {
        "check_in_id": "uuid",
        "user_display_name": "Алина Петрова",
        "goal_title": "ML в продакшене",
        "proof_preview": "Задеплоила первую модель, вот метрики...",
        "likes_count": 12
      }
    ],
    "potential_mentors": [
      {
        "user_id": "uuid",
        "display_name": "Дмитрий Орлов",
        "proofs_count": 18,
        "helps_count": 7,
        "expertise_tags": ["PyTorch", "MLOps"]
      }
    ],
    "repeat_season_signals": {
      "completed_members": 26,
      "survey_intent_count": 0,
      "estimated_repeat_pct": null
    }
  }
}
```

`circle_engagement.status`:
- `"thriving"` — completion > 70%
- `"active"` — completion 40–70%
- `"at_risk"` — completion < 40%

`potential_mentors` — участники с proof_count ≥ 10 И helps_count (buddy reviews) ≥ 5.

---

### GET /v1/community-spaces/:id/best-proofs

Витрина лучших пруфов — публичный endpoint для лидера, чтобы отбирать контент для постов.

```
GET /v1/community-spaces/:id/best-proofs?limit=10
```

**Response 200:**
```json
{
  "data": [
    {
      "check_in_id": "uuid",
      "user_display_name": "Алина Петрова",
      "avatar_url": "...",
      "goal_title": "ML в продакшене",
      "proof_text": "Задеплоила первую модель...",
      "proof_url": "https://...",
      "submitted_at": "2026-05-06T14:00:00Z",
      "likes_count": 12,
      "is_featured": false
    }
  ]
}
```

### POST /v1/community-spaces/:id/best-proofs/:check_in_id/feature

Лидер помечает пруф как "витринный" — он появляется в витрине сообщества.

```
POST /v1/community-spaces/:id/best-proofs/:check_in_id/feature
```

**Response 200:** `{ "data": { "check_in_id": "uuid", "is_featured": true } }`

---

## Структура файлов

```
backend/internal/analytics/
  community.go      ← GetCommunityAnalytics(), GetBestProofs(), FeatureProof()
  http_handler.go   ← добавить endpoints
```

### SQL — circle engagement

```sql
SELECT 
    c.id as circle_id,
    c.name as circle_name,
    COUNT(DISTINCT cm.user_id) as members_count,
    COUNT(DISTINCT ci.id) as proofs_count,
    ROUND(
        COUNT(DISTINCT CASE WHEN ci.status = 'approved' THEN ci.user_id END)::numeric / 
        NULLIF(COUNT(DISTINCT cm.user_id), 0) * 100, 1
    ) as completion_pct
FROM circles c
JOIN circle_memberships cm ON cm.circle_id = c.id AND cm.status = 'active'
LEFT JOIN check_ins ci ON ci.user_id = cm.user_id 
    AND ci.created_at > NOW() - INTERVAL '30 days'
WHERE c.community_space_id = $1
GROUP BY c.id, c.name
ORDER BY proofs_count DESC
```

---

## Acceptance Criteria

- [ ] 403 для не-лидеров community
- [ ] `retention_pct` = active_members / total_members × 100 (округлено до 1 знака)
- [ ] `circle_engagement.status` корректно классифицирует by completion_pct
- [ ] `potential_mentors` — только участники с ≥10 пруфов И ≥5 ревью
- [ ] `/best-proofs` возвращает отсортированные по `likes_count DESC` потом `submitted_at DESC`
- [ ] `feature` действие: повторный вызов — идемпотентен (возвращает 200, не 409)
- [ ] Пустое сообщество (0 members) — корректные нули

---

## Что нельзя делать

- Не показывать личные данные участников без их согласия
- Не создавать публичный рейтинг участников "лучший/худший"
- Не делать `best-proofs` endpoint доступным без авторизации (только для лидера)
