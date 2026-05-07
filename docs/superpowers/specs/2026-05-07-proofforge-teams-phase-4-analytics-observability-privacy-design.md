# ProofForge Teams Phase 4 — Analytics, Grafana, Privacy Retention и Quality Gates

**Дата:** 2026-05-07
**Статус:** design spec для реализации после Phase 3
**Фаза:** 4 из 4 после foundation
**Исходный контекст:** `2026-05-07-proofforge-teams-habit-loop-design.md`, Phase 1-3 specs

---

## Goal

Довести Teams MVP до расширяемого production-ready состояния: ввести durable `analytics_events`, Grafana dashboards, privacy/retention policy, smoke/performance/security gates и операционные сигналы для daily-log, weekly proof и AI. Phase 4 не добавляет новый habit behavior; она делает уже построенный Team mode измеримым, безопасным и управляемым.

Успех фазы: три Grafana dashboards живые, analytics retention documented, privacy/security gates green, staging smoke на realistic seed проходит, launch checklist готов для расширения с alpha на 5-10 teams.

## Non-Goals

- Нет внешних SaaS-аналитик: Amplitude, Mixpanel, PostHog.
- Нет продуктовых фич, меняющих daily/proof/AI flows.
- Нет публичного BI для клиентов.
- Нет raw text search по daily-log/proof/comments в Grafana.
- Нет долгосрочного data warehouse.
- Нет event replay для восстановления доменного состояния.

## Dependencies on Previous Phases

- Phase 1-3 должны уже вызывать `analytics.Recorder` в доменных местах или иметь no-op адаптеры.
- Phase 2 должен писать review latency и proof status transitions.
- Phase 3 должен писать AI invocation/fallback/acceptance metadata без raw prompt/completion.
- Phase 0 deployment должен позволять добавить Grafana и provisioning без микросервисов.

## Domain Model

### Core nouns

- **AnalyticsEvent** — immutable append-only metadata event about product behavior.
- **EventNameRegistry** — closed list names allowed by code.
- **EventSource** — `web`, `telegram_bot`, `cron`, `system`.
- **DashboardMetric** — SQL-backed Grafana panel query.
- **RetentionPolicy** — explicit rule for event/log/cache lifetime.
- **PrivacyGate** — automated test preventing raw content leakage.
- **QualityGateRun** — pre-release verification bundle.

### Invariants

- Analytics events are facts about domain actions, not clickstream noise.
- Event writes happen in the same transaction as the domain action when the event describes a mutation.
- `analytics_events` has no FK to product tables; analytics survives deletes as metadata while privacy deletion handles content tables separately.
- `properties` never stores raw daily-log text, proof text, comments, raw prompts, completions, private URLs, tokens or invite codes.
- Frontend may emit only UI-only events from a whitelist.
- Grafana is operator-only, never exposed to sotrudnik.

## Data Contracts

### Analytics migration

```sql
CREATE TABLE analytics_events (
    id BIGSERIAL,
    ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_name TEXT NOT NULL,
    user_id BIGINT,
    team_id BIGINT,
    goal_id BIGINT,
    proof_id BIGINT,
    source TEXT NOT NULL CHECK (source IN ('web', 'telegram_bot', 'cron', 'system')),
    properties JSONB NOT NULL DEFAULT '{}',
    PRIMARY KEY (id, ts)
) PARTITION BY RANGE (ts);
```

Partition policy:

- Create monthly partitions 6 months ahead.
- Retain analytics partitions 18 months.
- Drop expired partitions with explicit operator log entry.
- Partition creator worker runs monthly on day 25 at 03:00 server time.

Indexes per partition or parent strategy:

```sql
CREATE INDEX IF NOT EXISTS idx_analytics_team_event_ts
    ON analytics_events(team_id, event_name, ts DESC);

CREATE INDEX IF NOT EXISTS idx_analytics_event_ts
    ON analytics_events(event_name, ts DESC);
```

### Event registry

Closed list for Teams MVP:

- Onboarding: `team_created`, `team_member_invited`, `team_member_joined`, `telegram_linked`.
- Daily: `daily_log_prompted`, `daily_log_submitted`, `daily_log_skipped`, `streak_incremented`, `streak_broken`, `streak_frozen`.
- Weekly: `weekly_assembly_prompted`, `weekly_assembly_started`, `weekly_assembly_completed`, `weekly_assembly_skipped`.
- Proof: `proof_submitted`, `proof_approved`, `proof_rejected`, `proof_commented`.
- Team: `team_feed_opened`, `notification_sent`, `notification_clicked`.
- AI: `personalization_invoked`, `personalization_fallback`, `personalization_accepted`, `personalization_overridden`, `user_profile_viewed`, `alert_signal_triggered`.

