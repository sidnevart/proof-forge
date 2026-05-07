# ProofForge Teams — Habit Loop Design

**Дата:** 2026-05-07
**Автор:** brainstorm-сессия (Claude + Артём)
**Статус:** черновик, требует ревью пользователя
**Связанные спеки:**
- `2026-05-01-mvp-architecture-package-design.md`
- `2026-05-01-product-identity-package-design.md`

---

## 1. Цель и продуктовое позиционирование

### Что мы строим
**ProofForge Teams** — корпоративный режим существующего ProofForge, в котором тимлид и его команда совместно ведут учёт выполнения индивидуальных планов развития (ИПР) через ежедневный learning-log и еженедельный proof с артефактом.

### Какую боль закрываем
Тимлид не может проконтролировать выполнение ИПР сотрудника между performance-ревью. Сотрудник забивает на ИПР через 2 недели после составления. Когда подходит ревью — обе стороны выдумывают, что «вроде что-то делали».

### Чем отличаемся от существующего ProofForge (`personal` circles)
Принципиально другая социальная конструкция:
- Цели у каждого свои (а не общая тема), но контекст — общий тимлид.
- Апрув иерархичный (тимлид + доверенные), а не peer-to-peer.
- Видимость только внутри команды, наружу не утекает.
- Привычка формируется через двухслойную модель (см. §3), которой нет в personal-circles.

### Принцип привычки (одной фразой)
> Sotrudnik пишет в Telegram-бота **каждый день одну строчку про обучение** (learning-log), и **раз в неделю собирает из этого лога proof с артефактом**, который тимлид апрувит. Ничего больше не нужно делать вручную.

### Метрики успеха MVP
- **D7 retention sotrudnika в Telegram-боте ≥ 60%** (написал log хотя бы 4 дня из 7 на второй неделе).
- **Доля weekly-proof'ов, апрувленных в течение 48ч ≥ 70%** (тимлид реально реагирует).
- **Доля sotrudnikov с не-нулевым стриком через 30 дней ≥ 50%**.
- Долгосрочная (вне MVP-метрик): доля ИПР-пунктов с прогрессом ≥ 50% к концу квартала против baseline без ProofForge.

### Ключевые проектные решения (зафиксированы в брейншторме)

| Решение | Выбор |
|---|---|
| Целевой сценарий | A — корпоративный (тимлид + sotrudnik + ИПР) |
| Якорь привычки | №2 — proof в момент статуса, фундамент для других слоёв |
| Точка входа | Telegram (web вторичен) |
| Награда после proof | Бот мгновенно (стрик), тимлид-как-buddy реагирует асинхронно |
| Тип proof | Только структурированные артефакты (ссылка/файл/PR/сертификат) |
| Skip-day policy | Стрик считает только пн–пт |
| Видимость в команде | Все участники видят proof'ы всех |
| MVP-срез | Полный двухслойный flow (daily + weekly + лента) |
| Команда vs круг | `team` — отдельная новая сущность |
| Trusted approver | Апрувит всех в команде |
| Multi-membership | Sotrudnik может быть в personal-круге и в team одновременно |
| AI-фичи | Полный пакет: пинг, сборка proof, briefing |
| LLM | Kimi K2 (cloud) через Ollama API, fallback на шаблоны |

### Явный YAGNI
- Нет SSO/корп-логина, нет интеграции с HR-системами, нет SLA.
- Нет мультиязычности (UI только русский).
- Нет mobile-app (web responsive + Telegram достаточно).
- Нет cross-team видимости и иерархий из нескольких уровней.
- Нет публичной ленты для team-режима — приватность жёсткая.
- Нет лидербордов, бейджей, badge-коллекционирования — только стрик.

---

## 2. Аналитика и сбор метрик

### Принципы
1. Не подключаем внешние SaaS-аналитики (Amplitude, Mixpanel, Posthog) на MVP.
2. События пишем в Postgres — таблица `analytics_events` с partitioning by month.
3. Дашборды — Grafana поверх Postgres (datasource напрямую).
4. Каждое событие имеет одинаковую базовую структуру.
5. Транзакционная гарантия: событие пишется в той же транзакции, что и доменное действие.

### Модель данных событий

```
analytics_events
├── id              bigserial
├── ts              timestamptz   not null  -- когда произошло
├── event_name      text          not null  -- из закрытого списка
├── user_id         bigint        nullable
├── team_id         bigint        nullable
├── goal_id         bigint        nullable
├── proof_id        bigint        nullable
├── source          text          not null  -- 'web' | 'telegram_bot' | 'cron' | 'system'
├── properties      jsonb         not null
└── PARTITION BY RANGE (ts)
```

### Закрытый список событий MVP

**Habit-attribution (sotrudnik):**
- `daily_log_prompted` / `daily_log_submitted` / `daily_log_skipped`
- `streak_incremented` / `streak_broken` / `streak_frozen`

**Weekly proof:**
- `weekly_assembly_prompted` / `weekly_assembly_started` / `weekly_assembly_completed` / `weekly_assembly_skipped`
- `proof_submitted` / `proof_approved` / `proof_rejected` / `proof_commented`

**Активность тимлида/команды:**
- `team_feed_opened` / `notification_sent` / `notification_clicked`

**Onboarding:**
- `team_created` / `team_member_invited` / `team_member_joined` / `telegram_linked`

**AI:**
- `personalization_invoked` / `personalization_fallback` / `personalization_accepted` / `personalization_overridden`
- `user_profile_viewed` / `alert_signal_triggered`

Никаких «user_clicked_button» — только метрики, попадающие в дашборды.

### Дашборды Grafana (MVP)

1. **Habit health.** D7 retention по когортам, средняя длина стрика, распределение `daily_log_submitted` по часам, доля sotrudnikov с активным стриком.
2. **Approval health.** Время до апрува (p50/p90), доля апрувов от тимлида vs trusted, доля реджектов, доля без реакции > 48ч (alert).
3. **Onboarding funnel.** team_created → invited → joined → telegram_linked → first daily_log → first proof.

### Per-user метрики

Три отдельных кейса с разной видимостью:

#### 1. Sotrudnik про себя (`/me`, `/stats` в боте)
- Текущий и лучший стрик, дней в логе всего.
- Heatmap активности (30/90 дней).
- Прогресс по каждому пункту ИПР.
- История weekly proof'ов.
- Среднее время от submit до апрува.

