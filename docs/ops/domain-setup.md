# Domain + HTTPS setup for ProofForge

This is the one-time runbook for pointing your purchased domain at the VPS,
issuing a real Let's Encrypt certificate, and switching the public listener
from `:18080`/`:18443` (self-signed) to standard `80`/`443` (real cert).

You execute the SSH steps yourself. Each step says where it runs (your
laptop ⇒ DNS panel ⇒ VPS shell ⇒ browser).

---

## Prerequisites

| Thing | Value |
|------|-------|
| Domain | _to fill in_ — e.g. `proofforge.example.com` |
| VPS IP | _to fill in_ — the public IPv4 of your droplet |
| Domain registrar | wherever you bought the domain |
| VPS user with sudo | `ubuntu`/`root`/your deploy user |
| Email for ACME | a real address — Let's Encrypt sends expiry warnings here |
| Repo deployed at | `/opt/proofforge-prod` (default `DEPLOY_PATH` from CI) |

Replace placeholders below as you go:
- `<DOMAIN>` ⇒ your domain (e.g. `app.proofforge.io`)
- `<VPS_IP>` ⇒ your VPS public IP
- `<EMAIL>` ⇒ your contact email

---

## 1 · DNS — point the domain at the VPS

**Where:** your registrar's DNS panel.

Add an `A` record:

| Type | Host / Name | Value | TTL |
|------|-------------|-------|-----|
| A | `@` (or the subdomain you want, e.g. `app`) | `<VPS_IP>` | `300` |

If you want both `proofforge.example.com` and `www.proofforge.example.com`,
add two A records (or use a CNAME for `www`).

**Verify** from your laptop:

```bash
dig +short <DOMAIN>
```

Should print `<VPS_IP>`. If it prints nothing, wait 1–10 minutes for
propagation and try again.

---

## 2 · Firewall on the VPS

**Where:** SSH on the VPS as a sudo user.

```bash
sudo ufw status                # confirm ufw is active; if "Status: inactive" enable it
sudo ufw allow 22/tcp          # SSH (skip if already there)
sudo ufw allow 80/tcp          # HTTP (ACME http-01 + redirect to HTTPS)
sudo ufw allow 443/tcp         # HTTPS
sudo ufw reload
```

If you previously opened the high ports (`18080`, `18443`), you can leave
them in place during the cutover and remove them once you confirm the new
domain works:

```bash
# After cutover only:
sudo ufw delete allow 18080/tcp
sudo ufw delete allow 18443/tcp
```

---

## 3 · Stop the current nginx (frees ports 80/443)

The certbot `--standalone` plugin needs port 80 to itself for the first
issuance. The existing prod nginx listens on `18080`/`18443` so it usually
isn't binding 80 — but stopping it is the simplest way to make sure.

**Where:** SSH on the VPS.

```bash
cd /opt/proofforge-prod
docker compose -f compose.prod.yml down nginx || true

# Sanity check — nothing should be listening on :80 / :443:
sudo ss -ltnp | grep -E ':80\b|:443\b' || echo "ports free"
```

---

## 4 · Issue the cert (Let's Encrypt, standalone challenge)

**Where:** SSH on the VPS, sudo.

```bash
sudo apt update
sudo apt install -y certbot

sudo certbot certonly --standalone \
  --preferred-challenges http \
  -d <DOMAIN> \
  --agree-tos \
  -m <EMAIL> \
  --non-interactive
```

Successful output ends with:

```
Successfully received certificate.
Certificate is saved at: /etc/letsencrypt/live/<DOMAIN>/fullchain.pem
Key is saved at:         /etc/letsencrypt/live/<DOMAIN>/privkey.pem
```

If you get `Connection refused`, the firewall didn't actually open 80
(re-check Step 2) or DNS hasn't propagated (re-check Step 1).

Make sure the directory used by the renewal HTTP-01 webroot exists:

```bash
sudo mkdir -p /var/www/certbot
```

---

## 5 · Update `.env.prod` on the VPS

**Where:** SSH on the VPS.

```bash
sudo -e /opt/proofforge-prod/.env.prod
```

Set / replace these keys:

```
SERVER_NAME=<DOMAIN>
WEB_ORIGIN=https://<DOMAIN>
NEXT_PUBLIC_API_BASE_URL=https://<DOMAIN>
COOKIE_DOMAIN=<DOMAIN>
SESSION_TTL=15m
REFRESH_TTL=720h
APP_ENV=production
```

