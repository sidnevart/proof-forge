# ProofForge Post-Deploy Smoke Test

Run these checks immediately after every deploy. Each step has an explicit expected
result and a blocker verdict. Stop on any FAIL — do not proceed to the next section.

Set BASE once at the top of your terminal session:
```bash
BASE="https://yourdomain.com"
```

---

## 1. Infrastructure Health

### 1a. API readiness
```bash
curl -sf "${BASE}/readiness" | jq .
```
**Expected:** `{"status":"ok"}` with HTTP 200.  
**Blocker if:** non-200 or `"status":"degraded"` (postgres or redis unreachable).

### 1b. Container status
```bash
docker compose -f /opt/proofforge/infra/docker/compose.prod.yml ps
```
**Expected:** All 6 services (`postgres`, `redis`, `minio`, `api`, `worker`, `web`, `nginx`) show `running`. `minio-init` shows `exited (0)`.  
**Blocker if:** Any service in `restarting` or `exited (non-zero)`.

### 1c. Nginx TLS
```bash
curl -sI "${BASE}" | head -5
```
**Expected:** `HTTP/2 200` and `strict-transport-security` header present.  
**Blocker if:** TLS handshake error or missing HSTS header.

---

## 2. User Registration

```bash
USER_EMAIL="smoke-$(date +%s)@example.com"

curl -sf -c /tmp/pf-smoke.jar -X POST "${BASE}/v1/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${USER_EMAIL}\",\"display_name\":\"Smoke Owner\"}" | jq .
```
**Expected:** `{"user":{"id":<n>,"email":"...","display_name":"Smoke Owner",...}}` with HTTP 201.  
Session cookie `pf_session` written to `/tmp/pf-smoke.jar`.  
**Blocker if:** non-201 or no session cookie in jar.

---

## 3. Goal Creation

```bash
GOAL=$(curl -sf -b /tmp/pf-smoke.jar -c /tmp/pf-smoke.jar \
  -X POST "${BASE}/v1/goals" \
  -H "Content-Type: application/json" \
  -d '{"title":"Smoke goal","description":"Deploy validation","buddy_name":"Buddy","buddy_email":"buddy-smoke@example.com"}')
echo "${GOAL}" | jq .
GOAL_ID=$(echo "${GOAL}" | jq -r '.goal.goal.id')
```
**Expected:** Goal object with `status:"pending_buddy_acceptance"` and a non-null invite token. `GOAL_ID` must be a positive integer.  
**Blocker if:** non-200 or `GOAL_ID` is null/empty.

---

## 4. Dashboard

```bash
curl -sf -b /tmp/pf-smoke.jar "${BASE}/v1/dashboard" | jq '{total_goals:.summary.total_goals}'
```
**Expected:** `{"total_goals":1}` (the smoke goal created in step 3).  
**Blocker if:** 0 goals or non-200.

---

## 5. Check-In Flow

### 5a. Create draft check-in
```bash
CI=$(curl -sf -b /tmp/pf-smoke.jar -c /tmp/pf-smoke.jar \
  -X POST "${BASE}/v1/goals/${GOAL_ID}/check-ins" \
  -H "Content-Type: application/json" -d '{}')
echo "${CI}" | jq .
CI_ID=$(echo "${CI}" | jq -r '.check_in.id')
```
**Expected:** Check-in with `status:"draft"`. `CI_ID` is a positive integer.  
**Blocker if:** non-200 or `CI_ID` null.

### 5b. Add text evidence
```bash
curl -sf -b /tmp/pf-smoke.jar \
  -X POST "${BASE}/v1/check-ins/${CI_ID}/evidence/text" \
  -H "Content-Type: application/json" \
  -d '{"content":"Smoke test proof artifact — deploy validation."}' | jq .
```
**Expected:** `{"evidence":{"id":<n>,"kind":"text",...}}` with HTTP 200.  
**Blocker if:** non-200 or missing evidence id.

