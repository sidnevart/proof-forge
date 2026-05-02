#!/usr/bin/env bash
set -euo pipefail

# Auto-deploy script for ProofForge VPS.
# Run this on the server to pull latest images and restart the stack.
#
# Usage:
#   cd /opt/proofforge-prod
#   sudo ./scripts/auto_deploy.sh

DEPLOY_PATH="${DEPLOY_PATH:-/opt/proofforge-prod}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
IMAGE_NAMESPACE="${IMAGE_NAMESPACE:-sidnevart}"
GHCR_USERNAME="${GHCR_USERNAME:-}"
GHCR_TOKEN="${GHCR_TOKEN:-}"

cd "$DEPLOY_PATH"

if [[ ! -f .env.prod ]]; then
  echo ".env.prod is missing in $DEPLOY_PATH" >&2
  exit 1
fi

if [[ -n "$GHCR_USERNAME" && -n "$GHCR_TOKEN" ]]; then
  printf '%s' "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USERNAME" --password-stdin
else
  echo "Warning: GHCR credentials not set. Assuming already logged in." >&2
fi

export IMAGE_TAG IMAGE_NAMESPACE
COMPOSE_ARGS=(--env-file .env.prod -f compose.prod.yml)

echo "Pulling latest images..."
docker compose "${COMPOSE_ARGS[@]}" pull

echo "Restarting services..."
docker compose "${COMPOSE_ARGS[@]}" up -d

echo "Running smoke checks..."
if [[ -x "$DEPLOY_PATH/scripts/smoke_check.sh" ]]; then
  "$DEPLOY_PATH/scripts/smoke_check.sh"
else
  echo "Smoke check script not found, skipping."
fi

echo "Deploy complete."
