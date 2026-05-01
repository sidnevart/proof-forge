#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
COMPOSE_DIR="${REPO_ROOT}/infra/docker"

ENV_FILE="${COMPOSE_DIR}/.env.dev"
COMPOSE_FILE="${COMPOSE_DIR}/compose.dev.yml"

cd "${COMPOSE_DIR}"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Environment file missing: ${ENV_FILE}" >&2
  echo "Create it from the template:" >&2
  echo "  cp ${COMPOSE_DIR}/.env.dev.example ${ENV_FILE}" >&2
  exit 1
fi

echo "Starting ProofForge local dev stack..."
docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" up --build -d

echo ""
echo "Services should be starting up. Health checks:"
echo "  API health:    curl http://localhost:8080/healthz"
echo "  API ready:     curl http://localhost:8080/readyz"
echo "  Web:           curl http://localhost:3003/"
echo "  MinIO console: http://localhost:59001/"
echo ""
echo "To stop:  docker compose -f ${COMPOSE_FILE} down"
echo "To logs:  docker compose -f ${COMPOSE_FILE} logs -f"
