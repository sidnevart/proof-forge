# ProofForge Teams Phase 2 — Weekly Proof, Team Feed и Lead/Trusted Review

**Дата:** 2026-05-07
**Статус:** design spec для реализации после Phase 1
**Фаза:** 2 из 4 после foundation
**Исходный контекст:** `2026-05-07-proofforge-teams-habit-loop-design.md`, Phase 1 spec

---

## Goal

Превратить накопленный daily learning-log в proof-based accountability loop: раз в неделю sotrudnik собирает proof с обязательным артефактом, lead или trusted approver проверяет его, команда видит approved/pending карточки в team feed. Phase 2 доказывает, что daily-log не является habit tracker сам по себе, а питает проверяемый weekly proof.

Успех фазы: за две недели alpha >=50% sotrudnikov собрали хотя бы один weekly proof, >=70% submitted proof'ов approved/rejected за 48 часов, lead открывает feed >=3 раза в неделю.

## Non-Goals

- Нет LLM/AI suggestion; сборка proof только template/manual.
- Нет публичной ленты вне team.
- Нет leaderboard, likes, applause counters или shallow reactions.
- Нет cross-team feed и иерархий нескольких уровней.
- Нет автоматического approval без lead/trusted действия.
- Нет изменения Phase 1 privacy: daily-log content не становится видимым команде сам по себе.

## Dependencies on Previous Phases

- Phase 0: `teams`, `team_memberships`, role authz helpers, `check_ins.team_id`, `check_ins.approver_role`, `goals.team_id`.
- Phase 1: `daily_log_entries`, `user_streak`, Telegram link, worker/outbox patterns, daily-log privacy model.
- Phase 2 не должен менять semantics Phase 1 streak. Weekly proof approval может влиять на goal progress, но не пересчитывает daily streak.

## Domain Model

### Core nouns

- **WeeklyProofDraft** — ephemeral selection of daily entries and goal before submit.
- **TeamProof** — existing `check_ins` row scoped by `team_id`, with evidence in `evidence_items`.
- **ProofEntryLink** — связь `daily_log_entries.consumed_in_check_in_id` с submitted proof.
- **ApprovalDecision** — review by lead/trusted in `check_in_reviews`.
- **TeamFeedItem** — privacy-aware read model for team feed.
- **MemberMetricsFull** — lead/trusted view with counters, goals, proofs, risk signals.
- **MemberMetricsPeer** — member view with only allowed peer signals.

### Proof lifecycle

```text
Idle
  -> Prompted       Sunday 18:00 local time
  -> Picking        user opens weekly assembly
  -> Linking        user attaches/chooses goal + artifact
  -> Submitted      check_ins.status='submitted'
  -> Approved       lead/trusted approved
  -> Rejected       lead/trusted rejected with required comment
  -> Picking        author revises rejected proof
```

### Invariants

- Proof requires at least one artifact: external URL, file storage key, or evidence item with non-empty structured content.
- Proof belongs to exactly one team and one active team goal owned by author.
- Daily-log entries can be consumed by at most one proof. Reuse requires explicit unconsume on rejected proof revision.
- Lead/trusted can approve any team member except themselves.
- Reject requires non-empty comment; silent reject is impossible.
- Any active team member can comment on approved proof'ы. Author, lead and trusted can comment on pending/rejected threads. Comments do not trigger Telegram push to author.
- Members see approved proof'ы and limited pending team feed. Raw daily-log text appears in a proof only after the author intentionally included it in proof_text/evidence.
- Trusted approver can approve/reject, but cannot manage team roles or AI mode.

## Data Contracts

Phase 2 should reuse existing tables and add only the narrow linking/indexing needed around `check_ins`.

### Storage

`check_ins` team semantics:

- `team_id NOT NULL` means TeamProof.
- `goal_id` must point to a `goals` row with same `team_id`.
- `owner_user_id` must be active member of same team at submit time.
- `approver_role` is written on decision as `lead` or `trusted_approver`.