#### 2. Тимлид/trusted про члена команды (`/teams/:id/members/:userId`)
Всё из (1) **плюс**:
- Динамика стрика за последние недели.
- Список ИПР-целей с прогрессом.
- Все proof'ы (включая rejected).
- Тревожные сигналы (нет лога 5 дней, стрик сломался впервые, ни одного proof за 14 дней).
- Время реакции тимлида на proof'ы этого члена.

**Не показывает** содержимое daily learning-log полностью — только статистику.

#### 3. Sotrudnik про коллегу (`/teams/:id/members/:userId/peer`)
- Стрик, длина участия.
- Список ИПР-целей (только заголовки).
- Только апрувленные proof'ы.
- Heatmap активности.

**Не показывает** daily-log, тревожные сигналы, время реакции тимлида.

### Видимость метрик — таблица

| Метрика | Sam о себе | Тимлид о sam | Trusted о sam | Коллега о sam |
|---|---|---|---|---|
| Стрик (текущий, лучший) | ✓ | ✓ | ✓ | ✓ |
| Heatmap активности | ✓ | ✓ | ✓ | ✓ |
| Daily-log содержимое | ✓ | ✗ | ✗ | ✗ |
| Daily-log счётчик/динамика | ✓ | ✓ | ✓ | ✗ |
| Список ИПР-целей (заголовки) | ✓ | ✓ | ✓ | ✓ |
| Прогресс по ИПР (%, числа) | ✓ | ✓ | ✓ | ✗ |
| Apprоведные proof'ы | ✓ | ✓ | ✓ | ✓ |
| Реджектнутые / pending proof'ы | ✓ | ✓ | ✓ | ✗ |
| Тревожные сигналы | ✗ | ✓ | ✓ | ✗ |
| Время реакции тимлида | ✓ | ✓ | ✗ | ✗ |

### Приватность аналитики
- В `properties` не пишем содержимое заметок и текст комментариев. Только метаданные.
- Sotrudnik не имеет доступа к Grafana — только к продуктовым метрикам в UI.
- Retention событий: 18 месяцев (партишн → DROP), на MVP без ротации.

---

## 3. Двухслойная habit-модель и Telegram-flow

### Слой 1 — Daily learning-log

#### State machine
```
Idle → Prompted (cron 19:00 локально)
Prompted → Writing → Logged    (ответил текстом)
Prompted → Skipped              (нажал «Пропустить»)
Prompted → Frozen               (нажал «Заморозить», если есть freezes)
Logged → Idle (через сутки)
```

#### Контракт сообщения от бота
```
🌒 Что нового узнал сегодня?

Одна строчка — этого достаточно. Если есть ссылка/скрин, прикрепи.

🔥 Стрик: 4 дня
📚 Лог за неделю: 3 заметки

[Записать]  [Пропустить]  [Заморозить (2/2)]
```

- Sotrudnik может ответить текстом без нажатия кнопки — это тоже считается заметкой.
- «Пропустить» — день не активен, стрик не рвётся в выходные/праздники.
- «Заморозить» — пишет `streak_frozen`, лимит 2/месяц.

#### Контракт принятия заметки
```
✓ Записано.

🔥 Стрик: 5 дней (новый рекорд!)
📚 Заметки на неделе:
  пн — Кешировал список юзеров через Redis
  вт — Прочитал статью про consistent hashing
  ...
В воскресенье соберём proof для тимлида из этого лога.
```

#### Cron-расписание
- 19:00 локального времени sotrudnika.
- Не отправляем в выходные и праздники РФ (статический список в коде).
- Если sotrudnik уже отправил заметку сегодня — пинг не отправляется.
- Один пинг в день. Никаких повторных напоминаний.

### Слой 2 — Weekly proof assembly

#### State machine
```
Idle → Prompted (cron, воскресенье 18:00)
Prompted → Picking → Linking → Submitted
Submitted → Approved (lead/trusted ✓)
Submitted → Rejected (lead/trusted ✗ + комментарий)
Rejected → Picking (sam пересобирает)
```

#### Контракт воскресного пинга
```
📦 Воскресенье — время собрать proof.

За неделю у тебя 5 заметок, из них 2 с артефактами:
  📎 ср — consistent hashing (статья + конспект)
  📎 чт — семинар с Артёмом (запись)

[Собрать из этих]  [Выбрать другие]  [Не было движения]
```

- Артефакт обязателен. Без артефакта proof не уходит.
- «Не было движения» — пишется `weekly_assembly_skipped`, тимлиду уходит сигнал-light.

#### Структура proof-карточки
```
{
  author_alias: "Аня",
  team_name: "ML-команда Артёма",
  goal_title: "СИСТЕМНЫЙ ДИЗАЙН — БАЗА",
  proof_text: "...",
  artifacts: [{type, url, label}, ...],
  streak: 5,
  status: "submitted" | "approved" | "rejected",
  approver_role_required: "lead" | "trusted",
  comments: [...],
  submitted_at, approved_at, latency_seconds
}
```

#### Уведомления при `proof_submitted`
- Lead команды → пуш с inline-кнопками `[Одобрить] [Отклонить] [Открыть в web]`.
- Trusted approver(ы) → то же самое.
- Member'ы команды → лёгкий пуш без кнопок апрува, с `[Открыть в ленте]`.
- В web `/feed?tab=team` карточка появляется немедленно.

#### Reject требует комментарий
Reject невозможен без текстового комментария — анти-токсичность мера.

#### Комментарии и советы
- Любой member может ответить советом из Telegram → коммент попадает в web.
- Комментарии **не отправляют push** автору proof'а в Telegram (анти-спам).
- Push на автора уходит **только** при апруве/реджекте.

### Кросс-секционные решения

#### Telegram не привязан
- В web `/me` persistent-баннер «Привяжи Telegram».
- Без привязки daily-log невозможен, стрик не растёт.
- Команда с <50% привязок переходит в fallback-режим (proof через web-форму, без daily-log).

#### Бот лежит / Telegram API упал
- Все пинги через Redis-очередь, retry через 10 минут.
- Если user заблокировал бота → `telegram_blocked=true` → баннер «перепривяжи».

#### Тайм-зоны
- IANA tz хранится per-membership.
- Cron'ы UTC, выборка с конвертацией.
- Стрик считается по локальной полуночи user'а.

---

## 4. AI-персонализация