### API

`POST /v1/analytics/event`

Only for whitelisted UI events where no domain transaction exists, initially `team_feed_opened`, `notification_clicked`, `personalization_overridden`.

Request:

```json
{
  "event_name": "team_feed_opened",
  "team_id": 7,
  "properties": {
    "tab": "team",
    "source_surface": "web"
  }
}
```

Server attaches `user_id`, validates membership, source=`web`, strips unknown properties and rejects non-whitelisted events.

## Operational Flows

### Domain event recording

1. Domain service completes validation for a mutation such as daily-log submit, proof approval or AI fallback.
2. Service writes domain state and analytics event in the same database transaction.
3. Recorder validates event name, source and allowed properties before insert.
4. If analytics insert fails, the domain mutation rolls back for mutation events because metrics must not diverge from product state.
5. Read-only UI events use `POST /v1/analytics/event` and never block domain flows.

### Dashboard provisioning

1. Deploy applies Grafana provisioning files from the repo.
2. Grafana connects to Postgres with read-only datasource credentials.
3. Smoke opens all dashboards through nginx auth and verifies non-empty panels on staging seed.
4. Dashboard SQL snapshots are reviewed by privacy tests before release.

### Retention job

1. Partition creator ensures the next six monthly analytics partitions exist.
2. Retention job runs in dry-run mode first and logs candidate partitions older than 18 months.
3. Production drop runs only for partitions matching the exact `analytics_events_YYYY_MM` pattern and older than the retention boundary.
4. Operator log records partition name, date range and row estimate.

## Auth and Privacy

- Grafana is operator-only behind nginx basic auth and IP allowlist.
- Grafana datasource is read-only and cannot mutate product tables.
- Sotrudnik, lead and trusted users never receive direct Grafana access in MVP.
- `POST /v1/analytics/event` requires product authentication and validates team membership for any `team_id`.
- Frontend analytics endpoint accepts only UI-only events; domain mutation events must be server-side.
- Dashboard queries must use aggregate fields and never select raw text columns.
- Retention jobs never delete product content tables; they only manage analytics partitions.

## Grafana Dashboards

### Dashboard 1: Habit Health

Purpose: detect whether daily learning-log is becoming a real workday habit.

Panels:

- D7 retention by cohort: joined week -> logged >=4 working days on second week.
- Daily log submitted by local hour.
- Active streak distribution.
- Streak broken count by week.
- Freeze usage per team.
- Telegram linked ratio per team.

Alert candidates:

- D7 retention below 60% for alpha cohort.
- Telegram linked ratio below 50%.
- Streak broken spike >2x previous week.

### Dashboard 2: Approval Health

Purpose: detect whether lead/trusted review is keeping the accountability loop alive.

Panels:

- Approval latency p50/p90.
- Pending proof older than 48h.
- Approval/reject ratio.
- Approver role split: lead vs trusted.
- Proofs submitted per team/week.
- Rejected proof resubmission rate.

Alert candidates:

- Any alpha proof pending >48h.
- Approval p90 >72h.
- Weekly proof submissions drop below previous two-week baseline by 40%.

### Dashboard 3: Onboarding and AI Health

Purpose: watch team setup, Telegram activation and AI degradation.

Panels:

- Funnel: team_created -> member_joined -> telegram_linked -> first_daily_log -> first_weekly_proof.
- AI fallback rate by feature/reason.
- AI latency p50/p90 by feature.
- Daily budget usage.
- Circuit breaker open states.
- Privacy gate failures from CI as annotations.

Alert candidates:

- AI fallback >30% for feature/hour with >=20 invocations.
- Budget usage >80%.
- Circuit breaker opened.

## Privacy and Retention

### Data classes

- **Raw private content:** daily-log text, attachments before proof, reject comments, comments on rejected/pending proof threads.
- **Team-visible content:** submitted proof text, evidence metadata/files, comments on approved proof threads.
- **Operational metadata:** analytics events, IDs, counts, statuses, latencies.
- **AI cache:** generated outputs and sanitized payloads.

