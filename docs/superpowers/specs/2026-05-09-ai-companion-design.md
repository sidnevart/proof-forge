# Спек — AI-компаньон (проактивный интеллект, подход «AI-призраг»)

**Дата:** 2026-05-09
**Статус:** design spec
**Группа:** 8 · AI-персонализация
**Сложность:** M
**Заменяет:** `2026-05-08-spec-8-4-ai-growth-dossier.md` — старый спек 8.4 (ручное досье) считаем устаревшим

---

## Принцип

AI не ждёт действия пользователя. Он сам триггерится по времени или событиям, сам собирает контекст, сам формирует артефакт и сам доставляет его в нужную поверхность. Пользователь получает ценность, не нажимая кнопок.

---

## Что убираем

- Страница `/me/dossier` и все её подстраницы
- Понятие «досье» как ручного документа
- Ручной выбор периода и кнопку «Сгенерировать досье»
- Хранение Markdown-документов в таблице `dossiers`

Эти артефакты удаляются из кодовой базы в рамках реализации.

---

## Архитектура

```
┌─────────────────────────────────────────────────────────────┐
│  Trigger Router (кто, когда, по какому каналу)               │
│  • Time triggers (cron)                                        │
│  • Event triggers (Kafka/webhook-style)                      │
├─────────────────────────────────────────────────────────────┤
│  Context Assembler (собирает данные из БД под privacy mode) │
├─────────────────────────────────────────────────────────────┤
│  LLM Provider + Template Fallback                           │
│  • Claude / Ollama / template                                │
│  • Circuit breaker per feature                                │
│  • Budget & rate limits                                       │
├─────────────────────────────────────────────────────────────┤
│  Output Validator (проверяет формат, границы, запреты)      │
├─────────────────────────────────────────────────────────────┤
│  Surface Router (куда доставить результат)                  │
│  • Telegram bot message                                       │
│  • In-app notification (бейдж/карточка/черновик)             │
│  • Email (для руководителей)                                 │
│  • WebSocket push (real-time инсайты)                        │
└─────────────────────────────────────────────────────────────┘
```

---

## Триггеры

### Временные (cron)

| ID | Когда | Для кого | Что делает | Поверхность |
|---|---|---|---|---|
| `evening_ping` | Пн–Пт 19:00 по локальному времени пользователя | Пользователь | Вечерний пинг с контекстом дня | Telegram |
| `weekly_recap` | Пт 18:00 | Пользователь | Итог недели: пруфы, streak, следующий шаг | Telegram + in-app |
| `lead_weekly_brief` | Пн 09:00 | Руководитель / trusted | Team brief: кто на высоте, кто под риском | Telegram + email |
| `monthly_growth_digest` | 1-е число, 10:00 | Пользователь | Итог месяца, новые навыки, достижения | In-app карточка |
| `streak_reminder` | За 4 часа до полуночи, если streak под угрозой | Пользователь | Мягкое напоминание: «Ещё можно сделать пруф» | Telegram |

### Событийные

| ID | Событие | Для кого | Что делает | Поверхность |
|---|---|---|---|---|
| `proof_draft_ready` | `daily_log_entry_created` (≥3 заметки в неделю) | Пользователь | Черновик пруфа собран, одной кнопкой отправить | In-app карточка |
| `buddy_stalled` | `buddy_no_response_72h` | Пользователь | «Buddy не отвечает 3 дня — напомни или смени» | Telegram |
| `goal_risk` | `goal_no_proof_14d` | Пользователь + руководитель | «Цель «X» без пруфов 14 дней» | In-app + Telegram |
| `streak_milestone` | `streak_reached_7`, `30`, `100` | Пользователь | Поздравление с контекстом | Telegram |
| `leader_fair_play_nudge` | `approval_latency_p95 > 48h` | Руководитель | «Среднее время approve растёт — проверь очередь» | Telegram |
| `team_health_alert` | `team_proof_rate_drop_30pct_week` | Руководитель | «Активность команды упала на 30%» | Telegram + email |

---

## Поверхности доставки

