# Telegram Webhook Setup

ProofForge receives Telegram updates via webhook (not polling). The handler lives at
`POST /telegram/webhook` and validates each request with the `X-Telegram-Bot-Api-Secret-Token`
header before processing.

---

## Prerequisites

- Domain with valid TLS (`https://yourdomain.com`) — Telegram rejects plain HTTP.
- Bot created via @BotFather — copy the `TELEGRAM_BOT_TOKEN`.
- `TELEGRAM_WEBHOOK_SECRET` set in `.env.prod` (random 20–256 chars, no spaces).
- App deployed and nginx serving `https://yourdomain.com/telegram/webhook`.

---

## Register the webhook

Run once on the server (or from any machine that can reach api.telegram.org):

```bash
BOT_TOKEN="your-bot-token"
WEBHOOK_URL="https://yourdomain.com/telegram/webhook"
SECRET="your-webhook-secret"    # same value as TELEGRAM_WEBHOOK_SECRET in .env.prod

curl -s -X POST "https://api.telegram.org/bot${BOT_TOKEN}/setWebhook" \
  -H "Content-Type: application/json" \
  -d "{
    \"url\": \"${WEBHOOK_URL}\",
    \"secret_token\": \"${SECRET}\",
    \"max_connections\": 40,
    \"allowed_updates\": [\"message\", \"callback_query\"]
  }" | jq .
```

Expected response:
```json
{"ok": true, "result": true, "description": "Webhook was set"}
```

---

## Verify registration

```bash
curl -s "https://api.telegram.org/bot${BOT_TOKEN}/getWebhookInfo" | jq .
```

Key fields to check:

| Field | Expected |
|-------|----------|
| `url` | `https://yourdomain.com/telegram/webhook` |
| `has_custom_certificate` | `false` (Let's Encrypt is trusted) |
| `pending_update_count` | 0 after a clean start |
| `last_error_message` | absent or empty |

---

## Verify the handler is reachable

Send a test POST with the correct secret — expect `200 OK`:

```bash
curl -sv -X POST "https://yourdomain.com/telegram/webhook" \
  -H "X-Telegram-Bot-Api-Secret-Token: ${SECRET}" \
  -H "Content-Type: application/json" \
  -d '{"update_id": 1, "message": {"message_id": 1, "chat": {"id": 1, "type": "private"}, "date": 1}}'
```

Send the same request **without** the secret — expect `403 Forbidden`:

```bash
curl -sv -X POST "https://yourdomain.com/telegram/webhook" \
  -H "Content-Type: application/json" \
  -d '{"update_id": 1}'
```

Check the API log for the received update:
```bash
docker compose -f /opt/proofforge/infra/docker/compose.prod.yml logs api | grep "telegram webhook"
```

---

## Rotate the secret

1. Generate a new secret: `openssl rand -hex 20`
2. Update `TELEGRAM_WEBHOOK_SECRET` in `.env.prod`
3. Re-register the webhook with the new secret (same `setWebhook` call above)
4. Restart the API so it picks up the new env value:
   ```bash
   docker compose -f compose.prod.yml up -d --no-deps api
   ```

Do steps 3 and 4 in rapid succession — there is a brief window where the old secret is
still registered but the API already expects the new one. Schedule during low-traffic hours.

---

## Delete the webhook (disable Telegram integration)

```bash
curl -s "https://api.telegram.org/bot${BOT_TOKEN}/deleteWebhook" | jq .
```

Set `TELEGRAM_BOT_TOKEN=""` in `.env.prod` and restart the API to disable the handler.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| `getWebhookInfo` shows `last_error_message` | Nginx not proxying `/telegram/webhook` or TLS issue | Check nginx config and certbot renewal |
| API logs show "invalid secret token" | Secret mismatch between `.env.prod` and registered webhook | Re-register webhook with current secret |
| `pending_update_count` keeps growing | Handler returning non-200 or timing out | Check API logs; Telegram retries for 24h |
| 403 on all requests | `TELEGRAM_ENABLED=false` (bot token not set) | Set token and restart API |