### Retention rules

- Analytics events: 18 months, partition drop.
- AI cache: feature TTL; evening ping 24h, assemble 7d, briefing 6h.
- AI budget rows: 18 months for cost analysis.
- Daily-log entries: retained while user/team active; on user deletion, raw daily-log rows and private attachments are deleted, while analytics metadata remains orphaned without raw content.
- Notification outbox: keep delivery logs 30 days, payload redacted.
- Application logs: 30 days in production unless incident hold is declared.

### Privacy gates

Automated tests must prove:

- Daily-log content absent from analytics properties.
- Daily-log content absent from lead/trusted full metrics.
- Daily-log content absent from peer metrics.
- Metadata-only AI prompts do not include raw notes or proof text.
- Lead briefing never includes raw daily-log content.
- Grafana dashboard queries select counts/statuses/latencies, not text columns.
- Frontend analytics endpoint rejects unallowed events and unknown properties.

### Deletion posture

For MVP expansion, deletion can be metadata-preserving:

- User content rows are deleted or anonymized according to product delete flow.
- Analytics events keep nullable/orphaned IDs and metadata, no FK and no raw content.
- Grafana remains aggregate-only and should not reveal deleted content.

## Observability

### Logs

Use structured logs with `request_id`, `team_id`, `user_id` where safe. Never log raw content.

Required high-signal log events:

- `analytics.record.failed`
- `analytics.partition.created`
- `analytics.partition.drop`
- `grafana.provisioning.loaded`
- `privacy.gate.failed`
- `quality_gate.run.completed`
- `personalization.circuit.opened`

### Metrics and smoke thresholds

- `daily-log POST` p95 <200ms.
- `team feed` p95 <300ms with 1000 proof cards.
- `member full` p95 <500ms.
- `assemble-suggestion` template p95 <100ms, LLM p95 <8000ms.
- Telegram webhook ack p95 <100ms.
- Analytics record overhead p95 <20ms inside transaction.

### Deployment

Grafana:

- Container added to compose.
- Provisioned Postgres datasource.
- Dashboards from repo-controlled JSON/provisioning.
- Exposed behind nginx `/grafana/` with basic auth and IP allowlist.
- No anonymous access.

Secrets:

- `GRAFANA_ADMIN_USER`
- `GRAFANA_ADMIN_PASSWORD`
- Existing AI/Telegram secrets remain out of specs and repo values.

## Tests and Acceptance Criteria

### Unit tests

- Event registry rejects unknown event names.
- Properties sanitizer removes unknown fields.
- Frontend analytics whitelist denies mutation/domain events.
- Retention partition naming and date ranges are deterministic.
- Privacy scanner catches banned keys: `text_content`, `comment`, `prompt`, `completion`, `invite_code`, `token`.

### Integration tests

- Domain mutation and analytics event commit/rollback together.
- Partition creator creates next six monthly partitions idempotently.
- Expired partition drop only drops partitions older than 18 months.
- Grafana SQL snapshots do not select raw text columns.
- Analytics endpoint validates membership and source.

### E2E / smoke

- Seed 50 teams x 25 members x 26 weeks and run feed/member dashboards queries.
- Open Grafana dashboards through nginx auth in staging.
- Submit daily-log, weekly proof, approval, AI fallback and verify dashboard counters move.
- Run privacy E2E: member/lead/trusted visibility boundaries.

### Phase acceptance

- Habit Health, Approval Health, Onboarding/AI dashboards show non-empty data on staging.
- Quality gate runner passes: unit, integration, E2E happy-path, security/privacy, migrations up/down/up, performance smoke, lint/typecheck.
- Security review and privacy review are pass or conditional pass with named non-blockers.
- Launch checklist and rollback notes are updated.

## Release and Smoke Gates

Do not expand beyond alpha unless:

- Grafana operator access works and anonymous access is disabled.
- All dashboard SQL queries are reviewed for raw content leakage.
- Retention worker dry-run output reviewed.
- Staging seed performance thresholds pass.
- `privacy-reviewer` finds no high/critical issues.
- `security-reviewer` finds no high/critical issues.
- Rollback can disable analytics endpoint, Grafana route and partition worker without disabling core daily/proof flows.

Phase 4 is complete when the team can answer, from dashboards and tests, whether ProofForge Teams is forming a proof-backed accountability loop without leaking private daily learning logs.