### 1. Telegram
- Основной канал для пользователя: пинги, рекапы, напоминания.
- Основной канал для руководителя: briefs, health alerts.
- Формат: текст + inline-кнопки (одобрить черновик, перейти, скрыть).
- Не требует открытия приложения.

### 2. In-app (бейджи и карточки)
- **Dashboard AI-бейдж** — приоритетный инсайт на сегодня (один, самый важный).
- **Proof-форма AI-карточка** — черновик пруфа, собранный из daily log: «AI собрал пруф из 3 заметок → Отправить / Отредактировать / Отклонить».
- **Goal-страница AI-карточка** — рекомендация следующего шага или риск.
- **Team-дашборд руководителя** — AI-шапка с team health и fair play индикатором.

### 3. Email
- Только для руководителей.
- Weekly team brief в понедельник утром.
- Формат: сводка + ссылки на детали в приложении.
- Не содержит raw daily-log или приватных комментариев.

### 4. WebSocket push
- Real-time инсайты: streak milestone, buddy stalled, goal risk.
- Показываются как toast/баннер при открытом приложении.

---

## Фичи в деталях

### F1. Evening ping

**Триггер:** cron, Пн–Пт 19:00 по часовому поясу пользователя.
**Контекст:** streak, активные цели, количество пруфов сегодня, день недели.
**Privacy:** никакого raw daily-log даже в full mode.
**Вывод:** 1 вопрос, 8–25 слов, без похвалы/стыда/контроля.
**Fallback:** шаблонный вопрос из пула (зависит от streak и дня).
**Доставка:** Telegram message. Если пользователь не привязал Telegram — пропускаем.

### F2. Weekly recap

**Триггер:** cron, каждую пятницу 18:00.
**Контекст:** пруфы за неделю, streak, достижения, активные цели, следующий логичный шаг.
**Вывод:** 3–5 пунктов, Markdown-like в Telegram, с inline-кнопкой «Посмотреть в приложении».
**Fallback:** статистический шаблон (счётчики + стандартный next step).
**Доставка:** Telegram + in-app карточка на dashboard.

### F3. Lead weekly brief

**Триггер:** cron, каждый понедельник 09:00.
**Контекст:** team proofs за неделю, approval latency, streak агрегаты, goal coverage, риски (кто пропал, кто не одобряет).
**Privacy:** никакого raw daily-log. Только агрегаты и approved proof metadata.
**Вывод:** 30–80 слов, 1 секция рисков или «Без рисков», без оценок «слабый/сильный».
**Fallback:** статистический brief из счётчиков.
**Доставка:** Telegram + email.
**Rate limit:** 1 brief в неделю на команду.

### F4. Proof draft assembly

**Триггер:** `daily_log_entry_created` и ≥3 заметки за неделю, или ручной запрос через кнопку в proof-форме.
**Контекст:** заметки автора за текущую неделю, активные team goals, last approved proof metadata.
**Вывод:** до 3 candidate-групп с rationale и confidence.
**Validation:** goal принадлежит той же команде, note IDs принадлежат автору, high-confidence требует артефакта.
**Fallback:** TemplateProvider группирует unconsumed entries с артефактами по goal/date proximity.
**Доставка:** in-app карточка в proof-форме. Пользователь нажимает «Отправить» — проходит обычная валидация Phase 2.

### F5. Buddy stalled alert

**Триггер:** `buddy_no_response_72h` — buddy не одобрил/не отклонил пруф 72 часа.
**Контекст:** пруф, buddy alias, дата отправки.
**Вывод:** мягкое предложение: «Напомни buddy в Telegram / Сменить buddy».
**Fallback:** статическое сообщение с кнопками.
**Доставка:** Telegram + in-app notification.

### F6. Goal risk alert

**Триггер:** `goal_no_proof_14d` — цель без пруфов 14 дней.
**Контекст:** goal title, last proof date, streak связанный с goal.
**Вывод:** «Цель «X» без пруфов 14 дней. Пересмотреть цель? / Напомнить через 3 дня».
**Fallback:** статическое сообщение.
**Доставка:** in-app карточка + Telegram.

### F7. Streak milestone

