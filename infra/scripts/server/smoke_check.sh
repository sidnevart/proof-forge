#!/usr/bin/env bash
set -euo pipefail

DEPLOY_PATH="${DEPLOY_PATH:-/opt/proofforge-prod}"
HTTP_PORT="${PROOFFORGE_HTTP_PORT:-18080}"
HTTPS_PORT="${PROOFFORGE_HTTPS_PORT:-18443}"

cd "$DEPLOY_PATH"

docker compose --env-file .env.prod -f compose.prod.yml ps
curl -fsS "http://127.0.0.1:${HTTP_PORT}/" >/dev/null
curl -kfsS "https://127.0.0.1:${HTTPS_PORT}/" >/dev/null
curl -kfsS "https://127.0.0.1:${HTTPS_PORT}/readyz" >/dev/null
