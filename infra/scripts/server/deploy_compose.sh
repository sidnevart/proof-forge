#!/usr/bin/env bash
set -euo pipefail

DEPLOY_PATH="${DEPLOY_PATH:-/opt/proofforge-prod}"
IMAGE_TAG="${IMAGE_TAG:?IMAGE_TAG is required}"
IMAGE_NAMESPACE="${IMAGE_NAMESPACE:?IMAGE_NAMESPACE is required}"
GHCR_USERNAME="${GHCR_USERNAME:?GHCR_USERNAME is required}"
GHCR_TOKEN="${GHCR_TOKEN:?GHCR_TOKEN is required}"

cd "$DEPLOY_PATH"

if [[ ! -f .env.prod ]]; then
  echo ".env.prod is missing in $DEPLOY_PATH" >&2
  exit 1
fi

export IMAGE_TAG IMAGE_NAMESPACE
# Pick up local-only deployment overrides if the operator has dropped a
# compose.override.yml next to compose.prod.yml. We use this on the
# proof-forge.ru host to publish api/web on 127.0.0.1 so the system nginx
# (which terminates TLS for the domain) can reverse-proxy them. The override
# stays out of git because it's host-specific.
if [[ -f compose.override.yml ]]; then
  COMPOSE_ARGS=(--env-file .env.prod -f compose.prod.yml -f compose.override.yml)
else
  COMPOSE_ARGS=(--env-file .env.prod -f compose.prod.yml)
fi

printf '%s' "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USERNAME" --password-stdin
trap 'docker logout ghcr.io >/dev/null 2>&1 || true' EXIT

docker compose "${COMPOSE_ARGS[@]}" pull
docker compose "${COMPOSE_ARGS[@]}" up -d
docker compose "${COMPOSE_ARGS[@]}" restart nginx

"$DEPLOY_PATH/scripts/smoke_check.sh"