**Триггер:** `streak_reached_7`, `30`, `100`.
**Контекст:** streak count, top goal за streak, total proofs.
**Вывод:** поздравление + контекст («7 дней подряд — лучшая цель «X»»).
**Fallback:** шаблонное поздравление.
**Доставка:** Telegram toast + WebSocket push.

### F8. Leader fair play nudge

**Триггер:** `approval_latency_p95 > 48h` или `self_approval_rate > 80%` (руководитель сам одобряет свои пруфы).
**Контекст:** агрегаты approve/reject, latency, self-approval count.
**Вывод:** нейтральный факт + рекомендация («Среднее время approve — 52 часа. Проверь очередь»).
**Fallback:** статистическая сводка.
**Доставка:** Telegram.

### F9. Team health alert

**Триггер:** `team_proof_rate_drop_30pct_week` — количество пруфов в команде упало на 30% за неделю.
**Контекст:** week-over-week proof count, active members count, new members.
**Вывод:** факт + рекомендация («Активность упала на 30%. Провести stand-up?»).
**Fallback:** статистика.
**Доставка:** Telegram + email.

---

## Privacy и безопасность

- AIMode на уровне команды (`off`, `metadata-only`, `full`) + `ai_consent` на уровне пользователя.
- Временные триггеры (evening ping, weekly recap) никогда не отправляют raw daily-log в LLM.
- Событийные триггеры (proof draft) отправляют content только в `full` mode + согласие.
- Lead brief никогда не получает raw daily-log даже в `full` mode.
- AI-ответы помечены `✨` или `AI` в UI.
- Raw prompt и raw completion не логируются (только метаданные).
- Email и Telegram никогда не содержат приватные URL, токены, invite codes.

---

## Rate limits и budget

| Фича | Rate limit | Budget |
|---|---|---|
| `evening_ping` | 1/день на пользователя | — |
| `weekly_recap` | 1/неделя на пользователя | — |
| `lead_weekly_brief` | 1/неделя на команду | — |
| `proof_draft` | 5/неделя на пользователя | — |
| `streak_reminder` | 1/день на пользователя | — |
| **Глобальный daily budget** | — | 500K tokens / день |

При превышении budget — все LLM-вызовы падают на TemplateFallback до следующего дня.

---

## Backend

### Миграция

Удаляем таблицу `dossiers` (если была создана). Создаём таблицы для AI-компаньона:

```sql
-- 00026_ai_companion.sql

-- Триггеры, которые уже сработали (идемпотентность)
CREATE TABLE ai_companion_fired (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    feature     TEXT NOT NULL,
    trigger_id  TEXT NOT NULL,  -- "evening_ping:2026-05-09:user-123"
    fired_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, trigger_id)
);
CREATE INDEX idx_ai_companion_fired_user ON ai_companion_fired(user_id, fired_at DESC);

-- Черновики пруфов, собранные AI
CREATE TABLE ai_proof_drafts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_id     UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    goal_id     UUID REFERENCES goals(id) ON DELETE SET NULL,
    note_ids    UUID[] NOT NULL,
    rationale   TEXT,
    confidence  TEXT CHECK (confidence IN ('low','medium','high')),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    consumed_at TIMESTAMPTZ  -- когда пользователь отправил или отклонил
);
CREATE INDEX idx_ai_proof_drafts_user ON ai_proof_drafts(user_id, consumed_at NULLS FIRST);

-- Кэш AI-выводов (brief, recap)
CREATE TABLE ai_companion_cache (
    cache_key   TEXT PRIMARY KEY,
    feature     TEXT NOT NULL,
    payload     JSONB NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Circuit breaker (переиспользуем из Phase 3)
-- ai_circuit_breakers уже существует

-- Budget (переиспользуем из Phase 3)
-- ai_daily_budget уже существует
```

### Service interface

