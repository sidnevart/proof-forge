#!/usr/bin/env bash
set -euo pipefail

DEPLOY_PATH="${DEPLOY_PATH:-/opt/proofforge-prod}"
CERT_DIR="$DEPLOY_PATH/nginx/certs"
CERT_FILE="$CERT_DIR/cert.pem"
KEY_FILE="$CERT_DIR/key.pem"
COMMON_NAME="${COMMON_NAME:-80.74.25.43}"
DEPLOY_OWNER="${DEPLOY_OWNER:-proofforge-deploy}"
DEPLOY_GROUP="${DEPLOY_GROUP:-$DEPLOY_OWNER}"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "This script must run as root." >&2
  exit 1
fi

install -d -m 750 -o "$DEPLOY_OWNER" -g "$DEPLOY_GROUP" "$CERT_DIR"

if [[ -f "$CERT_FILE" && -f "$KEY_FILE" ]]; then
  echo "Certificate already exists at $CERT_DIR"
  exit 0
fi

openssl req -x509 -nodes -newkey rsa:4096 \
  -days 365 \
  -keyout "$KEY_FILE" \
  -out "$CERT_FILE" \
  -subj "/CN=$COMMON_NAME"

chown "$DEPLOY_OWNER:$DEPLOY_GROUP" "$CERT_FILE" "$KEY_FILE"
chmod 640 "$CERT_FILE" "$KEY_FILE"

echo "Self-signed certificate generated for $COMMON_NAME"
