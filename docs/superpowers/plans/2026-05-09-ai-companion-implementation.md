# План реализации — AI-компаньон (подход «AI-призраг»)

**Спек:** `docs/superpowers/specs/2026-05-09-ai-companion-design.md`  
**Статус:** approved by user 2026-05-09  
**Сложность:** M  
**Оценка effort:** 5–7 дней Sonnet 4.6

---

## Цель

Заменить ручное AI-досье на проактивного AI-компаньона, который сам триггерится по времени и событиям, сам собирает контекст и доставляет артефакты через Telegram, in-app и email без действий пользователя.

---

## Критические зависимости

- Phase 3 AI-персонализация (`ai_mode`, `ai_consent`, `ai_circuit_breakers`, `ai_daily_budget`) — должен быть в main
- Telegram бот — webhook и messaging infra
- Daily log entries — для proof draft assembly
- Team proofs и approval flow — для lead brief и alerts
- Timezone-aware cron — для evening ping

---

## Срезы (slices)

### Срез 1: Демонтаж ручного досье
**Цель:** убрать мёртвый код перед добавлением нового.

- [ ] Удалить `web/app/(product)/dossier/page.tsx`
- [ ] Удалить `web/components/product/growth-dossier.tsx`
- [ ] Удалить `web/components/product/growth-dossier.module.css`
- [ ] Удалить API handlers `POST /v1/ai/growth-dossier`, `GET /v1/me/dossiers`, `GET /v1/me/dossiers/:id`
- [ ] Удалить таблицу `dossiers` (миграция `00026_dossiers` → rollback)
- [ ] Удалить `GrowthDossierWidget` из dashboard (если есть ссылка)
- [ ] Удалить dossier-related из роутинга
- [ ] Smoke: убедиться, что `/me/dossier` возвращает 404

**Acceptance:** zero references to dossier/досье в кодовой базе.

---

### Срез 2: AI companion — инфраструктура
**Цель:** фундамент trigger router, context assembler, surface router.

- [ ] Миграция `00026_ai_companion.sql`:
  - `ai_companion_fired` (идемпотентность)
  - `ai_proof_drafts` (черновики)
  - `ai_companion_cache` (кэш)
- [ ] Go пакет `backend/internal/companion/`:
  - `Service` interface
  - `TriggerRouter` — time vs event triggers
  - `ContextAssembler` — сбор данных с privacy filtering
  - `SurfaceRouter` — доставка в Telegram / in-app / email / WebSocket
  - `ShouldFire` — проверка rate limits и идемпотентности
- [ ] API endpoints:
  - `GET /v1/me/ai/notifications`
  - `POST /v1/me/ai/notifications/:id/dismiss`
  - `GET /v1/me/ai/drafts`
  - `POST /v1/me/ai/drafts/:id/accept`
  - `POST /v1/me/ai/drafts/:id/reject`
  - `GET /v1/teams/:id/ai/health` (lead/trusted only)
- [ ] Интеграция с существующим `ai_circuit_breakers` и `ai_daily_budget`
- [ ] TDD: unit tests для `ShouldFire`, `ContextAssembler` privacy modes, `SurfaceRouter`

**Acceptance:** `backend/internal/companion/` компилируется, тесты зелёные, API возвращает пустые списки.

---

### Срез 3: Временные триггеры — часть 1 (evening ping + weekly recap)
**Цель:** первая проактивная фича для пользователя.

- [ ] Cron job `evening_ping` — Пн–Пт 19:00 по timezone пользователя
- [ ] Context assembler для evening ping (streak, goals, proof count, day of week)
- [ ] LLM prompt + TemplateFallback (пул шаблонных вопросов)
- [ ] Validator: 8–25 слов, ends with ?, без praise/shame
- [ ] Surface router → Telegram
- [ ] Cron job `weekly_recap` — Пт 18:00
- [ ] Context assembler для weekly recap (proofs, streak, achievements, next step)
- [ ] LLM prompt + TemplateFallback
- [ ] Surface router → Telegram + in-app notification
- [ ] `ShouldFire` rate limits: 1/день для ping, 1/неделя для recap
- [ ] TDD: unit tests для validator, integration tests для cron trigger

**Acceptance:** alpha-пользователь получает evening ping в Telegram в нужное время и weekly recap по пятницам.

---

### Срез 4: Временные триггеры — часть 2 (lead brief + streak reminder)
**Цель:** проактивность для руководителя и retention.

- [ ] Cron job `lead_weekly_brief` — Пн 09:00
- [ ] Context assembler для lead brief (team aggregates, latency, risks)
- [ ] **Privacy:** никакого raw daily-log даже в full mode
- [ ] LLM prompt + TemplateFallback (статистический brief)
- [ ] Validator: 30–80 слов, 1 risk section, без оценок
- [ ] Surface router → Telegram + email
- [ ] Cron job `streak_reminder` — за 4 часа до полуночи, если streak под угрозой
- [ ] Context: streak count, current goal, time left
- [ ] TemplateFallback: мягкое напоминание
- [ ] Surface router → Telegram
- [ ] TDD: privacy tests, integration tests для email delivery

**Acceptance:** руководитель получает brief в Telegram и email. Пользователь получает streak reminder до полуночи.

---

### Срез 5: Событийные триггеры — proof draft + buddy stalled
**Цель:** автоматизация рутины пользователя.

