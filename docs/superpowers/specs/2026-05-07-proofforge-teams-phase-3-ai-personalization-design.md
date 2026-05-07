# ProofForge Teams Phase 3 — AI Personalization Layer

**Дата:** 2026-05-07
**Статус:** design spec для реализации после Phase 2
**Фаза:** 3 из 4 после foundation
**Исходный контекст:** `2026-05-07-proofforge-teams-habit-loop-design.md`, Phase 1-2 specs

---

## Goal

Добавить AI-персонализацию поверх уже работающего daily-log и weekly proof loop так, чтобы продукт стал удобнее, но не зависел от LLM на критическом пути. AI помогает формулировать вечерний пинг, собирать weekly proof из лога и давать briefing тимлиду, но каждый сценарий имеет deterministic fallback и строгую privacy boundary.

Успех фазы: assemble suggestion acceptance >=60%, AI week in alpha проходит без circuit-breaker, daily budget не превышается, privacy tests не находят передачи запрещенного контента.

## Non-Goals

- AI не оценивает людей и не принимает approval decisions.
- AI не заменяет lead/trusted review.
- AI не получает raw daily-log в `metadata-only`.
- AI briefing не является performance review и не генерирует HR-вердикты.
- Нет чат-бота "спроси что угодно".
- Нет обучения моделей на пользовательских данных.
- Нет hard dependency на Kimi/Ollama availability.

## Dependencies on Previous Phases

- Phase 0: `teams.ai_mode`, `team_memberships.ai_consent`.
- Phase 1: daily-log entries, streak state, prompt worker, Telegram linking.
- Phase 2: team proofs, evidence, approval history, member metrics.
- Phase 4 analytics store может еще не существовать; Phase 3 обязан вызывать Recorder интерфейс с AI events, но работает с no-op implementation.

## Domain Model

### Core nouns

- **PersonalizationFeature** — `evening_ping`, `assemble_proof`, `lead_briefing`.
- **AIMode** — per-team privacy posture: `off`, `metadata-only`, `full`.
- **AIConsent** — per-membership individual consent.
- **PersonalizationRequest** — normalized context after privacy filtering.
- **PersonalizationResult** — validated output or template fallback.
- **TemplateProvider** — deterministic provider for all features.
- **LLMProvider** — Ollama-compatible provider for Kimi K2 cloud.
- **PromptVersion** — immutable prompt definition with validation rules.
- **CircuitBreakerState** — per-feature state that temporarily forces template fallback.
- **BudgetState** — daily token/request accounting.

### Privacy mode matrix

| Mode | Team setting | User consent | Raw daily-log to LLM | Proof content to LLM | Feature behavior |
|---|---|---:|---:|---:|---|
| Off | `off` | any | no | no | templates only |
| Metadata | `metadata-only` | any | no | approved proof metadata only | limited personalization |
| Full without consent | `full` | false | no | approved proof metadata only | same as metadata for that user |
| Full with consent | `full` | true | feature-specific | approved proof content allowed | full feature set |

Critical exception: lead briefing never sends raw daily-log content, even in full mode. It may use aggregates and approved proof content only.

### Invariants

- Every AI feature has a deterministic fallback that passes the same output schema.
- LLM output is never trusted until validated.
- Prompt inputs are built by privacy-filtered context assemblers, not by handlers.
- Raw notes/comments are never logged.
- AI responses are visibly marked with `AI` or `✨` in UI/bot surfaces.
- Circuit breaker disables feature, not whole product.
- Budget exceeded forces template fallback until the next budget day.

## Data and API Contracts

### Migration target

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

