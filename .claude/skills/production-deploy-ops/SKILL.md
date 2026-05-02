---
name: production-deploy-ops
description: Operational runbook for ProofForge production deployment — SSH key setup, GitHub secrets, CI/CD pipeline, rolling deploy, and known failure modes.
---

## Purpose

Use this skill when you need to:
- Set up or repair SSH-based deployment from GitHub Actions to the VPS
- Add or rotate GitHub Actions secrets for the `production` environment
- Diagnose CI/CD failures (build, push, deploy, smoke check)
- Understand how ProofForge's rolling deploy works end-to-end

---

## Production Infrastructure

| Component | Value |
|-----------|-------|
| VPS IP | `80.74.25.43` |
| SSH user | `root` |
| SSH port | `22` |
| Deploy path | `/opt/proofforge-prod` |
| HTTPS port (external) | `18443` |
| HTTP port (external) | `18080` |
| Registry | `ghcr.io/sidnevart` |

Services run as Docker Compose: `postgres`, `redis`, `minio`, `minio-init`, `api`, `worker`, `web`, `nginx`.

---

## CI/CD Pipeline Structure

```
push to main
  └── .github/workflows/deploy.yml       # build images → push GHCR → SSH deploy → smoke check
  └── .github/workflows/ci.yml           # tests + lint (no push, no deploy)
```

`deploy.yml` triggers on:
- `push: branches: [main]` — automatic on every merge
- `workflow_dispatch` — manual trigger with optional `image_tag` override

### Required GitHub Secrets (environment: `production`)

| Secret | Description |
|--------|-------------|
| `DEPLOY_SSH_PRIVATE_KEY` | Private key content (ed25519) for SSH to VPS |
| `DEPLOY_HOST` | VPS IP or hostname |
| `DEPLOY_USER` | SSH user (`root`) |
| `DEPLOY_PORT` | SSH port (default `22`) |
| `DEPLOY_PATH` | Deploy directory (default `/opt/proofforge-prod`) |

---

## Setting Up Deployment from Scratch

### Step 1 — Generate SSH key pair (run locally)

```bash
ssh-keygen -t ed25519 -C "proofforge-deploy" -f ~/.ssh/proofforge_deploy -N ""
```

### Step 2 — Add public key to the server

```bash
cat ~/.ssh/proofforge_deploy.pub | ssh root@80.74.25.43 \
  "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"
```

Verify connection works:
```bash
ssh -i ~/.ssh/proofforge_deploy root@80.74.25.43 "echo OK"
```

### Step 3 — Add secrets to GitHub via `gh` CLI

```bash
gh secret set DEPLOY_SSH_PRIVATE_KEY --env production --repo sidnevart/proof-forge \
  < ~/.ssh/proofforge_deploy

gh secret set DEPLOY_HOST   --env production --repo sidnevart/proof-forge --body "80.74.25.43"
gh secret set DEPLOY_USER   --env production --repo sidnevart/proof-forge --body "root"
gh secret set DEPLOY_PORT   --env production --repo sidnevart/proof-forge --body "22"
gh secret set DEPLOY_PATH   --env production --repo sidnevart/proof-forge --body "/opt/proofforge-prod"
```

Verify all secrets are set:
```bash
gh secret list --env production --repo sidnevart/proof-forge
```

### Step 4 — Ensure `.env.prod` exists on the server

The deploy script (`deploy_compose.sh`) exits if `/opt/proofforge-prod/.env.prod` is missing.
Copy and fill in the template:
```bash
scp infra/docker/.env.prod.example root@80.74.25.43:/opt/proofforge-prod/.env.prod
ssh root@80.74.25.43 "nano /opt/proofforge-prod/.env.prod"   # fill real values
```

### Step 5 — First deploy (SSL cert required)

Before first deploy, generate self-signed cert on the server:
```bash
ssh root@80.74.25.43 "bash /opt/proofforge-prod/scripts/install_self_signed_cert.sh"
```

---

## Triggering a Deploy Manually

```bash
gh workflow run deploy.yml --repo sidnevart/proof-forge
```

With a specific image tag:
```bash
gh workflow run deploy.yml --repo sidnevart/proof-forge \
  -f image_tag=<commit-sha>
```

Watch the run:
```bash
gh run list --repo sidnevart/proof-forge --limit 3
gh run watch <run-id> --repo sidnevart/proof-forge
```

---

## Known Failure Modes

### Smoke check 502 after rolling deploy

**Symptom**: All containers healthy, but `curl http://127.0.0.1:18080/` returns 502.

**Cause**: Nginx caches upstream IPs at startup. When `api`/`web` containers are recreated, their IPs change but nginx keeps the old ones.

**Fix already applied**:
- `resolver 127.0.0.11 valid=10s ipv6=off` in nginx.conf + proxy via `$var` variables
- `docker compose restart nginx` in `deploy_compose.sh` after `up -d`

If it recurs, restart nginx manually:
```bash
ssh root@80.74.25.43 "cd /opt/proofforge-prod && docker compose --env-file .env.prod -f compose.prod.yml restart nginx"
```

### Deploy fails with `.env.prod is missing`

Create the file on the server from the example template (see Step 4 above).

### 401 after registration / login

**Cause A**: No login endpoint existed — users had to re-register which failed with `email_taken`.
**Fix applied**: `POST /v1/login` endpoint added (passwordless, by email).

**Cause B**: Cookie `Secure` flag — only sent over HTTPS. Confirm the user is on `https://80.74.25.43:18443`, not plain HTTP.

### minio-init Exited in `docker compose ps`

This is **expected** — `minio-init` is a one-shot init container. `Exited (0)` means success.

---

## Deploy Script Flow (`infra/scripts/server/deploy_compose.sh`)

```
1. Check .env.prod exists → exit 1 if missing
2. docker login ghcr.io
3. docker compose pull          → pull new images
4. docker compose up -d         → recreate changed containers
5. docker compose restart nginx → flush stale upstream IPs
6. smoke_check.sh               → curl healthz + readyz + root
```

---

## Smoke Check (`infra/scripts/server/smoke_check.sh`)

```bash
curl http://127.0.0.1:18080/             # HTTP (redirects to HTTPS)
curl -k https://127.0.0.1:18443/         # HTTPS root (self-signed, -k skips cert check)
curl -k https://127.0.0.1:18443/readyz   # API readiness probe
```

---

## Checking Logs on Server

```bash
ssh root@80.74.25.43 "cd /opt/proofforge-prod && \
  docker compose --env-file .env.prod -f compose.prod.yml logs --tail=50 api"
```

Replace `api` with `web`, `nginx`, `worker`, `postgres` as needed.
