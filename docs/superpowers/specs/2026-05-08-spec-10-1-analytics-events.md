# Спек 10.1 — Инструментация событий для пилота

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 10 · Аналитика пилота  
**Зависимости:** Спек 1.3 (contracts), 3.1 (metrics)  
**Сложность:** S  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Пилот с первыми командами. Нужно измерить 5 ключевых метрик: первый пруф, buddy match, дошье, completion сезона, weekly активность. Инструментация должна быть лёгкой и не мешать основному коду.

---

## Миграция

```sql
-- 00026_analytics_events_v2.sql
-- (если 00026 занята dossiers — использовать 00027)

ALTER TABLE analytics_events
  ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES workspaces(id),
  ADD COLUMN IF NOT EXISTS teamspace_id UUID REFERENCES teamspaces(id);

-- Новые event_types добавляются только в CHECK constraint или просто в документацию.
-- Не меняем структуру таблицы — только добавляем новые events через сервис.
```

Если таблицы `analytics_events` нет — создаём:

```sql
CREATE TABLE IF NOT EXISTS analytics_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    teamspace_id UUID REFERENCES teamspaces(id) ON DELETE SET NULL,
    event_type   TEXT NOT NULL,
    properties   JSONB DEFAULT '{}',
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_analytics_events_type_at ON analytics_events(event_type, created_at DESC);
CREATE INDEX idx_analytics_events_user_at ON analytics_events(user_id, created_at DESC);
CREATE INDEX idx_analytics_events_workspace ON analytics_events(workspace_id, event_type, created_at DESC);
```

---

## Events каталог

| event_type | Когда | Properties |
|------------|-------|------------|
| `proof_contract_created` | POST /v1/goals/:id/contracts → 201 | `goal_id`, `mode`, `due_in_days` |
| `first_proof_submitted` | При первом check_in пользователя (всего) | `goal_id`, `has_artifact` |
| `buddy_matched` | POST /v1/goals/:id/buddy-invite → принято | `goal_id`, `source` (ai/manual) |
| `dossier_generated` | POST /v1/ai/growth-dossier → 200 | `period_days`, `total_proofs_in_period` |
| `season_completed` | Когда goal переходит в status=completed | `goal_id`, `duration_days`, `proof_count` |
| `weekly_proof_submitted` | При каждом check_in | `goal_id`, `week_number`, `has_artifact`, `buddy_approved` |
| `anti_proof_submitted` | check_in с prefill_anti | `goal_id` |
| `ai_suggestion_used` | Клик "Использовать →" из GoalToProofs | `goal_id` |
| `workspace_created` | POST /v1/workspaces → 201 | `type` |
| `teamspace_created` | POST /v1/workspaces/:id/teamspaces → 201 | `workspace_id` |

---

## Backend: Analytics Service

```go
// backend/internal/analytics/analytics.go

type Event struct {
    UserID      *string
    WorkspaceID *string
    TeamspaceID *string
    EventType   string
    Properties  map[string]interface{}
}

type Service struct {
    db *sqlx.DB
}

func (s *Service) Track(ctx context.Context, ev Event) {
    // Fire-and-forget: не блокируем основной поток
    go func() {
        props, _ := json.Marshal(ev.Properties)
        _, err := s.db.ExecContext(ctx,
            `INSERT INTO analytics_events (user_id, workspace_id, teamspace_id, event_type, properties)
             VALUES ($1, $2, $3, $4, $5)`,
            ev.UserID, ev.WorkspaceID, ev.TeamspaceID, ev.EventType, props,
        )
        if err != nil {
            // log.Printf но не паникуем — аналитика не критична для бизнес-логики
            _ = err
        }
    }()
}
```

**Принцип:** `Track()` всегда вызывается через `go func()`. Ошибка записи в analytics никогда не роллбэкит основную операцию.

### Инициализация

```go
// В platform/app/api.go или infrastructure:
analyticsService := analytics.NewService(db)
```

Инжектируется в handlers через deps-структуру.

---

## Добавление трекинга в handlers

Примеры — не переписывать handlers, а добавить одну строку после успешной операции:

