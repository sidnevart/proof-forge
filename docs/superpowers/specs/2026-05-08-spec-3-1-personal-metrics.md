# Спек 3.1 — Персональные метрики и умная карточка «Что сейчас»

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 3 · Backend метрики  
**Зависимости:** Спек 1.3 (proof_contracts), существующие goals + check_ins  
**Сложность:** M  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Умная карточка "Что сейчас" — ключевой UX-паттерн продукта (DNA Раздел 4). Она требует backend API который вычисляет текущее состояние пользователя и возвращает одно приоритетное действие.

Параллельно нужен endpoint со всеми персональными метриками для личного дашборда: streak, proofsThisWeek, динамика по сравнению с прошлой неделей.

---

## Что реализовать

### Backend
1. Пакет `backend/internal/stats/` — UserStats сервис
2. `GET /v1/me/now` — умная карточка (одно действие)
3. `GET /v1/me/stats` — все персональные метрики

---

## Метрики и их определения

| Метрика | Формула | Как часто |
|---------|---------|-----------|
| `proof_streak` | Количество последовательных недель с ≥1 подтверждённым check-in | Пересчёт при каждом check-in |
| `proofs_this_week` | Кол-во check-ins со статусом approved за текущую ISO-неделю | Real-time |
| `proofs_last_week` | То же за прошлую ISO-неделю | Пересчёт в конце недели |
| `week_vs_last_week` | proofs_this_week - proofs_last_week | Вычисляется |
| `active_weeks` | Кол-во недель с ≥1 approved check-in за последние 12 недель | Пересчёт weekly |
| `personal_record_week` | Максимальное кол-во approved check-ins за одну неделю (за всё время) | Пересчёт при каждом check-in |
| `season_completion_pct` | Для challenge mode: выполнено proof contracts / всего контрактов × 100 | Real-time |
| `next_proof_expected_at` | Из proof contract или rhythm_cadence цели | Вычисляется |

---

## API: GET /v1/me/stats

```
GET /v1/me/stats
Authorization: Bearer {token}
```

**Response 200:**
```json
{
  "data": {
    "proof_streak": 4,
    "proofs_this_week": 2,
    "proofs_last_week": 1,
    "week_vs_last_week": 1,
    "week_trend": "better",
    "active_weeks": 9,
    "personal_record_week": 3,
    "season_completion_pct": 75,
    "next_proof_expected_at": "2026-05-12T00:00:00Z",
    "active_goals_count": 2,
    "pending_contracts_count": 1
  }
}
```

`week_trend`: `"better"` | `"same"` | `"worse"` | `"first_week"`

---

## API: GET /v1/me/now

Реализует алгоритм умной карточки из DNA Раздел 4:

```
GET /v1/me/now
Authorization: Bearer {token}
```

**Алгоритм приоритетов (выполняются SQL-запросами по порядку, возвращается первый hit):**

```go
func (s *Service) GetNowCard(ctx context.Context, userID uuid.UUID) (*NowCard, error) {
    // 1. Buddy ждёт ответа > 24 часов
    if pending := s.db.PendingBuddyReview(ctx, userID, 24*time.Hour); pending != nil {
        return &NowCard{
            Type:    "buddy_waiting",
            Title:   "Бадди ждёт твоего ответа",
            Subtitle: fmt.Sprintf("%s оставил комментарий %s", pending.BuddyName, humanizeAge(pending.WaitingSince)),
            Action:  NowAction{Label: "Ответить бадди", URL: "/goals/" + pending.GoalID + "/checkins/" + pending.CheckInID},
            Urgency: "warn",
        }, nil
    }

    // 2. Просроченный proof contract
    if broken := s.db.BrokenContract(ctx, userID); broken != nil {
        return &NowCard{
            Type:    "contract_broken",
            Title:   "Пруф просрочен",
            Subtitle: fmt.Sprintf("По цели «%s» — %s", broken.GoalTitle, broken.WhatToProve),
            Action:  NowAction{Label: "Сдать пруф или обновить контракт", URL: "/goals/" + broken.GoalID},
            Urgency: "danger",
        }, nil
    }

    // 3. Дедлайн сегодня
    if today := s.db.ContractDueToday(ctx, userID); today != nil {
        return &NowCard{
            Type:    "due_today",
            Title:   "Дедлайн сегодня",
            Subtitle: fmt.Sprintf("«%s» — %s", today.GoalTitle, today.WhatToProve),
            Action:  NowAction{Label: "Сдать пруф", URL: "/goals/" + today.GoalID + "/check-in"},
            Urgency: "fire",
        }, nil
    }

    // 4. Дедлайн завтра
    if tomorrow := s.db.ContractDueTomorrow(ctx, userID); tomorrow != nil {
        return &NowCard{
            Type:    "due_tomorrow",
            Title:   "Дедлайн завтра",
            Subtitle: fmt.Sprintf("«%s» — %s", tomorrow.GoalTitle, tomorrow.WhatToProve),
            Action:  NowAction{Label: "Начать сейчас", URL: "/goals/" + tomorrow.GoalID + "/check-in"},
            Urgency: "fire",
        }, nil
    }

    // 5. Нет пруфа 5+ дней (regular rhythm)
    if overdue := s.db.InactiveRegularGoal(ctx, userID, 5); overdue != nil {
        return &NowCard{
            Type:    "long_pause",
            Title:   fmt.Sprintf("Ты не сдавал пруф %d дней", overdue.DaysSinceLastProof),
            Subtitle: fmt.Sprintf("По цели «%s»", overdue.GoalTitle),
            Action:  NowAction{Label: "Сдать пруф или сказать где застрял", URL: "/goals/" + overdue.GoalID + "/check-in"},
            Urgency: "warn",
        }, nil
    }

    // 6. Нет пруфа 3+ дня
    if paused := s.db.InactiveRegularGoal(ctx, userID, 3); paused != nil {
        return &NowCard{
            Type:    "small_pause",
            Title:   "Небольшой перерыв",
            Subtitle: fmt.Sprintf("Готов к следующему шагу по «%s»?", paused.GoalTitle),
            Action:  NowAction{Label: "Продолжить", URL: "/goals/" + paused.GoalID + "/check-in"},
            Urgency: "neutral",
        }, nil
    }

    // 7. Есть новый milestone для отмечания
    if milestone := s.db.UnacknowledgedMilestone(ctx, userID); milestone != nil {
        return &NowCard{
            Type:    "milestone",
            Title:   milestone.Title,
            Subtitle: "Ты добился этого!",
            Action:  NowAction{Label: "Посмотреть достижение", URL: "/me/achievements"},
            Urgency: "win",
        }, nil
    }

    // 8. Нет активных целей
    if !s.db.HasActiveGoals(ctx, userID) {
        return &NowCard{
            Type:    "no_goals",
            Title:   "Начни движение",
            Subtitle: "Создай первую цель или подключись к инициативе",
            Action:  NowAction{Label: "Создать цель", URL: "/goals/new"},
            Urgency: "neutral",
        }, nil
    }

    // 9. Всё хорошо
    nextProof := s.db.NextExpectedProof(ctx, userID)
    return &NowCard{
        Type:    "on_track",
        Title:   "Всё идёт по плану",
        Subtitle: formatNextProof(nextProof),
        Action:  nil, // нет срочного действия
        Urgency: "neutral",
    }, nil
}
```

