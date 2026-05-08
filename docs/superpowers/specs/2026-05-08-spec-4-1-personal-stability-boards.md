# Спек 4.1 — Личный лидерборд + Борд стабильности

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 4 · Лидерборды Backend  
**Зависимости:** Спек 3.1 (stats), существующие check_ins  
**Сложность:** S  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

**Личный лидерборд** — самый безопасный тип: пользователь видит только себя. Сравнение с собой вчерашним, а не с другими. Мотивирует не слиться без публичного унижения.

**Борд стабильности** — показывает кто стабильно сдаёт пруфы. Соревнуются в регулярности, не в "кто умнее". Показывает топ, никогда не показывает дно.

---

## API

### GET /v1/me/leaderboard

Личный борд — только данные текущего пользователя.

```
GET /v1/me/leaderboard
```

**Response 200:**
```json
{
  "data": {
    "current_week": {
      "proofs_count": 2,
      "vs_last_week": "+1",
      "trend": "better"
    },
    "current_season": {
      "proofs_count": 14,
      "vs_last_season": "+3",
      "trend": "better"
    },
    "streak": {
      "current_weeks": 4,
      "personal_record_weeks": 7,
      "is_personal_record": false
    },
    "weekly_history": [
      { "week": "2026-W16", "proofs_count": 1 },
      { "week": "2026-W17", "proofs_count": 3 },
      { "week": "2026-W18", "proofs_count": 2 },
      { "week": "2026-W19", "proofs_count": 2 },
      { "week": "2026-W20", "proofs_count": 2 }
    ]
  }
}
```

`weekly_history` — последние 8 недель для мини-спаркчарта в UI.

---

### GET /v1/circles/:id/stability-board

Борд стабильности — кто стабильнее всего сдаёт пруфы в круге. **Только топ**; аутсайдеры не показываются.

```
GET /v1/circles/:id/stability-board?limit=5
Требует: участник круга
```

**Response 200:**
```json
{
  "data": {
    "circle_id": "uuid",
    "circle_name": "Группа А",
    "board_type": "stability",
    "period": "last_4_weeks",
    "entries": [
      {
        "rank": 1,
        "user_id": "uuid",
        "display_name": "Артём Козлов",
        "avatar_url": "...",
        "proof_streak_weeks": 8,
        "active_weeks": 4,
        "consistency_score": 95,
        "is_current_user": false
      },
      {
        "rank": 2,
        "user_id": "uuid",
        "display_name": "Мария Иванова",
        "avatar_url": "...",
        "proof_streak_weeks": 6,
        "active_weeks": 4,
        "consistency_score": 88,
        "is_current_user": true
      }
    ],
    "current_user_position": {
      "rank": 2,
      "consistency_score": 88,
      "total_participants": 7
    }
  }
}
```

**Ключевые правила:**
- `limit` по умолчанию 5, максимум 10 — только топ
- Если текущий пользователь не в топе — добавляется `current_user_position` с его позицией, но список всё равно обрезан по limit
- Никогда не показывать участников ниже limit в основном списке

**consistency_score формула:**
```
weeks_with_proof_in_period / total_weeks_in_period * 100
```

---

## Структура файлов

```
backend/internal/leaderboards/
  personal.go     ← GetPersonalLeaderboard()
  stability.go    ← GetStabilityBoard()
  http_handler.go ← 2 endpoints
```

### SQL — stability board

```sql
SELECT
    u.id as user_id,
    u.display_name,
    u.avatar_url,
    COUNT(DISTINCT DATE_TRUNC('week', ci.created_at)) as active_weeks,
    ROUND(
        COUNT(DISTINCT DATE_TRUNC('week', ci.created_at))::numeric /
        $3 * 100
    ) as consistency_score
FROM circle_memberships cm
JOIN users u ON u.id = cm.user_id
LEFT JOIN check_ins ci ON ci.user_id = cm.user_id
    AND ci.created_at > NOW() - INTERVAL '1 week' * $3
    AND ci.status = 'approved'
WHERE cm.circle_id = $1
  AND cm.status = 'active'
GROUP BY u.id, u.display_name, u.avatar_url
ORDER BY consistency_score DESC, active_weeks DESC
LIMIT $2
```

---

## Acceptance Criteria

- [ ] `GET /v1/me/leaderboard` доступен только авторизованному пользователю, данные только его
- [ ] `weekly_history` содержит последние 8 ISO-недель (пустые недели = 0, не пропускаются)
- [ ] `trend` = `"first_week"` если нет данных за прошлую неделю
- [ ] stability board: только участники данного круга
- [ ] stability board: `is_current_user: true` для текущего пользователя в списке
- [ ] stability board: `current_user_position` заполнен даже если пользователь не в топ-5
- [ ] stability board: не показывает участников ниже `limit`
- [ ] 403 для не-участников круга на stability board

---

## Что нельзя делать

- Не показывать в stability board участников с rank > limit (нет публичного дна)
- Не добавлять в личный борд сравнение с другими пользователями
- Не показывать личный борд другим пользователям (только owner)
