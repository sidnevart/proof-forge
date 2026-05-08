# Спек 4.3 — Борд кругов + Борд роста

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 4 · Лидерборды Backend  
**Зависимости:** Спек 1.2 (circles расширены), check_ins  
**Сложность:** S  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

**Борд кругов** — сравниваем команды (круги), а не отдельных людей. Человеку не так больно быть "последним" если это его круг отстаёт — легче работать вместе. Создаёт командную динамику.

**Борд роста** — топ тех кто больше всего вырос относительно себя прошлой недели. Помогает новичкам попасть в позитивную видимость: было 0 пруфов → стало 2 — это заслуживает признания.

---

## API

### GET /v1/community-spaces/:id/circles-board

Сравнение кругов внутри community space.

```
GET /v1/community-spaces/:id/circles-board?period=last_4_weeks
Требует: участник community space
```

**Response 200:**
```json
{
  "data": {
    "board_type": "circles",
    "period": "last_4_weeks",
    "entries": [
      {
        "rank": 1,
        "circle_id": "uuid",
        "circle_name": "Группа А",
        "members_count": 7,
        "proofs_submitted": 34,
        "proofs_approved": 30,
        "completion_pct": 85.7,
        "active_members": 7,
        "circle_score": 91,
        "is_my_circle": false
      },
      {
        "rank": 2,
        "circle_id": "uuid",
        "circle_name": "Группа Б",
        "members_count": 6,
        "proofs_submitted": 18,
        "proofs_approved": 14,
        "completion_pct": 60.0,
        "active_members": 5,
        "circle_score": 72,
        "is_my_circle": true
      }
    ],
    "my_circle_position": {
      "rank": 2,
      "circle_score": 72,
      "total_circles": 5
    }
  }
}
```

**circle_score формула:**
```
(completion_pct * 0.5) + (active_members / members_count * 100 * 0.3) + 
(proofs_approved / max(proofs_submitted, 1) * 100 * 0.2)
```

Борд кругов **показывает все круги** (не обрезается), т.к. соревнование командное и знание своего места — нормально. Люди не унижаются — круги соревнуются.

---

### GET /v1/circles/:id/growth-board

Топ по росту относительно прошлой недели.

```
GET /v1/circles/:id/growth-board?limit=5
Требует: участник круга
```

**Response 200:**
```json
{
  "data": {
    "board_type": "growth",
    "entries": [
      {
        "rank": 1,
        "user_id": "uuid",
        "display_name": "Новичок Антон",
        "avatar_url": "...",
        "proofs_this_week": 3,
        "proofs_last_week": 0,
        "growth_delta": 3,
        "growth_label": "С нуля до 3 пруфов",
        "is_current_user": false
      },
      {
        "rank": 2,
        "user_id": "uuid",
        "display_name": "Мария Иванова",
        "avatar_url": "...",
        "proofs_this_week": 2,
        "proofs_last_week": 0,
        "growth_delta": 2,
        "growth_label": "Первые пруфы за 2 недели",
        "is_current_user": true
      }
    ],
    "current_user_position": {
      "rank": 2,
      "growth_delta": 2,
      "total_participants": 7
    }
  }
}
```

**growth_delta** = proofs_this_week - proofs_last_week. Только положительный delta попадает в борд.

**growth_label** — генерируется на бэкенде:
```go
func growthLabel(delta, lastWeek int) string {
    if lastWeek == 0 && delta > 0 {
        return fmt.Sprintf("С нуля до %d пруфов", delta)
    }
    if delta >= lastWeek {
        return "Удвоил активность"
    }
    return fmt.Sprintf("+%d к прошлой неделе", delta)
}
```

---

## Структура файлов

```
backend/internal/leaderboards/
  circles.go   ← GetCirclesBoard()
  growth.go    ← GetGrowthBoard()
  http_handler.go ← добавить endpoints
```

### SQL — growth board

```sql
WITH this_week AS (
    SELECT user_id, COUNT(*) as proofs_count
    FROM check_ins
    WHERE created_at >= DATE_TRUNC('week', NOW())
      AND status = 'approved'
    GROUP BY user_id
),
last_week AS (
    SELECT user_id, COUNT(*) as proofs_count
    FROM check_ins
    WHERE created_at >= DATE_TRUNC('week', NOW()) - INTERVAL '1 week'
      AND created_at < DATE_TRUNC('week', NOW())
      AND status = 'approved'
    GROUP BY user_id
)
SELECT
    u.id as user_id,
    u.display_name,
    u.avatar_url,
    COALESCE(tw.proofs_count, 0) as proofs_this_week,
    COALESCE(lw.proofs_count, 0) as proofs_last_week,
    COALESCE(tw.proofs_count, 0) - COALESCE(lw.proofs_count, 0) as growth_delta
FROM circle_memberships cm
JOIN users u ON u.id = cm.user_id
LEFT JOIN this_week tw ON tw.user_id = cm.user_id
LEFT JOIN last_week lw ON lw.user_id = cm.user_id
WHERE cm.circle_id = $1 AND cm.status = 'active'
  AND COALESCE(tw.proofs_count, 0) > COALESCE(lw.proofs_count, 0)
ORDER BY growth_delta DESC
LIMIT $2
```

---

## Acceptance Criteria

- [ ] circles board: содержит все круги community space (не обрезается по limit)
- [ ] `is_my_circle: true` для круга текущего пользователя
- [ ] `my_circle_position` заполнен всегда
- [ ] growth board: только участники с положительным delta
- [ ] growth board: top-N, `current_user_position` заполнен даже если вне топа
- [ ] growth_label генерируется корректно для 3 случаев
- [ ] 403 для не-участников

---

## Что нельзя делать

- Не показывать индивидуальные данные участников в circles board (только агрегат по кругу)
- Не включать участников с нулевым growth_delta в growth board