### 5c. Submit check-in
```bash
curl -sf -b /tmp/pf-smoke.jar \
  -X POST "${BASE}/v1/check-ins/${CI_ID}/submit" \
  -H "Content-Type: application/json" -d '{}' | jq .
```
**Expected:** `{"submitted":true}`.  
**Blocker if:** non-200 or `submitted:false`.

---

## 6. Recap List (No Recaps Yet)

```bash
curl -sf -b /tmp/pf-smoke.jar "${BASE}/v1/goals/${GOAL_ID}/recaps" | jq .
```
**Expected:** `{"recaps":null}` or `{"recaps":[]}` — no recaps until buddy approves a check-in and the weekly sweep runs.  
**Blocker if:** 5xx error. Empty or null is correct at this stage.

---

## 7. Unauthenticated Rejection

```bash
curl -sI "${BASE}/v1/dashboard" | head -3
```
**Expected:** `HTTP/2 401` — no session cookie used.  
**Blocker if:** 200 or 500.

---

## 8. Rate Limiting Probe

```bash
for i in $(seq 1 12); do
  curl -so /dev/null -w "%{http_code}\n" -X POST "${BASE}/v1/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"ratelimit-${i}@example.com\",\"display_name\":\"RL\"}"
done
```
**Expected:** The first 10 return 201; requests 11–12 return 429 (nginx rate limit zone `api_auth`).  
**Blocker if:** no 429 at all — rate limiting misconfigured.

---

## 9. Security Headers

```bash
curl -sI "${BASE}" | grep -iE "strict-transport|x-frame|x-content-type|referrer-policy|content-security"
```
**Expected (all 5 headers present):**
```
strict-transport-security: max-age=31536000; includeSubDomains
x-frame-options: DENY
x-content-type-options: nosniff
referrer-policy: strict-origin-when-cross-origin
content-security-policy: ...
```
**Blocker if:** any of the 5 headers missing.

---

## 10. Telegram Webhook (if configured)

```bash
# Only run if TELEGRAM_BOT_TOKEN is set
SECRET=$(grep TELEGRAM_WEBHOOK_SECRET /opt/proofforge/infra/docker/.env.prod | cut -d= -f2)

curl -sv -X POST "${BASE}/telegram/webhook" \
  -H "X-Telegram-Bot-Api-Secret-Token: ${SECRET}" \
  -H "Content-Type: application/json" \
  -d '{"update_id":1,"message":{"message_id":1,"chat":{"id":1,"type":"private"},"date":1}}' \
  2>&1 | grep "< HTTP"

curl -sI -X POST "${BASE}/telegram/webhook" \
  -H "Content-Type: application/json" \
  -d '{"update_id":1}' | head -3
```
**Expected:** First request → `HTTP/2 200`. Second (no secret) → `HTTP/2 403`.  
**Blocker if:** first returns 403, or second returns 200.

---

## 11. Cleanup

```bash
rm -f /tmp/pf-smoke.jar
```

---

## Summary Checklist

| # | Check | Expected | Status |
|---|-------|----------|--------|
| 1a | API readiness | 200 `{"status":"ok"}` | |
| 1b | All containers running | 7 running, minio-init exited 0 | |
| 1c | TLS + HSTS | HTTP/2 200, HSTS header | |
| 2 | User registration | 201 + session cookie | |
| 3 | Goal creation | 200 + goal id | |
| 4 | Dashboard | `total_goals:1` | |
| 5a | Draft check-in | 200 + check-in id | |
| 5b | Text evidence | 200 + evidence id | |
| 5c | Submit check-in | `submitted:true` | |
| 6 | Recap list | 200 (null/empty ok) | |
| 7 | Unauth rejection | 401 | |
| 8 | Rate limiting | 429 after burst | |
| 9 | Security headers | All 5 present | |
| 10 | Telegram webhook | 200 / 403 pair | |

**PASS**: all rows checked with expected result.  
**FAIL**: any row deviates → roll back, fix, redeploy, rerun from step 1.
