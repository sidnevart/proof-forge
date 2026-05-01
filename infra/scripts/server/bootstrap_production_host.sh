#!/usr/bin/env bash
set -euo pipefail

DEPLOY_USER="${DEPLOY_USER:-proofforge-deploy}"
DEPLOY_GROUP="${DEPLOY_GROUP:-$DEPLOY_USER}"
DEPLOY_HOME="${DEPLOY_HOME:-/home/$DEPLOY_USER}"
DEPLOY_PATH="${DEPLOY_PATH:-/opt/proofforge-prod}"
SSH_PUBLIC_KEY_PATH="${SSH_PUBLIC_KEY_PATH:-}"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "This script must run as root." >&2
  exit 1
fi

if [[ -z "$SSH_PUBLIC_KEY_PATH" || ! -f "$SSH_PUBLIC_KEY_PATH" ]]; then
  echo "SSH_PUBLIC_KEY_PATH must point to an existing public key file." >&2
  exit 1
fi

if ! getent group docker >/dev/null; then
  echo "docker group is required on the host." >&2
  exit 1
fi

if ! id "$DEPLOY_USER" >/dev/null 2>&1; then
  groupadd --system "$DEPLOY_GROUP"
  useradd --system --create-home --home-dir "$DEPLOY_HOME" --gid "$DEPLOY_GROUP" --shell /bin/bash "$DEPLOY_USER"
fi

usermod -aG docker "$DEPLOY_USER"

install -d -m 750 -o "$DEPLOY_USER" -g "$DEPLOY_GROUP" "$DEPLOY_PATH"
install -d -m 750 -o "$DEPLOY_USER" -g "$DEPLOY_GROUP" "$DEPLOY_PATH/nginx"
install -d -m 750 -o "$DEPLOY_USER" -g "$DEPLOY_GROUP" "$DEPLOY_PATH/nginx/certs"
install -d -m 750 -o "$DEPLOY_USER" -g "$DEPLOY_GROUP" "$DEPLOY_PATH/scripts"

install -d -m 700 -o "$DEPLOY_USER" -g "$DEPLOY_GROUP" "$DEPLOY_HOME/.ssh"
install -m 600 -o "$DEPLOY_USER" -g "$DEPLOY_GROUP" "$SSH_PUBLIC_KEY_PATH" "$DEPLOY_HOME/.ssh/authorized_keys"

if [[ ! -f "$DEPLOY_PATH/.env.prod" ]]; then
  install -m 640 -o "$DEPLOY_USER" -g "$DEPLOY_GROUP" /dev/null "$DEPLOY_PATH/.env.prod"
fi

echo "ProofForge deploy user and directories are ready:"
echo "  user: $DEPLOY_USER"
echo "  root: $DEPLOY_PATH"