- [ ] Event listener `daily_log_entry_created`
- [ ] Trigger condition: ≥3 заметки за неделю по одной команде
- [ ] `AssembleProofDraft`: группировка заметок по goals, rationale, confidence
- [ ] LLM prompt + TemplateFallback (proximity grouping)
- [ ] Validator: max 3 candidates, goal принадлежит команде, high-conf требует артефакт
- [ ] Сохранение в `ai_proof_drafts`
- [ ] In-app карточка в proof-форме (AI draft card)
- [ ] API `accept` → заполняет форму пруфа, `reject` → помечает consumed_at
- [ ] Event listener `buddy_no_response_72h`
- [ ] `BuddyStalledAlert`: мягкое предложение напомнить/сменить buddy
- [ ] Surface router → Telegram + in-app
- [ ] TDD: integration tests для proof draft assembly и buddy alert

**Acceptance:** пользователь видет AI-карточку черновика пруфа в форме. Buddy stalled alert приходит в Telegram.

---

### Срез 6: Событийные триггеры — goal risk + streak milestone + fair play
**Цель:** проактивная забота и fair play.

- [ ] Event listener `goal_no_proof_14d`
- [ ] `GoalRiskAlert`: карточка + Telegram, actions: пересмотреть / отложить
- [ ] Event listener `streak_reached_7`, `30`, `100`
- [ ] `StreakMilestone`: поздравление + WebSocket push
- [ ] Event listener `approval_latency_p95 > 48h`
- [ ] `LeaderFairPlayNudge`: нейтральный факт для руководителя, Telegram
- [ ] TDD: unit tests для каждого триггера

**Acceptance:** goal risk и streak milestone приходят в нужный момент. Fair play nudge достигает руководителя.

---

### Срез 7: Frontend — AI-бейджи и карточки
**Цель:** UI для полученных AI-инсайтов.

- [ ] `AIBadge` компонент для dashboard — 1 приоритетный инсайт
- [ ] `AIDraftCard` в proof-форме — принять/отклонить черновик
- [ ] `AINotificationCenter` — список активных AI-инсайтов
- [ ] `AITeamHealthHeader` в team-дашборде руководителя
- [ ] Удалить все ссылки на `/me/dossier` из навигации и dashboard
- [ ] Русский язык для всех UI-текстов (labels, buttons, descriptions)
- [ ] Адаптивность: mobile-first, работает на 320px
- [ ] TDD: frontend unit tests для компонентов

**Acceptance:** dashboard показывает ≤1 AI-бейдж. Proof-форма показывает AI-карточку. Руководитель видит team health header.

---

### Срез 8: Observability и circuit breaker
**Цель:** понимание, что AI делает и когда ломается.

- [ ] Events: `companion_triggered`, `companion_fired`, `companion_fallback`, `companion_dismissed`, `companion_accepted`, `companion_rejected`
- [ ] Интеграция с существующим `ai_circuit_breakers` — per-feature
- [ ] Интеграция с `ai_daily_budget` — глобальный daily budget
- [ ] Логи: INFO на fired, WARN на fallback, ERROR на delivery failure
- [ ] Grafana annotations (или временный лог) для circuit breaker open
- [ ] TDD: integration tests для budget exceeded → fallback, circuit breaker → disable feature

**Acceptance:** fallback rate < 20% в alpha. Circuit breaker не открывается на стабильных фичах.

---

### Срез 9: Релиз и smoke
**Цель:** безопасный rollout.

- [ ] Feature flags: `COMPANION_ENABLED`, `COMPANION_TELEGRAM_ENABLED`, per-feature flags
- [ ] Rollout: alpha team → 1 фича (evening ping) → stable → следующая фича
- [ ] Smoke tests для каждой поверхности (Telegram, in-app, email)
- [ ] Privacy smoke: metadata-only и off режимы не leak raw content
- [ ] Emergency runbook: как отключить все LLM вызовы за 1 команду
- [ ] Документация: обновить API docs, добавить companion в ops runbook

**Acceptance:** alpha проходит 1 неделю без circuit breaker. 0 privacy findings.

---

## Порядок работы

1. **Срез 1** — демонтаж (1 день)
2. **Срез 2** — инфраструктура (1.5 дня)
3. **Срез 3 + 4** — временные триггеры (1.5 дня)
4. **Срез 5 + 6** — событийные триггеры (1.5 дня)
5. **Срез 7** — frontend (1 день)
6. **Срез 8** — observability (0.5 дня)
7. **Срез 9** — релиз и smoke (0.5 дня)

**Итого:** ~7 дней Sonnet 4.6, или 5 дней если параллелить frontend и backend.

---

## Риски и митигация

| Риск | Вероятность | Митигация |
|---|---|---|
| Telegram webhook не доставляет | Средняя | Fallback на email для руководителей, in-app для пользователей |
| LLM timeout >2s для evening ping | Средняя | TemplateFallback с пулом шаблонов, circuit breaker |
| Privacy leak в proof draft | Низкая | TDD: assembler tests для каждого mode/consent |
| Cron timezone неверный | Средняя | Использовать user timezone из профиля, default UTC+3 |
| Budget exceeded | Средняя | Daily budget 500K tokens, fallback при превышении |
| Пользователь не привязал Telegram | Высокая | Пропускать Telegram triggers, fallback на in-app |

---

## Post-merge

- [ ] Обновить `docs/superpowers/specs/2026-05-08-spec-8-4-ai-growth-dossier.md` — пометить устаревшим, сослаться на новый спек
- [ ] Обновить `docs/superpowers/plans/README.md` — добавить ссылку на план
- [ ] Удалить навык `ai-growth-dossier` из `.claude/skills/` если существует