```go
package companion

type Service interface {
    // Временные триггеры
    EveningPing(ctx context.Context, userID string) error
    WeeklyRecap(ctx context.Context, userID string) error
    LeadWeeklyBrief(ctx context.Context, teamID string, leaderID string) error
    StreakReminder(ctx context.Context, userID string) error

    // Событийные триггеры
    AssembleProofDraft(ctx context.Context, userID string, teamID string) (*ProofDraft, error)
    BuddyStalledAlert(ctx context.Context, proofID string, buddyID string) error
    GoalRiskAlert(ctx context.Context, goalID string, userID string) error
    StreakMilestone(ctx context.Context, userID string, streak int) error
    LeaderFairPlayNudge(ctx context.Context, teamID string, leaderID string) error
    TeamHealthAlert(ctx context.Context, teamID string, leaderID string) error

    // Утилиты
    ShouldFire(ctx context.Context, userID string, feature string, triggerID string) (bool, error)
    CacheGet(ctx context.Context, cacheKey string) (*CacheEntry, error)
    CacheSet(ctx context.Context, entry *CacheEntry) error
}
```

### Surface Router

```go
type Surface string
const (
    SurfaceTelegram Surface = "telegram"
    SurfaceInApp    Surface = "inapp"
    SurfaceEmail    Surface = "email"
    SurfaceWebSocket Surface = "websocket"
)

type SurfaceRouter interface {
    Route(ctx context.Context, userID string, surfaces []Surface, payload *DeliveryPayload) error
}
```

---

## Frontend

### Удаляем

- `/me/dossier/page.tsx` → удалить
- `/me/dossier/new` → удалить
- `/me/dossier/:id` → удалить
- `GrowthDossierWidget` → удалить
- `growth-dossier.module.css` → удалить
- API `POST /v1/ai/growth-dossier` → удалить
- API `GET /v1/me/dossiers` → удалить
- API `GET /v1/me/dossiers/:id` → удалить

### Новые UI-элементы

**Dashboard AI-бейдж** — одна карточка на dashboard, показывает приоритетный инсайт на сегодня.
```tsx
<AIBadge
  type="goal_risk | proof_draft | streak_reminder | weekly_recap"
  title="..."
  actions={[
    { label: "Отправить", onClick: ... },
    { label: "Отложить", onClick: ... },
  ]}
/>
```

**Proof-форма AI-карточка** — внутри формы создания пруфа, если есть черновик:
```tsx
{draft && (
  <AIDraftCard
    rationale={draft.rationale}
    noteCount={draft.note_ids.length}
    onAccept={() => populateForm(draft)}
    onReject={() => dismissDraft(draft.id)}
  />
)}
```

**Team-дашборд руководителя** — AI-шапка:
```tsx
<AITeamHealthHeader
  teamHealthScore={78}
  fairPlayStatus="green"
  pendingApprovals={3}
  alerts={[{ type: "latency", message: "Среднее время approve — 52ч" }]}
/>
```

**In-app notification center** — список AI-инсайтов:
```tsx
<AINotificationCenter>
  {notifications.map(n => (
    <AINotificationItem
      key={n.id}
      feature={n.feature}
      text={n.text}
      actions={n.actions}
      dismissed={n.dismissed_at}
    />
  ))}
</AINotificationCenter>
```

---

## API

### Новые endpoints

```
GET  /v1/me/ai/notifications          — список активных AI-инсайтов
POST /v1/me/ai/notifications/:id/dismiss — скрыть инсайт
GET  /v1/me/ai/drafts                — список черновиков пруфов
POST /v1/me/ai/drafts/:id/accept    — принять черновик, заполнить форму
POST /v1/me/ai/drafts/:id/reject    — отклонить черновик
GET  /v1/teams/:id/ai/health         — team health score (lead/trusted only)
```

### Удаляемые endpoints

```
POST /v1/ai/growth-dossier  → удалить
GET  /v1/me/dossiers        → удалить
GET  /v1/me/dossiers/:id    → удалить
```

---

## Observability

### Events

| Event | Свойства |
|---|---|
| `companion_triggered` | `feature`, `trigger_type` (time/event), `trigger_id`, `user_id` |
| `companion_fired` | `feature`, `surface`, `user_id`, `latency_ms` |
| `companion_fallback` | `feature`, `reason` (budget/circuit/timeout/validation), `user_id` |
| `companion_dismissed` | `feature`, `user_id`, `notification_id` |
| `companion_accepted` | `feature`, `user_id`, `draft_id` |
| `companion_rejected` | `feature`, `user_id`, `draft_id` |

