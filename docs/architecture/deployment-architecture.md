# Архитектура деплоя ProofForge

## Назначение
Этот документ фиксирует production-like deployment model для ProofForge на общем VPS, где уже работает другой проект.

## Базовая схема
ProofForge деплоится как **полностью отдельный docker compose stack**:
- отдельный deploy root: `/opt/proofforge-prod`
- отдельный compose project: `proofforge-prod`
- отдельные контейнеры `web`, `api`, `worker`, `postgres`, `redis`, `minio`, `nginx`
- отдельные volumes и internal network

## Почему не используем host nginx
На сервере уже занят и используется host-level `nginx` на `80/443`.  
Чтобы не затронуть существующий проект:
- не меняем его vhost-ы
- не публикуем новый stack на `80/443`
- не проксируемся через текущий routing слой в первой итерации

## Внешний вход
Поскольку домена пока нет, ProofForge публикуется через:
- `http://80.74.25.43:18080`
- `https://80.74.25.43:18443`

HTTP используется только как redirect. Реальный app flow должен идти через HTTPS, потому что session cookie помечается как `Secure`.

## Артефактный путь
CI собирает и публикует Docker images в `ghcr.io`.
Сервер не делает `git pull` и не строит production images из исходников.

Сервер:
- получает `compose.prod.yml`
- получает nginx config и deploy scripts
- делает `docker compose pull`
- поднимает нужный tag образов

## Storage и data isolation
MinIO остаётся на том же VPS, но не переиспользуется из другого проекта.

У ProofForge свои:
- bucket
- credentials
- container
- volume

То же правило действует для Postgres и Redis.