### Принципы
1. AI помогает, не оценивает.
2. AI не на критическом пути — fallback на шаблон.
3. AI узнаваемо помечен (✨).
4. AI работает с минимально нужным контекстом.
5. Privacy-by-default — содержимое заметок только с opt-in.

### Архитектурный слой

```
backend/internal/personalization/
├── provider.go        — интерфейс PersonalizationProvider
├── kimi.go            — Ollama-совместимый провайдер (Kimi K2 cloud)
├── template.go        — fallback с детерминистскими шаблонами
├── prompts/           — yaml-файлы с промптами по фичам
├── cache.go           — postgres-cache (миграция 16)
├── budget.go          — daily budget tracking
└── service.go         — оркестратор: timeout, fallback, circuit-breaker
```

### Конфигурация (env)

```
OLLAMA_BASE_URL
OLLAMA_API_KEY                                   # секрет, см. env-secrets-hygiene
OLLAMA_MODEL=kimi-k2.6:cloud
PERSONALIZATION_ENABLED=true
PERSONALIZATION_TIMEOUT_PING_MS=2000
PERSONALIZATION_TIMEOUT_ASSEMBLY_MS=8000
PERSONALIZATION_TIMEOUT_BRIEFING_MS=4000
PERSONALIZATION_DAILY_BUDGET_TOKENS=500000
```

### Уровни AI per-команда

| Режим | Что идёт в LLM | Что работает |
|---|---|---|
| `off` | Ничего | Только шаблоны |
| `metadata-only` | Алиасы, длины, типы артефактов, заголовки целей | Briefing (упрощённый), пинг (умеренная персонализация) |
| `full` | Содержимое заметок и proof'ов | Все три фичи на полную |

**Default = `metadata-only`.** `full` требует opt-in на уровне команды И индивидуального согласия каждого sotrudnika (`team_memberships.ai_consent`).

### Фича 1 — Персональный вечерний пинг

**Контекст:** имя, последние 3–5 заметок (только в `full`), стрик, активные ИПР, день недели.

**Validation:**
- Длина: 8–25 слов.
- Не содержит запрещённых слов («молодец», «отлично», «плохо», «ты должен»).
- Заканчивается вопросом.

**Latency:** генерится **за 30 минут до отправки** (cron-batch). Если AI не успел за 2с — fallback-шаблон. Sotrudnik не замечает.

### Фича 2 — AI-сборка proof из лога

Самая мощная фича для habit. Снимает главное трение weekly-слоя.

**Когда вызывается:** sotrudnik нажал `[Собрать proof]` → бот пишет `✍️ Собираю...` → ждёт ≤8с.

**Контекст:** все заметки за неделю, ИПР-цели, последние апрувленные proof'ы (только `full`).

**Output:**
```json
{
  "candidates": [
    {
      "ipr_goal_id": 42,
      "note_ids": [101, 103, 104],
      "rationale": "...",
      "confidence": "high" | "low"
    }
  ]
}
```

**UI:**
```
✨ Я бы собрал так:

📦 PROOF 1 — «Системный дизайн»  [рекомендую]
  📎 ср — consistent hashing
  📎 чт — семинар с Артёмом

[Отправить PROOF 1]  [Поправить]  [Сделать вручную]
```

**Validation:** `note_ids` принадлежат user'у, `ipr_goal_id` активен, артефакты есть в `high`-confidence.

**Метрика качества:** `personalization_accepted{feature=assemble}` ≥ 60%.

### Фича 3 — Briefing тимлиду

**Когда:** тимлид открыл `/teams/:id/members/:userId`.

**UI:**
```
✨ За 2 недели:

Глубоко разобрала distributed systems — 3 proof'а, тимлид
одобрил все. Артефакты сильные.

⚠ По второму пункту ИПР («Системы мониторинга») — ни одного
proof'а за 14 дней. Возможно, поднять на 1:1.

[Скрыть briefing] [Сообщить, что бесполезно]
```

