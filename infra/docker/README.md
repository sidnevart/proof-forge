# Docker

`compose.local.yml` — локальный development Postgres.

`compose.prod.yml` — isolated production-like stack для VPS:
- отдельный `docker compose` project
- свой Postgres / Redis / MinIO / Nginx
- публикация только через high ports `18080` и `18443`
- образы тянутся из `ghcr.io`, а не собираются на сервере

`/.env.prod.example` — шаблон для серверного `.env.prod`. Реальный `.env.prod`
не коммитится и живёт только на сервере.
