# Спек 1.3 — Proof Contract

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 1 · Данные  
**Зависимости:** Существующие таблицы `goals`, `pacts`, `users`  
**Сложность:** M  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

**Proof Contract** — главная новая сущность v2. Это конкретное обещание: "К пятнице я покажу демо SupervisorJob и объясню где это применимо в нашем сервисе."

Proof contract отвечает на 5 вопросов:
1. Что я хочу доказать?
2. Чем я это докажу?
3. Когда я это покажу?
4. Кто подтвердит?
5. Что будет считаться достаточным пруфом?

**Отличие от существующего check-in:** check-in — это сданный факт. Proof contract — это обещание до сдачи. Контракт предшествует check-in'у.

**Статусная машина:**
```
pending → active → fulfilled
                 ↘ broken (просрочен и не выполнен)
         ↘ cancelled (пользователь отменил)
```

**Кто создаёт:** сам пользователь, под своей целью.  
**Кто подтверждает (buddy_seal):** buddy или teamspace lead/trusted_approver.

---

## Что реализовать

### Backend
1. SQL миграция `00020_proof_contracts.sql`
2. Пакет `backend/internal/contracts/` — domain, repository, service, handler
3. Фоновый воркер: cron раз в час помечает просроченные `active` контракты как `broken`
4. Роуты в `api.go`

---

## Схема базы данных

```sql
-- +goose Up

CREATE TABLE proof_contracts (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  goal_id           UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  user_id           UUID NOT NULL REFERENCES users(id),
  buddy_id          UUID REFERENCES users(id),          -- кто подтверждает (nullable = без buddy)

  -- Содержание контракта
  what_to_prove     TEXT NOT NULL,                      -- "покажу демо SupervisorJob"
  how_to_prove      TEXT NOT NULL,                      -- "код + короткое объяснение"
  proof_type        TEXT NOT NULL DEFAULT 'artifact'
                    CHECK (proof_type IN (
                      'artifact',    -- MR, файл, документ
                      'note',        -- текстовый отчёт
                      'demo',        -- демо, видео
                      'experiment',  -- результат исследования
                      'reflection'   -- разбор, конспект
                    )),

  -- Сроки
  due_at            TIMESTAMPTZ NOT NULL,
  reminded_at       TIMESTAMPTZ,                        -- когда последний раз напомнили

  -- Статус
  status            TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'active', 'fulfilled', 'broken', 'cancelled')),

  -- Связь с выполнением
  fulfilled_check_in_id UUID REFERENCES check_ins(id), -- какой check-in выполнил контракт

  -- Мета
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX proof_contracts_goal_idx ON proof_contracts(goal_id);
CREATE INDEX proof_contracts_user_idx ON proof_contracts(user_id);
CREATE INDEX proof_contracts_buddy_idx ON proof_contracts(buddy_id) WHERE buddy_id IS NOT NULL;
CREATE INDEX proof_contracts_status_idx ON proof_contracts(status);
CREATE INDEX proof_contracts_due_idx ON proof_contracts(due_at) WHERE status = 'active';

-- +goose Down
DROP TABLE IF EXISTS proof_contracts;
```

---

## Go Domain (`backend/internal/contracts/domain.go`)

```go
package contracts

import (
    "time"
    "github.com/google/uuid"
)

type ContractStatus string

const (
    ContractStatusPending   ContractStatus = "pending"
    ContractStatusActive    ContractStatus = "active"
    ContractStatusFulfilled ContractStatus = "fulfilled"
    ContractStatusBroken    ContractStatus = "broken"
    ContractStatusCancelled ContractStatus = "cancelled"
)

type ProofType string

const (
    ProofTypeArtifact   ProofType = "artifact"
    ProofTypeNote       ProofType = "note"
    ProofTypeDemo       ProofType = "demo"
    ProofTypeExperiment ProofType = "experiment"
    ProofTypeReflection ProofType = "reflection"
)

type ProofContract struct {
    ID                  uuid.UUID
    GoalID              uuid.UUID
    UserID              uuid.UUID
    BuddyID             *uuid.UUID
    WhatToProve         string
    HowToProve          string
    ProofType           ProofType
    DueAt               time.Time
    Status              ContractStatus
    FulfilledCheckInID  *uuid.UUID
    CreatedAt           time.Time
    UpdatedAt           time.Time
}

func (c *ProofContract) IsOverdue() bool {
    return c.Status == ContractStatusActive && time.Now().After(c.DueAt)
}

func (c *ProofContract) DaysUntilDue() int {
    return int(time.Until(c.DueAt).Hours() / 24)
}

// Используется умной карточкой "Что сейчас"
type ContractUrgency string

const (
    UrgencyOverdue   ContractUrgency = "overdue"
    UrgencyToday     ContractUrgency = "today"
    UrgencyTomorrow  ContractUrgency = "tomorrow"
    UrgencyThisWeek  ContractUrgency = "this_week"
    UrgencyLater     ContractUrgency = "later"
)

func (c *ProofContract) Urgency() ContractUrgency {
    days := c.DaysUntilDue()
    switch {
    case days < 0:
        return UrgencyOverdue
    case days == 0:
        return UrgencyToday
    case days == 1:
        return UrgencyTomorrow
    case days <= 7:
        return UrgencyThisWeek
    default:
        return UrgencyLater
    }
}
```

