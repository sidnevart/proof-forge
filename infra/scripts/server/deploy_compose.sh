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

printf '%s' "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USERNAME" --password-stdin
trap 'docker logout ghcr.io >/dev/null 2>&1 || true' EXIT

docker compose -f compose.prod.yml pull
docker compose -f compose.prod.yml up -d

"$DEPLOY_PATH/scripts/smoke_check.sh"