`evidence_items`:

- At least one row per submitted TeamProof.
- `kind` closed list for team mode: `text`, `external_url`, `file`, `pr`, `certificate`.

`daily_log_entries`:

- On submit, selected entries get `consumed_in_check_in_id = check_ins.id`, `consumed_at = now()`.
- Query for weekly assembly only returns entries by same `(user_id, team_id)` and date range.

Recommended additional indexes if not covered by Phase 0:

```sql
CREATE INDEX IF NOT EXISTS idx_check_ins_team_submitted
    ON check_ins(team_id, submitted_at DESC, id DESC)
    WHERE team_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_daily_log_entries_unconsumed_week
    ON daily_log_entries(team_id, user_id, log_date)
    WHERE consumed_in_check_in_id IS NULL AND status = 'logged';
```

### API

`POST /v1/teams/:teamId/proofs`

Request:

```json
{
  "goal_id": 42,
  "daily_log_entry_ids": [101, 103],
  "proof_text": "На этой неделе разобрал consistent hashing и применил это в design doc.",
  "evidence": [
    {
      "kind": "external_url",
      "external_url": "https://github.com/org/repo/pull/17",
      "label": "PR с реализацией"
    }
  ]
}
```

Response:

```json
{
  "data": {
    "proof_id": 501,
    "status": "submitted",
    "team_id": 7,
    "approval_required_from": ["lead", "trusted_approver"]
  }
}
```

`GET /v1/teams/:teamId/feed?tab=team&cursor=&limit=`

Returns feed cards ordered by submitted_at desc. Cards include author alias, goal title, proof text, evidence metadata, status, streak snapshot, comments count, and viewer-specific actions.

`POST /v1/proofs/:id/approve`

Request:

```json
{ "idempotency_key": "f9e5f1e6-2fc0-41d2-9e6b-7f061f7c2f54" }
```

`POST /v1/proofs/:id/reject`

Request:

```json
{
  "comment": "Артефакт не показывает прогресс по этому пункту ИПР. Приложи PR или конспект.",
  "idempotency_key": "731a2707-59b1-43dc-a5fb-e46d501c55e8"
}
```

`POST /v1/proofs/:id/comments`

Request:

```json
{ "comment": "Могу дать ссылку на хороший разбор trade-offs." }
```

### Metrics API

- `GET /v1/teams/:teamId/members/:userId/full` — lead/trusted only.
- `GET /v1/teams/:teamId/members/:userId/peer` — active members.
- `GET /v1/users/:userId/metrics/self` — self only.

Full view includes rejected/pending proofs, daily-log counters, goal progress and risk signals. Peer view includes only streak, tenure, goal titles and approved proofs.

## Telegram and Web Flows

### Sunday assembly prompt

```text
Воскресенье - время собрать proof.

За неделю у тебя 5 заметок, из них 2 с артефактами:
ср - consistent hashing
чт - семинар с Артемом

[Собрать из этих] [Выбрать другие] [Не было движения]
```

Flow:

1. Worker sends prompt Sunday 18:00 local time for active memberships with at least one logged entry in the week.
2. User chooses suggested entries or opens manual picker.
3. Bot asks for goal if more than one active team goal exists.
4. Bot requires artifact before submit. If no artifact, it asks for URL/file instead of submitting.
5. Submit creates check_in, evidence_items, consumes entries, records event and sends notifications in one transaction.

### Approval

1. Lead and trusted receive Telegram notification with approve/reject/open web actions.
2. Approval callback checks role via `teams.CanApproveProofForOwner`.
3. Approve writes `check_in_reviews(decision='approved')`, sets `check_ins.approved_at`, records latency, notifies author.
4. Reject requires comment. Bot enters `AwaitingRejectComment` state if callback had no comment.
5. Author sees reject reason and can resubmit revised proof.

### Team feed

