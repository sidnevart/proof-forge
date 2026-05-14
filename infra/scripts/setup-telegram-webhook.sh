#!/usr/bin/env bash
# Setup Telegram webhook for ProofForge bot.
# Requires TELEGRAM_BOT_TOKEN and TELEGRAM_WEBHOOK_BASE_URL env vars.

set -euo pipefail

TOKEN="${TELEGRAM_BOT_TOKEN:-}"
BASE_URL="${TELEGRAM_WEBHOOK_BASE_URL:-}"
SECRET="${TELEGRAM_WEBHOOK_SECRET:-}"

if [[ -z "$TOKEN" ]]; then
  echo "Error: TELEGRAM_BOT_TOKEN is not set"
  exit 1
fi

if [[ -z "$BASE_URL" ]]; then
  echo "Error: TELEGRAM_WEBHOOK_BASE_URL is not set"
  exit 1
fi

WEBHOOK_URL="${BASE_URL%/}/telegram/webhook"

echo "Registering webhook: $WEBHOOK_URL"

if [[ -n "$SECRET" ]]; then
  curl -s -X POST "https://api.telegram.org/bot${TOKEN}/setWebhook" \
    -H "Content-Type: application/json" \
    -d "{\"url\":\"${WEBHOOK_URL}\",\"secret_token\":\"${SECRET}\",\"allowed_updates\":[\"message\",\"callback_query\"]}" \
    | jq .
else
  curl -s -X POST "https://api.telegram.org/bot${TOKEN}/setWebhook" \
    -H "Content-Type: application/json" \
    -d "{\"url\":\"${WEBHOOK_URL}\",\"allowed_updates\":[\"message\",\"callback_query\"]}" \
    | jq .
fi

echo "Done."
