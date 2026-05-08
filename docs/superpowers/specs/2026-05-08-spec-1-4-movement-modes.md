# Спек 1.4 — Movement Modes (Режимы движения)

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 1 · Данные  
**Зависимости:** Существующая таблица `goals`  
**Сложность:** S  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

ProofForge не привязан к жёстким 7-дневным сезонам. Разные пользователи нуждаются в разных режимах:

| Режим | Когда | Пример |
|-------|-------|--------|
| `single_proof` | Разовый артефакт | Конспект по статье, MR для руководителя |
| `regular_rhythm` | ИПР, долгое обучение | Раз в неделю пруф по Kotlin Coroutines |
| `challenge` | Telegram-клуб, группа | 28 дней по ML, старт/финиш у всех общий |
| `work_initiative` | Исследование в команде | Research AI-инструментов, добровольно |
| `free_goal` | Без жёсткого срока | Читать книги по распределённым системам |

Это поле в цели влияет на: какой UI показывать при создании цели, как считать метрики активности, что показывать в умной карточке "Что сейчас".

---

## Что реализовать

### Backend
1. SQL миграция `00021_movement_modes.sql` — alter goals
2. Обновить `backend/internal/goals/domain.go` — добавить типы и поля
3. Обновить `backend/internal/goals/service.go` — валидация новых полей
4. Обновить `backend/internal/goals/http_handler.go` — принимать и возвращать новые поля

### Frontend
1. Обновить `web/components/product/goal-setup-screen.tsx` — добавить шаг выбора режима
2. Обновить `web/lib/types.ts` — добавить новые типы

---

## Схема базы данных

```sql
-- +goose Up

ALTER TABLE goals
  ADD COLUMN movement_mode     TEXT NOT NULL DEFAULT 'regular_rhythm'
                               CHECK (movement_mode IN (
                                 'single_proof',
                                 'regular_rhythm',
                                 'challenge',
                                 'work_initiative',
                                 'free_goal'
                               )),
  ADD COLUMN rhythm_cadence    TEXT
                               CHECK (rhythm_cadence IN (
                                 'daily', 'weekly', 'biweekly', 'custom', NULL
                               )),
  ADD COLUMN challenge_duration_days INTEGER CHECK (challenge_duration_days IN (7, 14, 28, 42)),
  ADD COLUMN challenge_starts_at     TIMESTAMPTZ,
  ADD COLUMN challenge_ends_at       TIMESTAMPTZ;

-- Бизнес-правила через check constraint:
-- regular_rhythm должен иметь cadence
-- challenge должен иметь duration
ALTER TABLE goals
  ADD CONSTRAINT goals_rhythm_cadence_required
    CHECK (
      movement_mode != 'regular_rhythm' OR rhythm_cadence IS NOT NULL
    ),
  ADD CONSTRAINT goals_challenge_duration_required
    CHECK (
      movement_mode != 'challenge' OR challenge_duration_days IS NOT NULL
    );

-- +goose Down
ALTER TABLE goals
  DROP CONSTRAINT IF EXISTS goals_rhythm_cadence_required,
  DROP CONSTRAINT IF EXISTS goals_challenge_duration_required,
  DROP COLUMN IF EXISTS movement_mode,
  DROP COLUMN IF EXISTS rhythm_cadence,
  DROP COLUMN IF EXISTS challenge_duration_days,
  DROP COLUMN IF EXISTS challenge_starts_at,
  DROP COLUMN IF EXISTS challenge_ends_at;
```

---

## Go Domain (обновление `backend/internal/goals/domain.go`)

```go
// Добавить к существующим типам:

type MovementMode string

const (
    MovementModeSingleProof    MovementMode = "single_proof"
    MovementModeRegularRhythm  MovementMode = "regular_rhythm"
    MovementModeChallenge      MovementMode = "challenge"
    MovementModeWorkInitiative MovementMode = "work_initiative"
    MovementModeFreeGoal       MovementMode = "free_goal"
)

type RhythmCadence string

const (
    RhythmCadenceDaily    RhythmCadence = "daily"
    RhythmCadenceWeekly   RhythmCadence = "weekly"
    RhythmCadenceBiweekly RhythmCadence = "biweekly"
    RhythmCadenceCustom   RhythmCadence = "custom"
)

// Обновить структуру Goal (добавить поля):
type Goal struct {
    // ... существующие поля ...
    MovementMode          MovementMode
    RhythmCadence         *RhythmCadence
    ChallengeDurationDays *int
    ChallengeStartsAt     *time.Time
    ChallengeEndsAt       *time.Time
}

// Helper: нужна ли cadence для этого режима
func (m MovementMode) RequiresCadence() bool {
    return m == MovementModeRegularRhythm
}

// Helper: активен ли challenge прямо сейчас
func (g *Goal) IsChallengeActive() bool {
    if g.MovementMode != MovementModeChallenge {
        return false
    }
    now := time.Now()
    return g.ChallengeStartsAt != nil &&
           g.ChallengeEndsAt != nil &&
           now.After(*g.ChallengeStartsAt) &&
           now.Before(*g.ChallengeEndsAt)
}

// Для умной карточки: когда ожидается следующий пруф
func (g *Goal) NextProofExpectedAt() *time.Time {
    switch g.MovementMode {
    case MovementModeSingleProof:
        return nil // нет расписания
    case MovementModeRegularRhythm:
        // вычислить по cadence от последнего check-in
        return calculateNextFromCadence(g.RhythmCadence)
    case MovementModeChallenge:
        return g.ChallengeEndsAt
    default:
        return nil
    }
}
```

