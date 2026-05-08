# Спек 3.2 — Метрики Buddy

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 3 · Backend метрики  
**Зависимости:** Существующие `pacts`, `check_ins`, `check_in_reviews`  
**Сложность:** S  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Buddy — не контролёр, а помощник. Его дашборд должен показывать **где нужна помощь**, а не "кто плохо работает".

Buddy видит:
- Кто ждёт ревью прямо сейчас
- Кто застрял (нет пруфа давно)
- Его собственная эффективность (среднее время ответа, сколько поддержал)

**Принцип:** buddy dashboard — это инструмент помощи, не надзора.

---

## API

### GET /v1/buddy/queue

Список пруфов ожидающих ревью, упорядочен по времени ожидания (самые долгие — первые).

```
GET /v1/buddy/queue
Authorization: Bearer {token}
```

**Response 200:**
```json
{
  "data": [
    {
      "check_in_id": "uuid",
      "goal_id": "uuid",
      "goal_title": "Kotlin Coroutines",
      "user": {
        "user_id": "uuid",
        "display_name": "Мария Иванова",
        "avatar_url": "..."
      },
      "submitted_at": "2026-05-07T14:30:00Z",
      "waiting_hours": 20,
      "proof_type": "artifact",
      "preview": "Показала пример SupervisorJob — код приложила...",
      "status": "submitted"
    }
  ],
  "meta": { "total": 3 }
}
```

`preview` — первые 120 символов текста пруфа (если есть).

---

### GET /v1/buddy/needs-attention

Подопечные которые давно не сдавали пруф (для buddy, не для лидера).

```
GET /v1/buddy/needs-attention
```

**Response 200:**
```json
{
  "data": [
    {
      "user_id": "uuid",
      "display_name": "Алексей Соколов",
      "avatar_url": "...",
      "goal_title": "System Design",
      "days_since_last_proof": 8,
      "last_check_in_at": "2026-04-29T10:00:00Z",
      "has_broken_contract": true
    }
  ]
}
```

Порог для включения: ≥ 5 дней без approved check-in при режиме `regular_rhythm`.

---

### GET /v1/buddy/stats

Статистика эффективности buddy.

```
GET /v1/buddy/stats
```

**Response 200:**
```json
{
  "data": {
    "active_buddies_count": 3,
    "total_reviews_given": 28,
    "avg_response_hours": 6.4,
    "helpful_reviews_count": 24,
    "people_supported_count": 3,
    "reviews_this_week": 4,
    "reviews_last_week": 3
  }
}
```

`helpful_reviews_count` — количество ревью где пользователь поставил "👍 полезно" (будущая фича; сейчас = total_reviews_given × 0.85 как placeholder).

`avg_response_hours` — среднее время от `check_in.created_at` до `check_in_review.created_at` за последние 30 дней.

---

## Go: структура сервиса

```
backend/internal/buddy/
  domain.go      ← BuddyQueueItem, NeedsAttentionItem, BuddyStats типы
  queries.go     ← SQL запросы
  service.go     ← GetQueue(), GetNeedsAttention(), GetStats()
  http_handler.go ← 3 endpoints выше
```

### Ключевые SQL запросы

```go
const buddyQueueSQL = `
    SELECT
        ci.id as check_in_id,
        g.id as goal_id,
        g.title as goal_title,
        u.id as user_id,
        u.display_name,
        u.avatar_url,
        ci.created_at as submitted_at,
        EXTRACT(EPOCH FROM (NOW() - ci.created_at))/3600 as waiting_hours,
        LEFT(ci.content, 120) as preview
    FROM check_ins ci
    JOIN goals g ON g.id = ci.goal_id
    JOIN pacts p ON p.goal_id = g.id AND p.buddy_id = $1 AND p.status = 'active'
    JOIN users u ON u.id = g.user_id
    WHERE ci.status = 'submitted'
    ORDER BY ci.created_at ASC
`

const buddyNeedsAttentionSQL = `
    SELECT
        u.id as user_id,
        u.display_name,
        u.avatar_url,
        g.title as goal_title,
        EXTRACT(DAY FROM (NOW() - MAX(ci.created_at))) as days_since_last_proof,
        MAX(ci.created_at) as last_check_in_at,
        EXISTS(
            SELECT 1 FROM proof_contracts pc
            WHERE pc.goal_id = g.id AND pc.status = 'broken'
        ) as has_broken_contract
    FROM pacts p
    JOIN goals g ON g.id = p.goal_id
    JOIN users u ON u.id = g.user_id
    LEFT JOIN check_ins ci ON ci.goal_id = g.id AND ci.status = 'approved'
    WHERE p.buddy_id = $1
      AND p.status = 'active'
      AND g.movement_mode = 'regular_rhythm'
    GROUP BY u.id, u.display_name, u.avatar_url, g.id, g.title
    HAVING MAX(ci.created_at) < NOW() - INTERVAL '5 days'
        OR MAX(ci.created_at) IS NULL
`
```

---

## Acceptance Criteria

- [ ] `GET /v1/buddy/queue` возвращает только check-ins с `status = 'submitted'` где текущий пользователь является buddy
- [ ] Очередь упорядочена по времени ожидания (самые долгие первые)
- [ ] `GET /v1/buddy/needs-attention` показывает подопечных с ≥5 дней без пруфа
- [ ] `GET /v1/buddy/stats` возвращает `avg_response_hours` корректно (null если нет ревью)
- [ ] Все 3 endpoint возвращают пустые данные (не ошибку) если buddy не имеет подопечных
- [ ] 401 без токена

---

## Что нельзя делать

- Не показывать личные блокеры подопечных (только факт отсутствия пруфа)
- Не давать buddy доступ к описаниям целей подопечных
- Не включать observer/stranger в buddy queue