CREATE TABLE ai_circuit_breakers (
    feature TEXT PRIMARY KEY,
    state TEXT NOT NULL CHECK (state IN ('closed', 'open')),
    opened_until TIMESTAMPTZ,
    reason TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Service interface

```go
type Service interface {
    EveningPing(ctx context.Context, req EveningPingRequest) (EveningPingResult, error)
    AssembleProof(ctx context.Context, req AssembleProofRequest) (AssembleProofResult, error)
    LeadBriefing(ctx context.Context, req LeadBriefingRequest) (LeadBriefingResult, error)
}
```

Errors from provider do not propagate to users unless both provider and template fail validation. That should be treated as `internal.unexpected` and alert.

### Env

```text
OLLAMA_BASE_URL
OLLAMA_API_KEY
OLLAMA_MODEL=kimi-k2.6:cloud
PERSONALIZATION_ENABLED=true
PERSONALIZATION_TIMEOUT_PING_MS=2000
PERSONALIZATION_TIMEOUT_ASSEMBLY_MS=8000
PERSONALIZATION_TIMEOUT_BRIEFING_MS=4000
PERSONALIZATION_DAILY_BUDGET_TOKENS=500000
```

### Prompt catalog

Prompt files live under `backend/internal/personalization/prompts/*.yaml` with immutable versions.

```yaml
version: 1
feature: evening_ping
model: kimi-k2.6:cloud
validation:
  min_words: 8
  max_words: 25
  must_end_with: "?"
  forbidden_substrings: ["молодец", "отлично", "плохо", "ты должен"]
```

Prompt version is recorded in `personalization_invoked.properties.prompt_version`.

## Feature Flows

### 3.1 Evening ping

Context:

- Alias or display name.
- Current streak and week log count.
- Active team goal titles.
- Last 3-5 note summaries only in full mode with consent.
- Day of week and local time.

Validation:

- 8-25 words.
- Ends with question mark.
- Does not contain praise/shame/control language.
- Does not mention private content if privacy mode filtered it out.

Flow:

1. Prewarm job runs 30 minutes before prompt window.
2. Service checks enabled, budget and circuit breaker.
3. Context assembler applies privacy mode.
4. LLM provider races timeout; fallback template on timeout/error/validation fail.
5. Prompt is cached by `(feature, team_id, user_id, local_date, prompt_version)`.
6. Evening prompt uses cached text; send path is not blocked by LLM.

### 3.2 Assemble proof

Context:

- Current week's logged entries and artifact metadata.
- Active team goals.
- Last approved proof metadata; content only in full mode with consent.

Output:

```json
{
  "candidates": [
    {
      "ipr_goal_id": 42,
      "note_ids": [101, 103],
      "rationale": "Обе заметки про один technical theme и есть артефакт.",
      "confidence": "high"
    }
  ]
}
```

Validation:

- Candidate goal belongs to same team and author.
- Note IDs belong to same author/team/week.
- High-confidence candidate has at least one artifact.
- Max 3 candidates.
- Rationale max 240 chars and no evaluative judgment.

Flow:

1. User presses `proof:assemble`.
2. Bot shows "Собираю..." and waits up to 8 seconds.
3. LLM result is validated. On fail, TemplateProvider groups unconsumed entries with artifacts by goal/date proximity.
4. User chooses candidate, edits, or switches to manual.
5. Final submit still uses Phase 2 validation; AI cannot bypass artifact requirement.

### 3.3 Lead briefing

Context:

- Last 30 days proof submitted/approved/rejected counts.
- Approval latency.
- Streak aggregates.
- Goal coverage and risk signals.
- Approved proof content allowed only if team mode permits and author consent allows; daily-log content never included.

Output constraints:

- 30-80 words.
- One risk section or explicit "Без рисков".
- No labels like weak/strong/poor performer.
- Must cite observed facts by counts/dates, not personality claims.

Flow:

1. Lead/trusted opens `/teams/:id/members/:userId`.
2. API returns cached briefing if fresh (<6h).
3. Cache miss starts synchronous attempt with 4s timeout; fallback statistical summary if timeout.
4. Cache invalidates on `proof_submitted`, `proof_approved`, `proof_rejected`.
5. UI has `Скрыть briefing` and `Сообщить, что бесполезно`; both are product events.

## Auth and Privacy

- Only lead/trusted can request lead briefing for another member.
- Member can request AI assemble for own proof only.
- Evening ping context is only for the receiving user.
- Team `full` mode is insufficient without individual `ai_consent`.
- Consent can be changed only by the user themselves.
- AI provider payloads never include email, Telegram chat_id, raw comments, secrets or invite codes.
- Metadata-only mode may include IDs, aliases, goal titles, counts, artifact types and dates.
- Prompt injection from user content is treated as untrusted text; system prompt states policy, validator enforces output shape, service never executes instructions from content.

## Observability and Analytics

Events:

- `personalization_invoked`
- `personalization_fallback`
- `personalization_accepted`
- `personalization_overridden`
- `user_profile_viewed`
- `alert_signal_triggered`

Required properties:

- `feature`
- `provider` (`template`, `kimi`)
- `mode` (`off`, `metadata-only`, `full`)
- `prompt_version`
- `latency_ms`
- `fallback_reason` when applicable
- `validation_error_code` when applicable
- `tokens_in`, `tokens_out` when available

Forbidden properties:

- Raw prompt text.
- Raw completion text for user content-bearing features.
- Daily-log text, proof text, comments, URLs with private tokens.

Logs:

- INFO on invoked/fallback with metadata.
- WARN on validation failure and circuit breaker open.
- ERROR on budget store failure or template validation failure.

Circuit breaker:

- Per feature, open for 1 hour if fallback due to timeout/validation/provider_error exceeds 30% in the last hour and at least 20 invocations occurred.
- Open state writes alert event and Grafana annotation in Phase 4.

## Tests and Acceptance Criteria

### Unit tests

- Privacy context assembler for every mode/consent combination.
- Evening ping validation: length, forbidden words, question mark.
- Assemble validation: no cross-team note, no no-artifact high confidence, max candidates.
- Briefing validation: length, one risk section, no daily-log content.
- Circuit breaker transitions closed/open/closed.
- Budget exceeded forces template.

### Integration tests

- Provider timeout falls back and records event.
- Validation fail falls back without retry.
- Cache hit avoids provider call.
- Cache invalidates on proof events.
- Consent false in full mode still strips raw notes.
- Lead briefing denied for ordinary member and non-member.

### Nightly live tests

- Kimi evening ping passes validation.
- Kimi assemble returns valid candidates on seeded dataset.
- Kimi briefing passes validation.
- p90 latency under configured timeout for alpha-sized payloads.

Nightly failures create incident but do not block production if template fallback remains green.

### Phase acceptance

- Assemble accepted by user >=60% in alpha.
- AI overridden/report useless <10% for briefing.
- Circuit breaker does not open for a full alpha week.
- Budget stays below 80% daily threshold.
- 0 privacy findings for metadata-only or consent boundaries.

## Release and Smoke Gates

Roll out in order:

1. Evening ping for alpha only.
2. Assemble proof for alpha only.
3. Lead briefing for alpha leads/trusted.
4. Partner team expansion after one stable week.

Do not enable next AI feature unless previous feature has:

- Template fallback green.
- Validation tests green.
- No privacy leak in smoke.
- Grafana/temporary logs show fallback rate under 20%.

Emergency controls:

- `PERSONALIZATION_ENABLED=false` disables all LLM calls.
- Team `ai_mode=off` disables AI per team.
- Circuit breaker disables feature automatically.
- Budget exceeded disables provider until next day.