---

## Обновление API

**POST /v1/goals — Request (расширенный):**
```json
{
  "title": "Kotlin Coroutines",
  "description": "Изучить structured concurrency и cancellation",
  "movement_mode": "regular_rhythm",
  "rhythm_cadence": "weekly"
}
```

**POST /v1/goals с challenge:**
```json
{
  "title": "28-дневный ML челлендж",
  "movement_mode": "challenge",
  "challenge_duration_days": 28,
  "challenge_starts_at": "2026-06-01T00:00:00Z"
}
```
*challenge_ends_at вычисляется автоматически = starts_at + duration_days*

**GET /v1/goals/:id — Response (расширенный):**
```json
{
  "data": {
    "id": "uuid",
    "title": "Kotlin Coroutines",
    "movement_mode": "regular_rhythm",
    "rhythm_cadence": "weekly",
    "challenge_duration_days": null,
    "next_proof_expected_at": "2026-05-15T00:00:00Z"
  }
}
```

---

## Frontend: Goal Setup Screen

Файл: `web/components/product/goal-setup-screen.tsx`

Добавить **Шаг 2: Выбери режим** между вводом цели и выбором buddy.

**UI шага выбора режима (мобайл):**
```
[EYEBROW] КАК ТЫ БУДЕШЬ ДВИГАТЬСЯ?

[Карточка A] Разовый пруф
  Сдать один конкретный результат
  → для MR, конспекта, исследования

[Карточка B] Регулярный ритм  ← рекомендованный (border --win)
  Сдавать пруф по расписанию
  → еженедельно / двухнедельно

[Карточка C] Челлендж
  Общий старт и финиш
  → 7, 14, 28 или 42 дня

[Карточка D] Свободная цель
  Без жёсткого ритма
  → долгосрочное развитие
```

После выбора "Регулярный ритм" — появляется sub-шаг:
```
[КАК ЧАСТО?]
[Раз в день] [Раз в неделю ✓] [Раз в две недели] [Свой ритм]
```

После выбора "Челлендж" — sub-шаг:
```
[НА СКОЛЬКО ДНЕЙ?]
[7 дней] [14 дней] [28 дней ✓] [42 дня]
[Дата старта: ________]
```

**CSS (goal-setup-screen.module.css) — добавить:**
```css
.modeCard {
  border: 2px solid var(--border);
  padding: 16px;
  cursor: pointer;
  transition: border-color 140ms ease;
}

.modeCard:hover {
  border-color: var(--border-strong);
}

.modeCard.selected {
  border-color: var(--win);
}

.modeCardTitle {
  font-size: 15px;
  font-weight: 700;
  color: var(--ink-primary);
  margin-bottom: 4px;
}

.modeCardDescription {
  font-size: 12px;
  color: var(--ink-mono);
}

.recommended {
  font-size: 10px;
  color: var(--win);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  margin-bottom: 6px;
}
```

---

## Acceptance Criteria

- [ ] Существующие goals получают `movement_mode = 'regular_rhythm'` по умолчанию (миграция)
- [ ] `POST /v1/goals` принимает `movement_mode`, `rhythm_cadence`, `challenge_duration_days`
- [ ] `regular_rhythm` без `rhythm_cadence` → 422 `rhythm_cadence_required`
- [ ] `challenge` без `challenge_duration_days` → 422 `challenge_duration_required`
- [ ] `challenge_ends_at` вычисляется автоматически из starts_at + duration
- [ ] GET /v1/goals возвращает все новые поля, включая `next_proof_expected_at`
- [ ] Goal setup экран показывает шаг выбора режима после ввода названия
- [ ] По умолчанию выбран "Регулярный ритм", визуально отмечен рекомендованным
- [ ] Выбор режима валидируется на клиенте перед отправкой

---

## Что нельзя делать

- Не делать movement_mode обязательным в UI (если пользователь пропустил шаг — default `regular_rhythm`)
- Не ломать существующие goals и check-ins
- Не добавлять новые поля в check-in в этом спеке