```go
// В check_ins handler, после создания check_in:
analyticsService.Track(ctx, analytics.Event{
    UserID:    &userID,
    EventType: "weekly_proof_submitted",
    Properties: map[string]interface{}{
        "goal_id":       checkIn.GoalID,
        "has_artifact":  checkIn.ArtifactURL != "",
        "buddy_approved": false, // обновляется при buddy_feedback
    },
})

// Первый пруф — специальная проверка:
count, _ := repo.CountUserCheckIns(ctx, userID)
if count == 1 {
    analyticsService.Track(ctx, analytics.Event{
        UserID:    &userID,
        EventType: "first_proof_submitted",
        Properties: map[string]interface{}{
            "goal_id":      checkIn.GoalID,
            "has_artifact": checkIn.ArtifactURL != "",
        },
    })
}
```

```go
// В contracts handler:
analyticsService.Track(ctx, analytics.Event{
    UserID:    &userID,
    EventType: "proof_contract_created",
    Properties: map[string]interface{}{
        "goal_id":     contract.GoalID,
        "mode":        contract.Mode,
        "due_in_days": int(contract.DueAt.Sub(time.Now()).Hours() / 24),
    },
})
```

```go
// В ai handler, после генерации дошье:
analyticsService.Track(ctx, analytics.Event{
    UserID:    &userID,
    EventType: "dossier_generated",
    Properties: map[string]interface{}{
        "period_days":              int(req.PeriodTo.Sub(req.PeriodFrom).Hours() / 24),
        "total_proofs_in_period":   dc.Stats.TotalProofs,
    },
})
```

---

## Frontend: client-side события

Некоторые события удобнее трекать на клиенте (клик "Использовать →" из AI suggestions).

```tsx
// lib/analytics.ts
export async function trackEvent(eventType: string, properties: Record<string, unknown> = {}) {
  try {
    await fetch('/api/v1/analytics/track', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ event_type: eventType, properties }),
    });
  } catch {
    // Тихий fail — аналитика не должна ломать UI
  }
}
```

```tsx
// В SuggestionCard:
<Link
  href={`...`}
  onClick={() => trackEvent('ai_suggestion_used', { goal_id: goalId })}
>
  Использовать →
</Link>
```

### Backend endpoint для client-side трекинга

```go
// POST /v1/analytics/track
type ClientTrackRequest struct {
    EventType  string                 `json:"event_type"`
    Properties map[string]interface{} `json:"properties"`
}

// Whitelist разрешённых event_types от клиента (не позволяем трекать произвольные):
var allowedClientEvents = map[string]bool{
    "ai_suggestion_used": true,
    "anti_proof_started": true,
    "buddy_match_opened": true,
}

func (h *Handler) TrackClientEvent(w http.ResponseWriter, r *http.Request) {
    var req ClientTrackRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    if !allowedClientEvents[req.EventType] {
        http.Error(w, "unknown event", 400)
        return
    }
    
    userID := userIDFromCtx(r.Context())
    h.analytics.Track(r.Context(), analytics.Event{
        UserID:    &userID,
        EventType: req.EventType,
        Properties: req.Properties,
    })
    
    w.WriteHeader(204)
}
```

---

## Acceptance Criteria

- [ ] Таблица `analytics_events` существует с индексами
- [ ] `first_proof_submitted` срабатывает ровно один раз на пользователя
- [ ] `proof_contract_created` срабатывает при каждом создании контракта
- [ ] `dossier_generated` содержит `period_days` и `total_proofs_in_period`
- [ ] `analyticsService.Track()` — fire-and-forget, ошибка не роллбэкит бизнес-операцию
- [ ] Client-side whitelist: неизвестный event_type → 400 (не 500)
- [ ] `trackEvent()` в lib/analytics.ts тихо игнорирует сетевые ошибки (try/catch)
- [ ] События содержат `workspace_id` где применимо (для пилотного отчёта по workspace)

---

## Что нельзя делать

- Не блокировать основную операцию из-за ошибки аналитики
- Не логировать PII (имена, email) в properties
- Не создавать client-side трекинг без whitelist на backend
- Не добавлять analytics в тесты (мокировать интерфейс)
