# ProofForge Teams Phase 1 — Daily Learning-Log и Streak Foundation

**Дата:** 2026-05-07
**Статус:** design spec для реализации после Phase 0
**Фаза:** 1 из 4 после foundation
**Исходный контекст:** `2026-05-07-proofforge-teams-habit-loop-design.md`, Phase 0 plan

---

## Goal

Построить первый рабочий habit-loop для Team mode: sotrudnik каждый рабочий день пишет приватную learning-log заметку в Telegram, получает мгновенный feedback от бота и видит per-team streak. Это не weekly proof и не team feed: Phase 1 создает регулярный ежедневный вход, из которого Phase 2 соберет доказательства.

Успех фазы: alpha-команда 3-5 человек две недели пишет лог без privacy-инцидентов, и минимум 4 из 5 участников на второй неделе имеют 4+ logged рабочих дня.

## Non-Goals

- Нет weekly proof assembly, lead/trusted approval и team feed.
- Нет AI-персонализации; все тексты deterministic template.
- Нет лидербордов, бейджей, общего рейтинга, публичного сравнения людей.
- Нет показа сырого daily-log тимлиду, trusted или коллегам.
- Нет HR-интеграций, SSO, email-invites, Slack.
- Нет попытки чинить personal circles или legacy `goals.current_streak_count`.

## Dependencies on Previous Phases

- Phase 0 должен предоставить `teams`, `team_memberships`, роли `lead`, `trusted_approver`, `member`, invite-flow и `team_memberships.timezone`.
- Phase 0 должен оставить `team_memberships.ai_consent` в модели, но Phase 1 не использует AI.
- Team-bound goals уже существуют через `goals.team_id`; Phase 1 может читать заголовки активных team-goals только для собственного `/me` и Telegram context.
- Если Phase 0 API skeleton меняется, Phase 1 сохраняет product invariant: daily-log всегда scoped by `(user_id, team_id, local_date)`.

## Domain Model

### Core nouns

- **DailyLogEntry** — приватная заметка sotrudnika за локальную дату внутри одной team-membership.
- **StreakState** — per-user/per-team состояние streak: current, best, last_active_date, freezes_used_this_month.
- **DailyPrompt** — scheduled bot prompt для конкретной membership и локальной даты.
- **Freeze** — осознанное сохранение streak в рабочий день без заметки, максимум 2 раза в календарный месяц.
- **WorkingDayPolicy** — календарная политика: понедельник-пятница, российские праздники не считаются активными днями.

### State machine

```text
Idle
  -> Prompted      cron выбрал membership на 19:00 local time
  -> Logged        пользователь прислал текст/вложение до local midnight
  -> Skipped       пользователь нажал "Пропустить"
  -> Frozen        пользователь нажал "Заморозить" и лимит доступен
  -> Missed        рабочий день закончился без log/skip/freeze
```

`Logged`, `Skipped`, `Frozen`, `Missed` являются дневным состоянием. `Skipped` и `Missed` не прибавляют streak. `Frozen` сохраняет current streak, но не увеличивает best streak.

### Invariants

- Один canonical log per `(user_id, team_id, log_date)`. Повторная отправка в тот же день overwrites content, но streak не увеличивается повторно.
- Daily-log content видит только автор. Lead/trusted видят счетчики, динамику и risk signals, но не текст.
- Per-team streak полностью изолирован: membership в двух teams имеет две независимые streak-линии.
- Streak считается по локальной дате membership, а не по server timezone.
- Выходные и российские праздники не ломают streak и не требуют freeze.
- Freeze лимит: 2 per user/team/month. Новый месяц сбрасывает counter.
- Если Telegram не привязан, log через Telegram невозможен; web показывает persistent banner и fallback-форму только как degraded mode.
- Если пользователь покинул или удален из team, новые prompts не создаются, но historical log остается для автора до retention/delete policy Phase 4.

## Data Contracts

### Migration target

Phase 1 использует следующие таблицы. Номера миграций ориентировочные после Phase 0: `00013_daily_log.sql`, `00014_user_streak.sql`.

```sql
CREATE TABLE daily_log_entries (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    log_date DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'logged'
        CHECK (status IN ('logged', 'skipped', 'frozen', 'missed')),
    text_content TEXT NOT NULL DEFAULT '',
    has_artifact BOOLEAN NOT NULL DEFAULT FALSE,
    external_url TEXT,
    storage_key TEXT,
    mime_type TEXT,
    file_size_bytes BIGINT,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    overwritten_at TIMESTAMPTZ,
    consumed_in_check_in_id BIGINT REFERENCES check_ins(id) ON DELETE SET NULL,
    consumed_at TIMESTAMPTZ,
    UNIQUE (user_id, team_id, log_date)
);

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

### API

`POST /v1/teams/:teamId/daily-log`

Request:

```json
{
  "log_date": "2026-05-07",
  "text_content": "Разобрался, почему consistent hashing уменьшает rebalancing.",
  "external_url": "https://example.com/article",
  "client_source": "web"
}
```

Response:

```json
{
  "data": {
    "entry_id": 101,
    "status": "logged",
    "streak": {
      "current": 5,
      "best": 5,
      "is_new_record": true
    }
  }
}
```

`POST /v1/teams/:teamId/daily-log/freeze`

Request:

```json
{ "log_date": "2026-05-07", "reason": "busy" }
```

`GET /v1/teams/:teamId/daily-log?from=2026-05-01&to=2026-05-07`

- Author получает own entries with content.
- Lead/trusted получают только `{log_date, status, has_artifact}` для target user через full metrics endpoint, не через raw log endpoint.
- Member не может читать чужие entries.

### Telegram contracts

Evening prompt:

```text
Что нового узнал сегодня?

