# Runbook: isolated VPS deploy ProofForge

## Что уже предполагается
- сервер доступен по SSH
- на сервере установлен Docker с Compose plugin
- у вас есть root-доступ для первичной подготовки
- existing project на VPS не должен быть затронут

## Целевые пути
- deploy root: `/opt/proofforge-prod`
- deploy user: `proofforge-deploy`
- compose project: `proofforge-prod`

## Что заполнить
Реальный production env нужно хранить в:

`/opt/proofforge-prod/.env.prod`

Заполнять его по шаблону:

`infra/docker/.env.prod.example`

Минимум нужны:
- `POSTGRES_PASSWORD`
- `S3_ACCESS_KEY_ID`
- `S3_SECRET_ACCESS_KEY`
- `S3_BUCKET`
- `OPENAI_API_KEY`
- `SMTP_HOST`
- `SMTP_PORT`
- `SMTP_USERNAME`
- `SMTP_PASSWORD`
- `SMTP_FROM`

Также проверить:
- `WEB_ORIGIN=https://80.74.25.43:18443`
- `NEXT_PUBLIC_API_BASE_URL=https://80.74.25.43:18443`

## Первичная подготовка сервера
1. Сгенерировать SSH key для GitHub Actions.
2. Запустить `infra/scripts/server/bootstrap_production_host.sh` от root.
3. Запустить `infra/scripts/server/install_self_signed_cert.sh` от root.
4. Заполнить `/opt/proofforge-prod/.env.prod`.

## Как работает deploy
`deploy.yml`:
- вручную запускается из GitHub Actions
- собирает `api`, `worker`, `web`
- пушит их в `ghcr.io`
- копирует compose/nginx/scripts на сервер
- делает remote `docker compose pull && up -d`
- запускает smoke checks

## Smoke checks
Проверить:
- `curl http://80.74.25.43:18080/`
- `curl -k https://80.74.25.43:18443/`
- `curl -k https://80.74.25.43:18443/readyz`

Для браузерной проверки:
- открыть `https://80.74.25.43:18443`
- принять self-signed certificate warning
- пройти регистрацию
- убедиться, что dashboard открывается