Save and exit. (`SERVER_NAME` is what the new nginx config substitutes into
`server_name` and the cert paths. `COOKIE_DOMAIN` pins both auth cookies
to your apex so they don't get dropped on redirect.)

---

## 6 · Pick up the new code and restart the stack

**Where:** your laptop (push), then SSH on VPS (deploy).

The repo changes that need to be deployed:
- new nginx template `infra/nginx/nginx.prod.conf` (replaces high-port one)
- `compose.prod.yml` now opens `80`/`443` and mounts `/etc/letsencrypt`
- `.env.prod.example` documents the new variables
- backend supports `COOKIE_DOMAIN`, `REFRESH_TTL`, `REFRESH_COOKIE_NAME`
- new endpoints `POST /v1/auth/refresh`, `POST /v1/auth/logout`

Push to `main` (or trigger the deploy workflow manually from the Actions
tab). The CI pipeline does the SCP + remote `docker compose pull && up -d`
for you.

If you'd rather deploy manually, on the VPS:

```bash
cd /opt/proofforge-prod
docker compose -f compose.prod.yml pull
docker compose -f compose.prod.yml up -d
```

---

## 7 · Smoke check

**Where:** your laptop.

```bash
# HTTPS works and serves the API:
curl -I https://<DOMAIN>/healthz
# Expect: HTTP/2 200

# HTTP redirects to HTTPS:
curl -I http://<DOMAIN>/
# Expect: 301 Moved Permanently, Location: https://<DOMAIN>/

# The cert chain is real (no -k!):
curl -v https://<DOMAIN>/ 2>&1 | grep -E "subject:|issuer:|verify"
# Expect: issuer: ...Let's Encrypt..., verify: SSL certificate verify ok
```

In a browser:
1. Open `https://<DOMAIN>` → no padlock warnings, page loads.
2. Sign in / register.
3. Open DevTools → Application → Cookies → `https://<DOMAIN>`.
   - `pf_session`: `Domain=<DOMAIN>`, `HttpOnly`, `Secure`, `SameSite=Lax`.
   - `pf_refresh`: `Path=/v1/auth`, `HttpOnly`, `Secure`, `SameSite=Lax`.
4. Wait ~15 minutes (or temporarily set `SESSION_TTL=2m` in `.env.prod`,
   redeploy, log in, wait three minutes). Refresh the page → `pf_session`
   silently rotates, the user stays logged in.

---

## 8 · Auto-renewal cron

Let's Encrypt certs expire in 90 days. Schedule renewal twice a day; the
certbot CLI no-ops when the cert isn't close enough to expiry.

**Where:** SSH on the VPS, sudo.

```bash
sudo tee /etc/cron.d/certbot-renew <<'CRON'
# Renew Let's Encrypt certs daily at 03:17 (random-ish minute to avoid the
# global Let's Encrypt herd that hits :00). Reload nginx in the running
# proofforge-prod stack so the new cert takes effect without downtime.
17 3 * * * root certbot renew --quiet --webroot -w /var/www/certbot --deploy-hook "docker compose -f /opt/proofforge-prod/compose.prod.yml exec -T nginx nginx -s reload"
CRON
```

Test the renewal hook (no actual renewal happens, just validates flow):

```bash
sudo certbot renew --dry-run
```

---

## 9 · Cleanup (optional, after the new domain is verified)

Remove the old self-signed material and high-port firewall rules:

```bash
# Old self-signed certs (only if they exist):
sudo rm -rf /opt/proofforge-prod/nginx/certs

# Old firewall openings:
sudo ufw delete allow 18080/tcp || true
sudo ufw delete allow 18443/tcp || true
```

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `curl https://<DOMAIN>` → cert error | nginx still using old cert | `docker compose -f compose.prod.yml restart nginx` |
| `503` from nginx | `api`/`web` containers not ready | `docker compose ps` → check unhealthy services |
| Login works once, refresh page asks to log in again | `COOKIE_DOMAIN` mismatch | confirm `.env.prod` has `COOKIE_DOMAIN=<DOMAIN>` (no scheme, no port), then redeploy |
| `certbot: connection refused` | port 80 not open OR DNS not propagated | `dig +short <DOMAIN>`; `sudo ufw status` |
| nginx fails to start: `cannot load certificate` | `SERVER_NAME` doesn't match cert dir | the cert was issued for a different name; re-run certbot for `<DOMAIN>` exactly |
| Telegram bot stops getting webhooks | webhook still pointing at old URL | re-register: `curl "https://api.telegram.org/bot$TOKEN/setWebhook?url=https://<DOMAIN>/v1/telegram/webhook"` (skip if you don't use the bot yet) |