### Forbidden properties

- Raw prompt text
- Raw completion text
- Daily-log text, proof text, comments
- Email addresses, Telegram chat_id

### Logs

- INFO: trigger fired, surface delivered
- WARN: fallback activated, circuit breaker open
- ERROR: surface delivery failed, budget store failure

---

## Тесты

### Unit tests

- `ShouldFire` — идемпотентность, rate limits
- Privacy assembler — каждый mode/consent
- Output validator — длина, запретные слова, формат
- Template fallback — корректный output schema
- Surface router — Telegram/email/in-app доставка

### Integration tests

- Cron trigger запускает EveningPing в нужное время
- Event trigger `daily_log_entry_created` → AssembleProofDraft
- BuddyStalledAlert отправляет Telegram сообщение
- LeadWeeklyBrief доступен только lead/trusted
- GoalRiskAlert не отправляется, если goal в `off` mode
- Budget exceeded → все вызовы fallback
- Cache hit → не ходит в LLM

### E2E

- Пользователь видет AI-бейдж на dashboard
- Пользователь принимает черновик пруфа — форма заполняется
- Пользователь отклоняет черновик — карточка исчезает
- Руководитель получает weekly brief в Telegram

---

## Acceptance Criteria

- [ ] Удалены все страницы, компоненты и API ручного досье
- [ ] `evening_ping` отправляется в Telegram Пн–Пт 19:00 (timezone-aware)
- [ ] `weekly_recap` отправляется каждую пятницу 18:00
- [ ] `lead_weekly_brief` отправляется каждый понедельник 09:00
- [ ] `proof_draft` собирается при ≥3 daily-log заметках в неделю
- [ ] `buddy_stalled` отправляется через 72ч неответа buddy
- [ ] `goal_risk` отправляется при 14 днях без пруфов по цели
- [ ] `streak_milestone` отправляется на 7, 30, 100 streak
- [ ] `leader_fair_play_nudge` отправляется при latency > 48h
- [ ] `team_health_alert` отправляется при drop > 30%
- [ ] Каждый AI-вывод имеет TemplateFallback
- [ ] Каждый AI-инсайт можно скрыть (dismiss)
- [ ] Dashboard показывает не более 1 AI-бейджа одновременно
- [ ] AI-черновик пруфа можно принять или отклонить одной кнопкой
- [ ] Rate limits не превышаются
- [ ] Privacy tests: 0 leakов в metadata-only и off режимах

---

## Что нельзя делать

- Не создавать отдельную страницу «AI-компаньон» — он бесшовен
- Не требовать от пользователя ручного запуска AI
- Не отправлять raw daily-log в LLM для lead brief
- Не использовать LLM без TemplateFallback
- Не показывать raw prompt пользователю
- Не логировать raw completion
- Не делать AI chat-bot «спроси что угодно»
- Не обучать модели на пользовательских данных

---

## Релиз

1. Удалить ручное досье (frontend + backend + БД)
2. Добавить AI-компаньон инфраструктуру (trigger router, context assembler, surface router)
3. Включить `evening_ping` для alpha
4. Включить `weekly_recap` для alpha
5. Включить `proof_draft` для alpha
6. Включить `lead_weekly_brief` для alpha leads
7. Остальные триггеры — после стабилизации

Emergency controls:
- `COMPANION_ENABLED=false` — отключить все LLM-вызовы
- `COMPANION_TELEGRAM_ENABLED=false` — отключить Telegram
- Team `ai_mode=off` — отключить AI per team
- Per-feature circuit breaker — отключить конкретный триггер

---

## Саморевью спека

1. **Placeholder scan:** нет TBD, все фичи описаны.
2. **Internal consistency:** privacy mode matrix соответствует Phase 3 spec. Lead brief не получает raw daily-log — соответствует invariant.
3. **Scope check:** фокус на подход A (AI-призраг), без страницы AI-чата и без предсказательной аналитики. Это один implementation plan.
4. **Ambiguity check:** «средняя автономия» определена как «AI сам доставляет рутину, важные решения — человек». Все фичи соответствуют: AI предлагает, но approve, goal change, buddy change — ручные.
