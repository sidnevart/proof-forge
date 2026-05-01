# ProofForge VPS Deploy Runbook

Target: single Ubuntu 24.04 LTS VPS, Docker Compose stack.  
Stack: nginx → Next.js web + Go API + Go worker + PostgreSQL + Redis + MinIO.

---

## 1. Server Provisioning

### Minimum specs
- 2 vCPU, 4 GB RAM, 40 GB SSD (8 GB RAM recommended for AI recap workloads)
- Ubuntu 24.04 LTS
- Open ports: 22 (SSH), 80 (HTTP redirect), 443 (HTTPS)

### First login
```bash
ssh root@YOUR_SERVER_IP

# Create deploy user
adduser deploy
usermod -aG sudo docker deploy   # docker group added after Docker install
rsync --archive --chown=deploy:deploy ~/.ssh /home/deploy/.ssh
```

### Install Docker
```bash
curl -fsSL https://get.docker.com | sh
usermod -aG docker deploy
# Log out and back in as deploy, then verify:
docker run --rm hello-world
docker compose version   # must be >= 2.20
```

### Install Certbot (Let's Encrypt)
```bash
apt install -y certbot
```

---

## 2. DNS

Point two A records at the server IP before TLS provisioning:

| Record | Value |
|--------|-------|
| `yourdomain.com` | VPS IP |
| `api.yourdomain.com` | VPS IP (optional — can share one domain) |

Wait for DNS propagation (`dig yourdomain.com +short`) before proceeding.

---

## 3. TLS Certificate

```bash
# Stop anything on port 80 first (nothing should be running yet)
certbot certonly --standalone \
  -d yourdomain.com \
  --non-interactive --agree-tos \
  -m admin@yourdomain.com

# Symlink certs into the nginx certs directory
mkdir -p /opt/proofforge/infra/nginx/certs
ln -sf /etc/letsencrypt/live/yourdomain.com/fullchain.pem \
       /opt/proofforge/infra/nginx/certs/cert.pem
ln -sf /etc/letsencrypt/live/yourdomain.com/privkey.pem \
       /opt/proofforge/infra/nginx/certs/key.pem
```

### Auto-renewal
```bash
# Test renewal works
certbot renew --dry-run

# Cron: renew at 02:30 on the 1st and 15th of each month, then reload nginx
echo "30 2 1,15 * * root certbot renew --quiet && docker compose -f /opt/proofforge/infra/docker/compose.prod.yml exec nginx nginx -s reload" \
  > /etc/cron.d/certbot-renew
```

---

## 4. Repository Checkout

```bash
su - deploy
git clone https://github.com/sidnevart/proof-forge.git /opt/proofforge
cd /opt/proofforge
```

---

## 5. Environment Configuration

```bash
cd /opt/proofforge/infra/docker
cp .env.prod.example .env.prod
chmod 600 .env.prod
$EDITOR .env.prod
```

Required values to fill in `.env.prod`:

| Variable | What to set |
|----------|-------------|
| `POSTGRES_PASSWORD` | Strong random password (use `openssl rand -hex 32`) |
| `S3_ACCESS_KEY_ID` | MinIO root user (16+ chars) |
| `S3_SECRET_ACCESS_KEY` | MinIO root password (`openssl rand -hex 32`) |
| `WEB_PUBLIC_API_URL` | `https://yourdomain.com` |
| `WEB_ORIGIN` | `https://yourdomain.com` |
| `NEXT_PUBLIC_API_BASE_URL` | `https://yourdomain.com` |
| `OPENAI_API_KEY` | `sk-...` (leave empty to disable AI recaps) |
| `TELEGRAM_BOT_TOKEN` | From @BotFather (leave as `change-me` if Telegram is not yet wired) |
| `TELEGRAM_WEBHOOK_SECRET` | `openssl rand -hex 20` |