Одна строка достаточно. Если есть ссылка или скрин, прикрепи.

Стрик: 4 дня
Лог за неделю: 3 заметки

[Записать] [Пропустить] [Заморозить (2/2)]
```

Logged response:

```text
Записано.

Стрик: 5 дней. Новый рекорд.
В воскресенье соберем weekly proof из этого лога.
```

`callback_data`:

- `log:write`
- `log:skip:<team_id>:<date>`
- `log:freeze:<team_id>:<date>`
- `team:select:<team_id>`

## Flows

### Evening prompt

1. Worker каждые 5 минут выбирает active memberships, у которых local time попадает в окно 19:00-19:05.
2. Membership без Telegram link пропускается; lead получает aggregate team signal только если <50% команды linked.
3. Если на `log_date` уже есть `logged/skipped/frozen`, prompt не отправляется.
4. Notification пишется через outbox с dedup key `daily-log-prompt:<membership_id>:<date>`.
5. Telegram webhook отвечает быстро, внешние send операции идут через worker.

### Submit log

1. Пользователь отвечает текстом боту или отправляет web POST.
2. Service проверяет active membership, local date today/yesterday, минимальную длину 10 символов после trim.
3. В одной транзакции upsert `daily_log_entries`, update `user_streak`, record analytics event, publish domain event.
4. Bot возвращает streak echo.
5. Web `/me` и `/teams/:id` получают свежий heatmap через polling/refetch.

### Day rollover

1. Worker каждые 10 минут ищет memberships, у которых local date перешла за midnight.
2. Если предыдущий рабочий день без `logged/frozen/skipped`, создает `missed` и пересчитывает streak.
3. Weekend/holiday не создает missed.
4. Rollover idempotent по `(user_id, team_id, log_date)`.

## Auth and Privacy

- Author может создать, overwrite и читать свой daily-log content.
- Lead/trusted могут видеть counters, heatmap и risk flags, но не `text_content`, `external_url`, `storage_key`.
- Member может видеть peer heatmap только как aggregate activity, без daily count динамики по дням с текстом.
- Non-member не видит ничего.
- Raw log content не пишется в `analytics_events.properties`, logs, Grafana dashboards, AI prompts или notification payloads.
- Attachments наследуют private scope daily-log: они не попадают в team feed до Phase 2 proof submission.
- `team_memberships.ai_consent` не меняет Phase 1 поведение; AI отсутствует.

## Observability and Analytics

Phase 1 может писать analytics только через временный no-op Recorder, если Phase 4 event store еще не готов. События должны быть уже названы и вызываемы в доменных местах:

- `daily_log_prompted`
- `daily_log_submitted`
- `daily_log_skipped`
- `streak_incremented`
- `streak_broken`
- `streak_frozen`
- `notification_sent`
- `telegram_linked`

Required logs:

- `daily_log.prompt.sent` INFO with `team_id`, `user_id`, `local_date`, `dedup_key`.
- `daily_log.submit` INFO without content.
- `streak.update` DEBUG with old/new current and reason.
- `telegram.blocked` WARN on 403.

Alerts for alpha:

- Prompt send failure rate >5% over 30 minutes.
- Rollover job failed twice.
- More than 20% active memberships in a team have no Telegram link.

## Tests and Acceptance Criteria

### Unit tests

- `streak/rules_test.go`: consecutive days, weekend skip, Russian holiday skip, workday gap break, freeze limit, per-team isolation, timezone boundary, best streak.
- `daily_log/validation_test.go`: min length, today/yesterday local date, reject future, overwrite same day, attachment metadata derivation.
- `teams/authz_test.go`: full/peer/self metric visibility remains aligned with Phase 0 policy.

### Integration tests

- Submit log commits entry, streak, event and outbox atomically.
- Transaction rollback leaves no streak drift.
- Two same-day submits race to one entry and one streak increment.
- Rollover creates `missed` once and breaks streak only on working day.
- Telegram duplicate update_id is idempotent.

### E2E / smoke

- User links Telegram, receives prompt, submits one-line log, sees streak in bot and `/me`.
- User freezes a day, streak stays unchanged and freeze counter decrements.
- Lead opens member full metrics and cannot see raw daily-log content.
- Non-member API call to daily-log returns 403/404 without leakage.

### Phase acceptance

- Alpha week 2: at least 4/5 active users logged 4+ working days.
- Streak values match manual audit for all alpha users.
- 0 incidents where raw daily-log content was visible to lead/trusted/member or Grafana.
- `daily-log POST` p95 < 200ms without AI.

## Release and Smoke Gates

Do not release Phase 1 beyond alpha unless:

- Migrations up/down/up pass.
- `streak/` and `daily_log/` coverage >=85%.
- Privacy tests for raw content pass.
- Telegram webhook ack p95 <100ms.
- Worker prompt and rollover jobs have idempotency tests.
- Rollback plan is documented: disable prompt cron, keep submit endpoint read/write, preserve data.

Do not start Phase 2 unless daily habit success is met or product decision explicitly accepts lower adoption.