---

## API Contract

```
POST   /v1/goals/:goal_id/contracts    — создать контракт под цель
GET    /v1/goals/:goal_id/contracts    — контракты по цели
GET    /v1/contracts                   — все мои активные контракты
GET    /v1/contracts/:id               — детали контракта
PATCH  /v1/contracts/:id              — обновить (только пока status=pending)
POST   /v1/contracts/:id/activate     — перевести в active (начать)
POST   /v1/contracts/:id/cancel       — отменить
POST   /v1/contracts/:id/fulfill      — отметить как выполненный (со ссылкой на check_in)
```

**POST /v1/goals/:goal_id/contracts — Request:**
```json
{
  "what_to_prove": "Покажу мини-пример с SupervisorJob",
  "how_to_prove": "Код + короткое объяснение где применимо",
  "proof_type": "artifact",
  "due_at": "2026-05-09T18:00:00Z",
  "buddy_id": "uuid-or-null"
}
```

**Response 201:**
```json
{
  "data": {
    "id": "uuid",
    "goal_id": "uuid",
    "what_to_prove": "Покажу мини-пример с SupervisorJob",
    "how_to_prove": "Код + короткое объяснение где применимо",
    "proof_type": "artifact",
    "due_at": "2026-05-09T18:00:00Z",
    "status": "pending",
    "urgency": "tomorrow",
    "buddy": {
      "user_id": "uuid",
      "display_name": "Артём Козлов"
    }
  }
}
```

**GET /v1/contracts — Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "goal_title": "Kotlin Coroutines",
      "what_to_prove": "Покажу мини-пример с SupervisorJob",
      "due_at": "2026-05-09T18:00:00Z",
      "status": "active",
      "urgency": "tomorrow",
      "days_until_due": 1
    }
  ]
}
```

**POST /v1/contracts/:id/fulfill — Request:**
```json
{ "check_in_id": "uuid" }
```

---

## Фоновый воркер (broken contracts)

В `backend/cmd/worker/` добавить задачу:

```go
// Запускается каждый час
func markBrokenContracts(ctx context.Context, db *pgxpool.Pool) error {
    _, err := db.Exec(ctx, `
        UPDATE proof_contracts
        SET status = 'broken', updated_at = NOW()
        WHERE status = 'active'
          AND due_at < NOW()
          AND fulfilled_check_in_id IS NULL
    `)
    return err
}
```

---

## Интеграция с умной карточкой "Что сейчас"

Endpoint `GET /v1/me/now` (создаётся в спеке 3.1) использует контракты для определения срочности.

Порядок приоритетов:
1. status = 'broken' → "Пруф просрочен — сдай сейчас или обнови контракт"
2. urgency = 'today' → "Дедлайн сегодня"
3. urgency = 'tomorrow' → "Дедлайн завтра"

---

## Ошибки и валидация

| Ситуация | HTTP | Код |
|----------|------|-----|
| due_at в прошлом | 422 | `due_at_in_past` |
| goal не принадлежит пользователю | 403 | `forbidden` |
| контракт уже fulfilled/broken | 409 | `contract_not_modifiable` |
| fulfill без существующего check_in | 422 | `check_in_not_found` |
| check_in принадлежит другой цели | 422 | `check_in_goal_mismatch` |

---

## Acceptance Criteria

- [ ] Контракт создаётся под любую активную цель пользователя
- [ ] `due_at` в прошлом → 422
- [ ] Воркер через час помечает просроченные active контракты как broken
- [ ] `fulfill` связывает контракт с check_in и меняет status → fulfilled
- [ ] GET /v1/contracts возвращает `urgency` и `days_until_due` для каждого
- [ ] Контракт со статусом fulfilled/broken нельзя изменить → 409
- [ ] Buddy получает уведомление при создании контракта с buddy_id (через существующий notification сервис)

---

## Что нельзя делать

- Не заменять существующую логику check-in — контракт дополняет, не заменяет
- Не делать контракт обязательным для сдачи check-in (пользователь может сдать check-in без контракта)
- Не реализовывать AI-предложения контрактов в этом спеке (спек 8.1)