**Контекст:** последние 30 дней доменных событий, относящихся к sotrudniku в этой команде (proof'ы submit/approve/reject, апрув-latency, стрик), плюс содержимое только **апрувленных** proof'ов. **Daily-log content не уходит в LLM**, даже в `full`-режиме — только агрегаты (число заметок, дни активности).

**Validation:** 30–80 слов, ровно одна `⚠`-секция или явное «Без рисков», без оценочных суждений.

**Cache:** 6 часов на пару (team, member). Инвалидация на `proof_submitted`/`proof_approved`.

### Анти-зависимость от LLM

Каждая фича имеет детерминистский TemplateProvider:
- `EveningPing` — ротация из 8 шаблонов по дню недели/числа.
- `AssembleProof` — эвристика: группирует заметки с артефактами по дате близости.
- `LeadBriefing` — статистическая сводка без анализа.

Продукт работает без AI вообще. AI — улучшение, не зависимость.

### Промпт-каталог и версионирование

`backend/internal/personalization/prompts/*.yaml` со схемой:
```yaml
version: 1
feature: evening_ping
model: kimi-k2.6:cloud
system: "Ты пишешь..."
user_template: "Контекст: ..."
validation:
  min_words: 8
  max_words: 25
  must_end_with: "?"
  forbidden_substrings: ["молодец", "отлично", "плохо"]
```

В метриках `personalization_invoked` пишет `prompt_version`.

### Circuit breaker

Если за последний час доля fallback по `timeout` или `validation` > 30% по любой фиче — авто-отключение фичи на час, переход на template, алерт в Grafana.

### Privacy
- `ai_consent` per-membership.
- Содержимое не передаётся в `metadata-only`.
- AI-ответы помечаются ✨.
- Никаких PII в промптах — только алиасы и id.

---

## 5. Модель данных и миграции

### Принципы
1. Расширяем, не пересобираем существующие сущности.
2. `team` ≠ `circle` — отдельная сущность, отдельные роли.
3. Каждая миграция reversible (`goose Up`/`Down`).
4. Все новые поля nullable или с дефолтами.
5. Индексы — по реальным запросам.

### Карта новых миграций

```
00011_teams.sql                  — teams + team_memberships
00012_team_extensions.sql        — goals/check_ins.team_id, role-aware approval
00013_daily_log.sql              — daily_log_entries
00014_user_streak.sql            — user_streak
00015_analytics_events.sql       — partitioned event store
00016_personalization.sql        — ai_personalization_cache + ai_daily_budget
```

### Миграция 11 — `teams` + `team_memberships`

```sql
CREATE TABLE teams (
    id BIGSERIAL PRIMARY KEY,
    lead_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    invite_code TEXT NOT NULL UNIQUE,
    member_limit INTEGER NOT NULL DEFAULT 25,
    ai_mode TEXT NOT NULL DEFAULT 'metadata-only'
        CHECK (ai_mode IN ('off', 'metadata-only', 'full')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ
);

CREATE TABLE team_memberships (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('lead', 'trusted_approver', 'member')),
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'left', 'removed')),
    ai_consent BOOLEAN NOT NULL DEFAULT FALSE,
    timezone TEXT NOT NULL DEFAULT 'Europe/Moscow',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    UNIQUE (team_id, user_id)
);

-- Ровно один lead на команду
CREATE UNIQUE INDEX uq_team_one_lead ON team_memberships(team_id)
    WHERE role = 'lead' AND status = 'active';
```

**Инварианты:**
- `lead_user_id ON DELETE RESTRICT` — нельзя удалить пользователя, лидящего команду.
- Ровно один `lead` обеспечен partial unique index.
- `ai_consent` индивидуальный — даже в `full`-режиме команды, без consent работает как `metadata-only`.

### Миграция 12 — расширение goals/check_ins

```sql
ALTER TABLE goals
    ADD COLUMN team_id BIGINT REFERENCES teams(id) ON DELETE SET NULL,
    ADD CONSTRAINT goal_circle_or_team_xor
    CHECK (
        (circle_id IS NULL AND team_id IS NULL) OR
        (circle_id IS NOT NULL AND team_id IS NULL) OR
        (circle_id IS NULL AND team_id IS NOT NULL)
    );

ALTER TABLE check_ins
    ADD COLUMN team_id BIGINT REFERENCES teams(id) ON DELETE SET NULL,
    ADD COLUMN approver_role TEXT
        CHECK (approver_role IS NULL OR approver_role IN ('lead', 'trusted_approver'));
```

**XOR-инвариант:** цель не может одновременно быть в circle и в team. Существующие данные backwards-compatible — все `team_id = NULL`.

### Миграция 13 — `daily_log_entries`

```sql
CREATE TABLE daily_log_entries (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    log_date DATE NOT NULL,           -- локальная дата sotrudnika
    text_content TEXT NOT NULL,
    has_artifact BOOLEAN NOT NULL DEFAULT FALSE,
    external_url TEXT,
    storage_key TEXT,
    mime_type TEXT,
    file_size_bytes BIGINT,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    consumed_in_check_in_id BIGINT REFERENCES check_ins(id) ON DELETE SET NULL,
    consumed_at TIMESTAMPTZ,
    UNIQUE (user_id, team_id, log_date)
);
```

**Инварианты:**
- Одна запись в день per-team. Повтор → overwrite.
- `log_date` как `DATE` — стрик считается по дням.
- `consumed_in_check_in_id` — ссылка на proof, в который заметка вошла. Заметка не удаляется при consume.

### Миграция 14 — `user_streak`

```sql
CREATE TABLE user_streak (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    current_streak INTEGER NOT NULL DEFAULT 0,
    best_streak INTEGER NOT NULL DEFAULT 0,
    last_active_date DATE,
    freezes_used_this_month SMALLINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, team_id)
);
```

**Per-team стрик** — sotrudnik в нескольких командах имеет независимые стрики.

Денормализованные `current_streak`/`best_streak` обновляются в той же транзакции, что `daily_log_entries.INSERT`.

### Миграция 15 — `analytics_events` (партиционированная)

```sql
CREATE TABLE analytics_events (
    id BIGSERIAL,
    ts TIMESTAMPTZ NOT NULL,
    event_name TEXT NOT NULL,
    user_id BIGINT,
    team_id BIGINT,
    goal_id BIGINT,
    proof_id BIGINT,
    source TEXT NOT NULL CHECK (source IN ('web', 'telegram_bot', 'cron', 'system')),
    properties JSONB NOT NULL DEFAULT '{}',
    PRIMARY KEY (id, ts)
) PARTITION BY RANGE (ts);

-- Партиции на 6 месяцев вперёд
CREATE TABLE analytics_events_2026_05 PARTITION OF analytics_events
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
-- ...
```

- Closed-list `event_name` проверяется в коде (Go const + CI), не в БД CHECK.
- Никаких FK от `analytics_events` к другим таблицам — данные переживают удаление user'а.

### Миграция 16 — Personalization

```sql
CREATE TABLE ai_personalization_cache (
    cache_key TEXT PRIMARY KEY,
    feature TEXT NOT NULL,
    payload JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ai_daily_budget (
    budget_date DATE PRIMARY KEY,
    tokens_used BIGINT NOT NULL DEFAULT 0,
    requests_count INTEGER NOT NULL DEFAULT 0,
    fallback_count INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Open question (решённый)
**Точечные приглашения по email — не на MVP.** Используем shared-link через `teams.invite_code`.

---

## 6. Архитектура и deployment

### Принципы
1. Не плодим новые процессы (`api` + `worker` достаточно).
2. Domain-by-package в `internal/`.
3. Транзакционная outbox-модель уже есть (`domain_events`).
4. Telegram-callbacks через `domain_events` + `worker`, не из HTTP-хендлера.
5. AI-вызовы асинхронные, кроме AI-сборки proof (синхронно с timeout 8с).
6. Никаких микросервисов.

### Топология процессов

```
nginx
  ├── /api/* → cmd/api  (Go: REST + Telegram webhook)
  └── /*    → Next.js web

cmd/worker (Go, no HTTP)
  ├── outbox-processor
  ├── evening-ping cron
  ├── weekly-assembly-ping cron
  ├── streak-monthly-reset
  ├── streak-day-rollover
  ├── partition-creator
  ├── personalization-prewarm
  └── circuit-breaker

Postgres (data + analytics + outbox)
Redis (queue, bot state, cache)
MinIO (files, evidence, log attachments)
Ollama API (Kimi K2 cloud)
```

### Backend-пакеты

```
backend/internal/
├── teams/            ★ новый — team, membership, authz
├── daily_log/        ★ новый — entries, validation
├── streak/           ★ новый — rules (пн-пт + freezes)
├── personalization/  ★ новый — provider + service
├── analytics/        ★ новый — Recorder, EventNames const
├── checkins/         ↑ team-aware approval, role-check, team feed
├── goals/            ↑ team_id, XOR validation
├── telegram/         ↑ handlers/{daily_log, weekly_proof, approve, stats}
├── notifications/    ↑ новые типы
├── circles/          — без изменений (personal продолжает работать)
├── recaps/           — без изменений на MVP
├── users/            ↑ /metrics/self endpoint
└── platform/         — без изменений
```

### Межпакетные интерфейсы

- `analytics.Recorder` — единая точка записи событий.
- `streak.Calculator` — `daily_log/service` зовёт при submit.
- `personalization.Service` — `telegram/handlers` зовёт для AI-фич.
- `notifications.Sender` — единая точка отправки.
- `teams.Authorizer` — единая точка проверки прав.

### Транзакционная гарантия

```go
func (s *Service) SubmitLog(ctx, req) error {
    return s.db.InTx(ctx, func(tx) error {
        entry, _ := s.repo.Insert(tx, req)
        s.streakCalc.UpdateOnSubmit(tx, ...)
        s.analytics.Record(tx, "daily_log_submitted", ...)
        s.events.Publish(tx, "daily_log.submitted", entry)
        return nil
    })
}
```

Если что-то падает — вся транзакция откатывается. Метрики никогда не врут.

### Worker — расписание

| Поток | Расписание | Параллелизм |
|---|---|---|
| outbox-processor | каждые 2с | 4 воркера |
| evening-ping batch | tick каждые 5 мин | 1 |
| weekly-assembly-ping | tick каждые 5 мин | 1 |
| streak-monthly-reset | cron 1 0 * * | 1 |
| streak-day-rollover | tick каждые 10 мин (filter by tz) | 1 |
| partition-creator | cron 0 3 25 * * | 1 |
| materialized-refresh | cron 17 * * * * | 1 |
| personalization-prewarm | cron 23 * * * * | 1 |
| circuit-breaker | tick каждые 5 мин | 1 |

Идемпотентность через `dedup_key` — реплицируемо безопасно. На MVP — один worker.

### Telegram bot

- Встроен в `cmd/api` как webhook `POST /telegram/webhook/:secret`.
- `:secret` — UUID в URL.
- State диалога — Redis (`tg:state:<chat_id>`, TTL 1ч).
- Outgoing — через `notifications` пакет (outbox).
- Webhook отвечает Telegram'у в <100ms (без AI и без внешних вызовов).
- Идемпотентность через `update_id`.

### Web

- `/feed` получает 3-й tab «КОМАНДА» рядом с «МОЙ КРУГ» / «ПОХОЖИЕ КРУГИ».
- Новые страницы: `/teams`, `/teams/:id`, `/teams/:id/members/:userId`.
- В `/me`: «Команды» + per-team настройки таймзоны и AI-consent.
- Привязка Telegram — расширение `/me/integrations`.
- Без новых dependency.

### Deployment

#### Compose добавки
```yaml
grafana:
  image: grafana/grafana:11.2.0
  environment: [GF_SECURITY_ADMIN_USER, GF_SECURITY_ADMIN_PASSWORD, ...]
  volumes: [grafana-data, ./grafana/provisioning]
  ports: ["127.0.0.1:3001:3000"]   # только nginx
```

Доступ — через nginx под `/grafana/` с basic-auth + IP-allowlist.

#### Новые env (см. также `env-secrets-hygiene` skill)
```
OLLAMA_BASE_URL, OLLAMA_API_KEY, OLLAMA_MODEL
PERSONALIZATION_ENABLED, PERSONALIZATION_TIMEOUT_*, PERSONALIZATION_DAILY_BUDGET_TOKENS
TELEGRAM_WEBHOOK_SECRET
GRAFANA_ADMIN_USER, GRAFANA_ADMIN_PASSWORD
```

#### CI/CD добавки
- `migrate up && migrate down && migrate up` — reversible.
- `lint analytics-events` — все вызовы используют names из `analytics.EventNames`.
- `test telegram-handlers` — стейт-машина диалога.

#### Observability
- Логи: JSON через slog, nginx → файл.
- `personalization` — INFO с latency, WARN на fallback, ERROR на budget overrun.
- `analytics.record` — DEBUG.
- Метрики инфры — отдельный поток (`observability-builder`).

### Open question (решённый)
**State Telegram-диалога — Redis**, TTL 1ч.

---

## 7. Контракты API и бота, обработка ошибок

### Принципы
1. REST-first, JSON, http-статусы.
2. Идемпотентность для всех write — `Idempotency-Key` или естественный ключ.
3. Versioning через path (`/api/v1/...`).
4. Coarse-grained team-эндпоинты (feed одним запросом).
5. Errors как структуры с `code` (закрытый список).
6. Bot — отдельная зона.

### Форма REST-ответов

**Успех:** `{ "data": {...}, "meta": {...} }`
**Ошибка:** `{ "error": { "code", "message", "details" }, "meta": {...} }`

### Закрытый список `error.code`

```
team.not_found / not_member / not_lead / full / archived /
  already_member / must_have_lead / cannot_demote_self /
  cannot_remove_self_as_lead / cannot_leave_as_only_lead /
  cannot_change_others_consent
invite.invalid / expired
daily_log.text_too_short / privacy_denied
proof.no_artifact / entries_already_consumed / goal_not_in_team /
  not_pending / cannot_approve_self / no_approve_rights /
  reject_comment_required / not_in_team
comment.empty
telegram.not_linked
analytics.event_not_allowed_from_frontend
validation.field_invalid (generic, with details)
user.not_authenticated / forbidden
rate_limit.exceeded
internal.unexpected
```

### REST-эндпоинты — основные

#### Teams
- `POST /api/v1/teams` — создать.
- `GET /api/v1/teams` — список своих.
- `GET /api/v1/teams/:id` — детали + `my_role`.
- `PATCH /api/v1/teams/:id` — `name`, `ai_mode` (lead-only).
- `POST /api/v1/teams/:id/archive` (lead-only).
- `POST /api/v1/teams/:id/regenerate-invite`.

#### Membership
- `POST /api/v1/teams/join` — body `{invite_code}`.
- `POST /api/v1/teams/:id/members/:userId/role` — изменить роль (lead-only, не себя).
- `DELETE /api/v1/teams/:id/members/:userId` — удалить (lead-only).
- `POST /api/v1/teams/:id/leave`.
- `POST /api/v1/teams/:id/members/:userId/ai-consent` — только сам.

#### Daily log
- `GET /api/v1/teams/:teamId/daily-log?from=&to=&user_id=` — privacy-aware.
- `POST /api/v1/teams/:teamId/daily-log` — body с `log_date` (локальная дата). Повтор = overwrite.

#### Weekly proof
- `POST /api/v1/teams/:teamId/proofs` — body `{goal_id, daily_log_entry_ids, ...}`. Артефакт обязателен.
- `GET /api/v1/teams/:teamId/feed?cursor=&limit=`.
- `POST /api/v1/proofs/:id/approve` (lead/trusted, не свой).
- `POST /api/v1/proofs/:id/reject` — body с `comment` (обязателен).
- `POST /api/v1/proofs/:id/comments`.

#### Per-user metrics
- `GET /api/v1/users/:userId/metrics/self` — только сам.
- `GET /api/v1/teams/:teamId/members/:userId/full` — lead/trusted.
- `GET /api/v1/teams/:teamId/members/:userId/peer` — member.

#### Personalization
- `POST /api/v1/teams/:teamId/proofs/assemble-suggestion` — timeout 8с, fallback heuristic.
- `GET /api/v1/teams/:teamId/members/:userId/briefing` — cache 6ч, async refresh.

#### Telegram
- `POST /api/v1/me/telegram/link-token` → `{token, expires_at, deeplink}`.
- `DELETE /api/v1/me/telegram/link`.

#### Analytics
- `POST /api/v1/analytics/event` — только UI-only события из whitelist.

### Bot-контракты

#### Команды
- `/start`, `/start <token>` — приветствие, linking.
- `/stats` — свои метрики.
- `/stats @username` — метрики члена команды (lead/trusted only).
- `/teams` — переключение активной команды.
- `/freeze` — заморозить день.
- `/help`.

#### Inline-кнопки (callback_data prefixes)
```
log:write / log:skip / log:freeze
proof:assemble
proof:assemble:auto:<idx>
proof:approve:<id>
proof:reject:<id>
proof:comment:<id>
team:select:<id>
briefing:hide / briefing:report
```

#### Состояния диалога (Redis)
`AwaitingDailyLog` | `AwaitingRejectComment` | `AwaitingProofComment`

Любая команда `/...` сбрасывает state.

### Обработка ошибок

**Принципы:**
1. Никогда не показываем сырой 500. Только `error.code` или `internal.unexpected` с `request_id`.
2. Логируем с `request_id`.
3. Ошибки не падают молча — фейлим транзакцию целиком.
4. Bot-ошибки дружелюбные (не `proof.no_artifact`, а «Без ссылки/файла proof не уйдёт. Добавь артефакт следующим сообщением.»).

**Маппинг error → user message** — в yaml-файле, локализуемо.

**Telegram-specific:**
- 429 → exponential backoff в worker.
- 403 (blocked) → `telegram_links.status='blocked'`, баннер в web.
- Webhook timeout → Telegram retry, идемпотентность по `update_id`.

**AI-specific:**
- Timeout → fallback + `personalization_fallback{reason=timeout}`.
- Validation fail → fallback, **без retry** (мусор не лечится повторами).
- 5xx → retry через 2 мин в worker.
- Daily budget exceeded → авто-template до конца суток + Grafana-алерт.

### Идемпотентность

| Операция | Ключ | Поведение |
|---|---|---|
| daily_log POST | (user_id, team_id, log_date) | Overwrite |
| proof submit | Idempotency-Key (uuid) | Дедуп 24ч |
| proof approve/reject | Idempotency-Key | Если уже approved → 200 с текущим |
| team join | (invite_code, user_id) | Если уже member → 200 |
| team leave | Idempotency-Key | Если уже left → 200 |
| notification send | dedup_key | Не отправляем повторно |

`Idempotency-Key` хранится в Redis (`idem:<key>`) 24ч.

### Open questions (решённые/отложенные)
1. **Real-time feed:** polling 30с на MVP, SSE/WebSocket позже.
2. **Rate limiting:** минимальный (60 RPM per IP per endpoint group).
3. **Webhook'и наружу:** не закладываем точку расширения сейчас.

---

## 8. Стратегия тестирования

### Принципы
1. TDD для domain-логики (по проектной политике).
2. Пирамида: много unit, средне integration, мало e2e.
3. AI и Telegram за интерфейсом — `TemplateProvider`/`FakeTelegramClient` в CI.
4. Каждый bug → regression test (`bug-reproducer`).

### Карта

```
E2E (Playwright)        — 12 сценариев
  ↓
Integration (Go)        — ~80 тестов
  ↓
Contract (Go)           — ~30 тестов
  ↓
Unit (Go + TS)          — 300+ тестов
```

### Unit — TDD-обязательные

#### `streak/rules_test.go` (≥85% coverage)
- FirstSubmissionStartsAtOne, ConsecutiveDaysIncrement.
- WeekendNotCounted, RussianHolidayNotCounted.
- GapOnWorkdayBreaks.
- Freeze: PreservesStreak, TwoFreezesPerMonthLimit, BeyondLimitRefused.
- MultipleSubmissionsSameDay_Idempotent, OverwriteEntry_NoStreakChange.
- Timezone_DayBoundaryRespected.
- BestStreak: UpdatesOnNewRecord, NotDecreasedOnReset.
- NewMonthResetsFreezeCounter.
- PerTeamIsolated.

#### `teams/authz_test.go` (≥85%)
Таблица решений из §2 в виде кода:
- Lead/Trusted: CanApproveAnyProof, CannotApproveOwn.
- Member: CannotApprove, AnyMember CanComment.
- NonMember: CannotSeeAnything.
- Metrics visibility (full/peer/self).

#### `goals/validation_test.go`
- XOR_CircleAndTeamCannotBoth / NeitherIsAllowed.

#### `daily_log/validation_test.go`
- RejectShorterThan10Chars (мягкий перeспрос).
- AcceptOnSecondTry, OverwriteSameDay_HistoryPreserved.
- LogDateMustBeLocalTodayOrYesterday.
- HasArtifact derivation, FileSizeLimit.

#### `personalization/validation_test.go` (≥80%)
- EveningPing: MinMaxWords, ForbiddenWords, RequiresQuestionMark.
- AssembleProof: RejectsNoArtifactInHigh, RejectsGoalNotInTeam, RejectsEntriesNotOwnedByUser.
- Briefing: ValidatesLength, ContainsExactlyOneAlert.

#### `analytics/event_name_test.go`
- AllUsedNamesAreInClosedList (CI walks codebase).

### Integration

- `daily_log_integration_test.go` — Submit, Overwrite, TransactionRollback, RaceConditions.
- `weekly_proof_integration_test.go` — SubmitWithEntries, Approve+Latency, RejectRequiresComment, Resubmit, TwoApproversInRace.
- `telegram_webhook_integration_test.go` — Linking, DailyLogText, ApprovalCallback (Lead/Member), RejectFlow, DuplicateUpdate, RateLimit, BlockedBot.
- `worker_cron_integration_test.go` — EveningPing (only linked, tz, weekend), StreakDayRollover, PartitionCreator, CircuitBreaker.
- `migrations_test.go` — UpDownUp_Clean, DataPreservation, XOR/OneLead enforced.
- `api_contract_test.go` — все эндпоинты, payload-snapshots, error-cases.

### AI Contract Tests (nightly only, с `OLLAMA_API_KEY`)

```go
TestKimiK2_EveningPing_Live_PassesValidation
TestKimiK2_AssembleProof_Live_ReturnsValidCandidates
TestKimiK2_Briefing_Live_PassesValidation
TestKimiK2_LatencyP90_Under5s_EveningPing
TestKimiK2_LatencyP90_Under8s_Assemble
```

Падают — не блокирует прод (template fallback). Создают инцидент.

### E2E (Playwright)

```
e2e/
├── happy-path/ — 5 сценариев главных user journey
├── critical-paths/ — 4 сценария (reject+resubmit, streak break, AI flow, trusted demote)
└── privacy/ — 3 сценария (member cannot see others, AI metadata leak, public proofs)
```

В CI — `FakeTelegramClient`. В nightly — реальный тестовый бот.

### Performance Smoke (`performance-smoke-tester`)

- `daily-log POST` p95 < 200ms (без AI).
- `assemble-suggestion` template p95 < 100ms, Kimi p95 < 8000ms.
- `team feed` p95 < 300ms (с 1000 proofs).
- `member full` p95 < 500ms.
- Telegram webhook ack < 100ms.

### Security & Privacy

`security-reviewer` + `privacy-reviewer` — отдельные тест-батареи:

```go
TestSecurity_NonMember_CannotReadFeed
TestSecurity_Member_CannotEscalateRoleViaPATCH
TestSecurity_CrossTeam_GoalCannotBeStolen
TestSecurity_AI_MetadataMode_DoesNotLeakLogContent  // ★ критический
TestSecurity_PromptInjection_DoesNotChangeBriefingPolicy
TestSecurity_Telegram_WebhookSecretRequired

TestPrivacy_DailyLogContent_NotInLeadFullMetrics
TestPrivacy_DailyLogContent_NotInPeerView
TestPrivacy_AnalyticsEvents_DoNotContainNoteText
TestPrivacy_AILogs_DoNotContainNoteText_InMetadataMode
TestPrivacy_GrafanaDashboard_DoesNotShowRawNoteText
TestPrivacy_DeletedUser_LogsBecomeOrphaned_NotPropagated
```

### Coverage thresholds

- `streak/`, `teams/`, `daily_log/` — ≥85%.
- `personalization/` — ≥80% (без `kimi.go`).
- `checkins/` (расширения) — ≥75%.
- Алерт при просадке >5% в PR.

### Quality gates перед merge

`quality-gate-runner` прогоняет:
1. Unit (полные).
2. Integration (полные).
3. E2E happy-path.
4. Security & privacy батареи.
5. Migrations Up/Down/Up.
6. Performance smoke.
7. Lint + typecheck.

### Open questions (решённые)
1. **E2E Telegram:** Fake в CI + Real в nightly.
2. **Coverage tooling:** Go встроенный, без codecov на MVP.
3. **Property-based testing:** для streak имеет смысл, рассматриваем при реализации.

---

## 9. Release plan

### Принципы
1. Релиз — четыре фазы, каждая самостоятельно полезна.
2. Каждая фаза имеет success criterion.
3. AI и команды разделены — сначала команды без AI, потом AI-слой.
4. Alpha-команда из своих 3–5 человек, потом расширение.
5. Канареечный rollout AI-фич (по одной).

### Фаза 0 — Foundation (~1 нед)
- Миграции 11–12.
- `internal/teams/` пакет, HTTP-эндпоинты `/api/v1/teams`.
- Web: skeleton `/teams`, `/teams/:id`.
- `analytics/` пакет (миграция 15) пустой.

**Success:**
- Тимлид создаёт команду, sotrudnik присоединяется по invite.
- Тесты team authz зелёные.
- Migrations up/down/up чистые.

**Риск:** XOR-инвариант → ручной dry-run в staging.

### Фаза 1 — Habit-loop без AI (~1.5 нед)
- Миграции 13–14.
- `daily_log/` + `streak/` (TDD-first).
- Telegram: linking, daily-log приём, вечерний пинг (template), стрик-эхо.
- Worker: evening-ping, streak-day-rollover, streak-monthly-reset.
- Web: `/me` со стриком и heatmap.

**Success (alpha 3–5 чел, 2 нед):**
- ≥4/5 пишут лог ≥4 дней/нед на 2-й нед.
- Стрики сходятся.
- 0 privacy-инцидентов.

**Не двигаемся к фазе 2, если success не выполнен.**

**Риски:** streak-баг (TDD + fake clock), пинг не в то время (явная tz в onboarding), Telegram blocked (баннер + алерт лиду).

### Фаза 2 — Weekly proof в команде (~1.5 нед)
- Расширение `checkins/` — team-aware.
- Telegram: воскресный пинг (template), сборка через чекбоксы, апрув одной кнопкой, reject + comment.
- Web: `/feed?tab=team`, `/teams/:id/members/:userId/full` (template-briefing), `/peer`.
- Per-user metrics endpoints.

**Success (alpha, ещё 2 нед):**
- ≥70% proof'ов одобрены за 48ч.
- Тимлид заходит в feed ≥3 раз/нед.
- ≥50% sotrudnikov собрали ≥1 proof за 2 нед.

**Риски:** тимлид не реагирует (эскалация-пуш + onboarding), sotrudnik не находит как собрать (копирайт-проверка).

### Фаза 3 — AI-слой (~1 нед)

Канареечный rollout:

**3.1 AI evening_ping** (1–2 дня) — alpha only. Метрика: `daily_log_submitted` rate +10–15%.
**3.2 AI assemble_proof** (2–3 дня) — alpha only. Метрика: acceptance ≥60%, время сборки сокращается.
**3.3 AI lead_briefing** (1–2 дня) — лиды alpha. Метрика: «бесполезно» <10%.
**3.4** Расширение на 1–2 партнёрские команды.

**Success фазы 3:**
- Все три AI-фичи в `full` для alpha неделю без circuit-breaker.
- Acceptance assemble ≥60%.
- Daily budget не превышен.
- 0 privacy-инцидентов.

**Риски:** Kimi мусорит (validators + circuit-breaker), бюджет (алерт на 80%), promp-injection (privacy-тесты + ручной аудит), стоимость (пересмотр после 3.4).

### Фаза 4 — Production polish (~3-4 дня)
- Grafana 3 дашборда.
- Materialized view (если нужно).
- Performance smoke на staging-seed (50 команд × 25 чел × 26 нед).
- `quality-gate-runner` зелёный.
- `security-review`, `privacy-reviewer` пройдены.
- `production-deploy-ops` runbook обновлён.
- `decision-log-writer`, `agent-handoff-writer`.

**Success:** дашборды живые, smoke зелёный, `launch-checklist-runner` пройден.

### Сводный календарь

| Фаза | Длительность | Итог |
|---|---|---|
| 0 — Foundation | ~1 нед | teams работают |
| 1 — Habit без AI | ~1.5 нед | daily-log + стрик |
| 2 — Team proof | ~1.5 нед | weekly proof + апрув |
| 3 — AI-слой | ~1 нед | три AI-фичи |
| 4 — Polish | ~3-4 дня | готовность к расширению |
| **Итого** | **~5 нед** | MVP для 5–10 команд |

### Risk-radar (top-5)

| Риск | Импакт | Митигация |
|---|---|---|
| Sotrudniki не пишут daily-log | Критический | Alpha 2 нед раньше, право на pivot |
| Тимлид не апрувит вовремя | Высокий | Эскалация-пуши + onboarding тимлидов |
| Streak-баг ломает доверие | Высокий | TDD + property-based + alpha-валидация |
| AI выдаёт чушь / privacy leak | Высокий | Validators + circuit breaker + nightly + manual audit |
| Стоимость AI на масштабе | Средний | Daily budget + per-feature acceptance → отключим briefing |

### Отсечения скоупа (если припрёт)

В порядке "сначала что отрежем":
1. AI briefing.
2. AI assemble.
3. AI evening_ping.
4. Per-user metrics для коллег.
5. Trusted approver роль.
6. Heatmap.

**Нельзя отрезать:** daily-log + стрик + weekly proof + lead-approval.

### Definition of Done (per phase)

- [ ] Unit + integration зелёные.
- [ ] Coverage пороги.
- [ ] E2E happy-path для фазы.
- [ ] Migrations up/down/up.
- [ ] `security-review` пройден.
- [ ] `privacy-reviewer` пройден.
- [ ] Alpha-команда отработала ≥1 неделю без блокирующих жалоб.
- [ ] Документация обновлена.

### Backlog после MVP

- Точечные приглашения по email.
- Голосовые заметки.
- Multi-team dashboard для тимлидов с несколькими командами.
- AI-feedback на отдельную заметку.
- SSE/WebSocket для real-time feed.
- Иерархия команд (тимлид-тимлида).
- Public sharable proof-cards.
- Лента команды как утренний кофе (якорь №4).

---

## Приложение A. Безопасность секретов

`OLLAMA_API_KEY`, `TELEGRAM_WEBHOOK_SECRET`, `GRAFANA_ADMIN_PASSWORD` — sensitive. Поток:

1. Хранятся в GitHub Actions Secrets.
2. Деплой инжектит в `.env.prod` на сервере (`production-deploy-ops` skill).
3. В коде/спеках — только имена, никогда значения.
4. `.env` в `.gitignore`, `.env.example` без значений.
5. Ротация при компрометации — отзыв на стороне провайдера + новый secret в Actions.
6. См. `env-secrets-hygiene` skill для процедур.

## Приложение B. Сводка проектных решений из брейншторма

| # | Решение | Выбор |
|---|---|---|
| 1 | Сценарий | Корпоративный (А) |
| 2 | Якорь привычки | Статус сотрудника (№2) |
| 3 | Точка входа | Telegram |
| 4 | Награда после proof | Бот мгновенно + тимлид-как-buddy асинхронно |
| 5 | Тип proof | Только структурированные артефакты |
| 6 | Частота | Daily learning-log → weekly proof |
| 7 | Daily touch-point | Learning-log (вариант А) |
| 8 | Skip-day policy | Только пн–пт |
| 9 | Видимость в команде | Все участники видят всех |
| 10 | MVP-срез | Полный двухслойный flow |
| 11 | Команда vs круг | Новая отдельная сущность `team` |
| 12 | Trusted approver scope | Апрувит всех в команде |
| 13 | Multi-membership | Sotrudnik может быть в personal-круге и в team |
| 14 | Telegram-привязка | MVP — ручная через web |
| 15 | AI-фичи | Полный пакет: пинг, сборка, briefing |
| 16 | LLM | Kimi K2 (cloud) через Ollama |
| 17 | Default AI mode | `metadata-only` |
| 18 | Bot state storage | Redis (TTL 1ч) |
| 19 | Team invites | Shared-link (не email) |
| 20 | Real-time feed | Polling 30с |
| 21 | E2E Telegram | Fake в CI, real в nightly |

## Приложение C. Глоссарий

- **ИПР** — индивидуальный план развития сотрудника.
- **Proof** — еженедельное доказательство движения по ИПР с обязательным артефактом.
- **Daily learning-log** — ежедневная заметка sotrudnika в Telegram про обучение.
- **Стрик (streak)** — последовательность дней с заметками (только пн–пт).
- **Lead** — тимлид команды, owner.
- **Trusted approver** — sotrudnik с делегированным правом апрува.
- **Member** — обычный участник команды.
- **Circle (круг)** — существующая personal-сущность ProofForge, не путать с team.
- **Team (команда)** — новая корпоративная сущность.
- **Briefing** — AI-сводка для тимлида о sotrudnike перед 1:1.
- **Habit loop** — trigger → action → reward.
- **Anchor (якорь)** — существующий ритуал, на который продукт паразитирует.