Verify no placeholder values remain:
```bash
grep -E "change-me|example\.com|sk-\.\.\." .env.prod && echo "STOP: unfilled placeholders"
```

---

## 6. First Deploy

```bash
cd /opt/proofforge/infra/docker

# Build images (takes 3–5 min on first run)
docker compose -f compose.prod.yml build

# Start infrastructure first, then app
docker compose -f compose.prod.yml up -d postgres redis minio
sleep 15

# Verify postgres and redis are healthy
docker compose -f compose.prod.yml ps

# Start API (runs DB migrations on boot via DB_RUN_MIGRATIONS=true)
docker compose -f compose.prod.yml up -d api
# Wait for healthy
docker compose -f compose.prod.yml ps api
# Should show "healthy" within 30s

# Start remaining services
docker compose -f compose.prod.yml up -d worker web nginx
docker compose -f compose.prod.yml ps
```

All 7 services (`postgres`, `redis`, `minio`, `minio-init`, `api`, `worker`, `web`, `nginx`) must show `running` or `exited (0)` for minio-init.

---

## 7. Subsequent Rollouts

```bash
cd /opt/proofforge

# Pull latest code
git pull --ff-only

# Rebuild changed images
cd infra/docker
docker compose -f compose.prod.yml build api worker web

# Rolling restart — keeps postgres/redis/minio up
docker compose -f compose.prod.yml up -d --no-deps api
# Wait for healthy before touching worker and web
sleep 20
docker compose -f compose.prod.yml ps api

docker compose -f compose.prod.yml up -d --no-deps worker
docker compose -f compose.prod.yml up -d --no-deps web
docker compose -f compose.prod.yml up -d --no-deps nginx
```

Tag the deployed commit:
```bash
git tag deploy-$(date +%Y%m%d-%H%M) && git push --tags
```

---

## 8. Rollback

```bash
cd /opt/proofforge/infra/docker

# Find previous working image tag (or use a git tag)
git log --oneline -5

# Hard rollback to a tagged deploy
git checkout deploy-YYYYMMDD-HHMM
docker compose -f compose.prod.yml build api worker web
docker compose -f compose.prod.yml up -d --no-deps api worker web nginx
```

**Database migrations**: `goose` migrations are forward-only in this codebase.  
If a migration shipped with the bad release, restore from backup before rolling back:
```bash
# Restore from last backup (see backup procedure)
docker compose -f compose.prod.yml stop api worker
# restore postgres volume from backup
docker compose -f compose.prod.yml start api worker
```

---

## 9. Logs and Observability

```bash
# Tail all services
docker compose -f compose.prod.yml logs -f

# API only
docker compose -f compose.prod.yml logs -f api

# Worker (shows recap sweep results)
docker compose -f compose.prod.yml logs -f worker

# Check nginx access log for 5xx
docker compose -f compose.prod.yml exec nginx tail -f /var/log/nginx/access.log | grep '" 5'
```

Structured JSON logs from api/worker: search by `"level":"error"` or `"source":"recaps"`.

---

## 10. Postgres Backup

```bash
# Manual backup
docker compose -f compose.prod.yml exec postgres \
  pg_dump -U proofforge proofforge | gzip > /opt/backups/proofforge-$(date +%Y%m%d).sql.gz

# Cron: daily at 03:00
mkdir -p /opt/backups
echo "0 3 * * * deploy docker compose -f /opt/proofforge/infra/docker/compose.prod.yml exec -T postgres pg_dump -U proofforge proofforge | gzip > /opt/backups/proofforge-\$(date +\%Y\%m\%d).sql.gz" \
  > /etc/cron.d/proofforge-backup
```

Keep at least 7 days of backups. Rotate with:
```bash
find /opt/backups -name "proofforge-*.sql.gz" -mtime +7 -delete
```

---

## 11. Firewall

```bash
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
ufw status
```

PostgreSQL, Redis, and MinIO ports must NOT be open externally — they are Docker-internal only.
