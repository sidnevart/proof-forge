# Docker

`compose.local.yml` — только Postgres для локальной разработки (legacy).

`compose.dev.yml` — полный локальный стек:
- Postgres / Redis / MinIO / API / Worker / Web
- образы собираются локально из `backend/` и `web/`
- web на порту `3003`, api на `8080`
- запуск: `../../scripts/local/start_dev.sh`

`compose.prod.yml` — isolated production-like stack для VPS:
- отдельный `docker compose` project
- свой Postgres / Redis / MinIO / Nginx
- публикация только через high ports `18080` и `18443`
- образы тянутся из `ghcr.io`, а не собираются на сервере

`/.env.dev.example` — шаблон для локального `.env.dev`.
`/.env.prod.example` — шаблон для серверного `.env.prod`. Реальный `.env.prod`
не коммитится и живёт только на сервере.