- `/feed?tab=team` shows team proof cards immediately after submit.
- Pending/rejected cards are visible to author, lead and trusted. Ordinary members see only approved proof'ы; they do not see pending/rejected proof text, evidence or comments.
- Polling 30 seconds; no SSE/WebSocket in Phase 2.

## Auth and Privacy

| Surface | Author | Lead | Trusted | Member | Non-member |
|---|---:|---:|---:|---:|---:|
| Submit own proof | yes | yes if owner | yes if owner | yes | no |
| Approve proof | no self | yes | yes | no | no |
| Reject proof | no self | yes | yes | no | no |
| Comment | yes | yes | yes | yes | no |
| See raw daily-log | own only | no | no | no | no |
| See approved proof | yes | yes | yes | yes | no |
| See rejected/pending proof | own | yes | yes | no | no |
| Full member metrics | self | yes | yes | no | no |
| Peer metrics | yes | yes | yes | yes limited | no |

Privacy rules:

- Selecting a daily-log entry for proof is explicit publication of selected content into proof. Non-selected log content remains private.
- Analytics properties must not include proof text, comments or daily-log text.
- Rejection comments are visible to author, lead and trusted; ordinary members do not see rejected proof threads.
- Files stay private until attached to submitted proof. Proof attachments inherit team visibility, not public visibility.

## Observability and Analytics

Events:

- `weekly_assembly_prompted`
- `weekly_assembly_started`
- `weekly_assembly_completed`
- `weekly_assembly_skipped`
- `proof_submitted`
- `proof_approved`
- `proof_rejected`
- `proof_commented`
- `team_feed_opened`
- `notification_sent`
- `notification_clicked`

Properties allowed:

- IDs, counts, status, latency_seconds, approver_role, evidence_kind_count.
- No text content, no comments, no URLs. Use counts, status, latency, evidence kind and approver role only.

Operational logs:

- `weekly_proof.submit` INFO with ids/counts only.
- `proof.review` INFO with decision, approver_role, latency.
- `proof.review.denied` WARN for role/authz failures.
- `team_feed.query` DEBUG with duration and row count.

Alerts:

- Pending proof older than 48h count >0 in alpha team.
- Approval callback failures >5% over 30 minutes.
- Team feed p95 >300ms with alpha dataset.

## Tests and Acceptance Criteria

### Unit tests

- Team proof validation: goal in team, author active member, artifact required, entries owned by author, entries unconsumed.
- Authz: lead/trusted approve all except self, member denied, removed membership denied.
- Reject comment required.
- Metrics visibility matches table above.

### Integration tests

- Submit proof consumes entries and creates evidence atomically.
- Race: two submissions cannot consume the same daily entry.
- Approve race: two approvers produce one terminal state.
- Reject then resubmit preserves audit history and allows revised proof.
- Notification outbox uses dedup keys.
- Feed query excludes cross-team proofs.

### E2E / smoke

- User receives Sunday prompt, submits proof with URL, lead approves from Telegram, author receives approval.
- Lead rejects without comment path asks for comment and does not reject until comment exists.
- Member can comment but cannot approve.
- Member cannot read another member's rejected proof.
- `/teams/:id/members/:userId/full` hides daily-log text.

### Phase acceptance

- >=50% active alpha sotrudnikov submit >=1 proof within 2 weeks.
- >=70% submitted proof'ов reviewed within 48h.
- Lead opens team feed >=3 times/week.
- 0 cases of daily-log content becoming visible without explicit proof submission.

## Release and Smoke Gates

Do not release Phase 2 beyond alpha unless:

- Phase 1 success criteria were met or explicitly waived.
- Team proof unit/integration tests pass.
- `checkins/` team-aware coverage >=75%.
- Feed p95 <300ms with 1000 proof cards seed.
- Approval/reject Telegram callbacks pass duplicate update tests.
- Privacy E2E proves member cannot access rejected/pending proof content.
- Rollback plan exists: disable weekly prompt and proof submit while preserving feed read-only.