**Response 200 (пример "buddy ждёт"):**
```json
{
  "data": {
    "type": "buddy_waiting",
    "title": "Бадди ждёт твоего ответа",
    "subtitle": "Артём оставил комментарий вчера",
    "urgency": "warn",
    "action": {
      "label": "Ответить бадди",
      "url": "/goals/uuid/checkins/uuid"
    }
  }
}
```

**Response 200 (пример "всё хорошо"):**
```json
{
  "data": {
    "type": "on_track",
    "title": "Всё идёт по плану",
    "subtitle": "Следующий пруф ожидается 12 мая",
    "urgency": "neutral",
    "action": null
  }
}
```

Urgency values: `"danger"` | `"fire"` | `"warn"` | `"win"` | `"neutral"`

---

## Структура файлов

```
backend/internal/stats/
  domain.go          ← UserStats, NowCard, NowAction типы
  queries.go         ← SQL запросы (методы для db.PendingBuddyReview и др.)
  service.go         ← GetStats(), GetNowCard()
  http_handler.go    ← GET /v1/me/stats и GET /v1/me/now
```

### SQL запросы (queries.go)

```go
// PendingBuddyReview — check-in где buddy оставил review > N часов назад
// и пользователь ещё не ответил (нет follow-up check-in)
const pendingBuddyReviewSQL = `
    SELECT cr.id, cr.check_in_id, u.display_name as buddy_name, cr.created_at as waiting_since,
           g.id as goal_id
    FROM check_in_reviews cr
    JOIN check_ins ci ON ci.id = cr.check_in_id
    JOIN goals g ON g.id = ci.goal_id
    JOIN users u ON u.id = cr.reviewer_id
    WHERE g.user_id = $1
      AND cr.created_at < NOW() - INTERVAL '1 hour' * $2
      AND ci.status = 'changes_requested'
    ORDER BY cr.created_at ASC
    LIMIT 1
`

// BrokenContract — просроченные контракты
const brokenContractSQL = `
    SELECT pc.id, pc.what_to_prove, g.title as goal_title, g.id as goal_id
    FROM proof_contracts pc
    JOIN goals g ON g.id = pc.goal_id
    WHERE pc.user_id = $1
      AND pc.status = 'broken'
    ORDER BY pc.due_at ASC
    LIMIT 1
`
```

---

## Acceptance Criteria

- [ ] `GET /v1/me/stats` возвращает все 8 метрик корректно
- [ ] `proof_streak` = 0 если в текущей неделе нет approved check-in
- [ ] `week_trend` = "first_week" для пользователей без прошлой недели
- [ ] `GET /v1/me/now` возвращает карточку типа `buddy_waiting` если есть pending review > 24ч
- [ ] Приоритет соблюдается: broken contract > due today > due tomorrow > pause > milestone > no goals > on_track
- [ ] `urgency` цветовые значения соответствуют DNA токенам: danger→--danger, fire→--fire, warn→--warn, win→--win
- [ ] Оба endpoint возвращают 401 без токена
- [ ] Unit-тесты для алгоритма `GetNowCard` с мок-данными для каждого из 9 состояний

---

## Что нельзя делать

- Не кешировать результат `GET /v1/me/now` — данные должны быть real-time
- Не возвращать более одного действия из `now` (принцип П3 из DNA)
- Не показывать имена других пользователей кроме buddy (приватность)
