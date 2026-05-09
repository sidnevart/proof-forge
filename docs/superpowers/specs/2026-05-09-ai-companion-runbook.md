# AI Companion — Emergency Runbook

**Date:** 2026-05-09  
**Status:** active  
**Scope:** production incident response for AI companion subsystem

---

## Emergency: Disable All LLM Calls Immediately

### Option 1: Kill switch via env var (fastest — 1 minute)

Set `COMPANION_ENABLED=false` and restart the worker container:

```bash
# Docker Compose
docker compose exec api env COMPANION_ENABLED=false

# Or edit .env and restart
docker compose restart api worker
```

This disables the companion worker entirely — no triggers fire, no LLM calls, no notifications.

### Option 2: Circuit breaker (no deploy)

Insert a row into `ai_circuit_breakers` to open the circuit for all features:

```sql
INSERT INTO ai_circuit_breakers (feature, state, opened_at, failure_count)
VALUES ('all', 'open', NOW(), 999)
ON CONFLICT (feature) DO UPDATE
SET state = 'open', opened_at = NOW(), failure_count = 999;
```

Effect: all companion features fall back to templates within ~5 minutes (next worker tick).

### Option 3: Per-feature kill switch

Disable a single feature without touching others:

```sql
-- Disable evening ping only
UPDATE companion_feature_flags SET enabled = false WHERE feature = 'evening_ping';
```

Or via env var:

```bash
COMPANION_FEATURE_evening_ping=false
```

---

## Emergency: Privacy Leak Suspected

### Immediate steps

1. **Disable the suspected feature** using Option 3 above.
2. **Check logs** for the last hour:
   ```bash
   docker compose logs api | grep "companion\|personalization\|llm"
   ```
3. **Verify no raw content in notifications**:
   ```sql
   SELECT feature, LEFT(body, 200) FROM ai_notifications
   WHERE created_at >= NOW() - INTERVAL '1 hour';
   ```
4. **If evidence text leaked in lead brief**: the lead brief assembler is designed to only query aggregates. If raw content appears, the bug is in `GetLeadBriefContext` — check the SQL.

---

## Emergency: Runaway Token Spend

### Check current spend

```sql
SELECT SUM(tokens_in + tokens_out) FROM ai_companion_fired
WHERE fired_at >= CURRENT_DATE;
```

### Throttle

Set daily budget to zero (immediate template fallback):

```sql
UPDATE ai_daily_budget SET budget_tokens = 0, date = CURRENT_DATE;
```

Or reduce via env:

```bash
COMPANION_DAILY_BUDGET_TOKENS=0
```

---

## Emergency: Telegram Spam

### Disable Telegram delivery only

```bash
COMPANION_TELEGRAM_ENABLED=false
```

Notifications will still appear in-app. No Telegram messages sent.

---

## Rollback Plan

If a release causes companion instability:

1. `git revert <companion-commit>`
2. `docker compose build api worker`
3. `docker compose up -d api worker`

The companion tables (`ai_companion_fired`, `ai_proof_drafts`, `ai_notifications`) are additive — rollback does not delete user data.

---

## Feature Flags Reference

| Flag | Default | Effect |
|---|---|---|
| `COMPANION_ENABLED` | `true` | Master switch. `false` = worker exits immediately, no triggers |
| `COMPANION_TELEGRAM_ENABLED` | `true` | Telegram delivery surface |
| `COMPANION_FEATURE_evening_ping` | `true` | Time trigger |
| `COMPANION_FEATURE_weekly_recap` | `true` | Time trigger |
| `COMPANION_FEATURE_lead_weekly_brief` | `true` | Time trigger |
| `COMPANION_FEATURE_streak_reminder` | `true` | Time trigger |
| `COMPANION_FEATURE_proof_draft` | `true` | Event trigger |
| `COMPANION_FEATURE_buddy_stalled` | `true` | Event trigger |
| `COMPANION_FEATURE_goal_risk` | `true` | Event trigger |
| `COMPANION_FEATURE_streak_milestone` | `true` | Event trigger |
| `COMPANION_FEATURE_leader_fair_play` | `true` | Event trigger |
| `COMPANION_DAILY_BUDGET_TOKENS` | `500000` | Global LLM token budget |

---

## Health Check Queries

### Fallback rate (last 24h)

```sql
SELECT
  feature,
  COUNT(*) FILTER (WHERE fallback_reason IS NOT NULL) AS fallback_count,
  COUNT(*) AS total,
  ROUND(100.0 * COUNT(*) FILTER (WHERE fallback_reason IS NOT NULL) / NULLIF(COUNT(*), 0), 1) AS fallback_pct
FROM ai_companion_fired
WHERE fired_at >= NOW() - INTERVAL '24 hours'
GROUP BY feature;
```

Target: fallback rate < 20% per feature.

### Pending notifications per user (alert if > 5)

```sql
SELECT user_id, COUNT(*) AS cnt
FROM ai_notifications
WHERE dismissed_at IS NULL
GROUP BY user_id
HAVING COUNT(*) > 5;
```

### Stalled proofs awaiting buddy (alert if > 10)

```sql
SELECT COUNT(*) FROM check_ins
WHERE status = 'submitted'
  AND submitted_at <= NOW() - INTERVAL '48 hours';
```

---

## Contacts

- On-call engineer: check PagerDuty
- AI companion owner: see git blame on `backend/internal/companion/`
- Privacy officer: legal@company.com

---

## Post-incident

After any emergency:

1. Document in incident log (Notion / Ops log)
2. Update this runbook if steps were missing
3. Review analytics events for the window
4. Decide if feature needs circuit breaker tuning or prompt hardening
