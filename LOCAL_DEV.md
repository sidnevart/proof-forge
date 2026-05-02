# Локальный запуск ProofForge

Всё поднимается одной командой через Docker Compose. Нужен только Docker Desktop.

## Быстрый старт

```bash
# 1. Склонить репо
git clone git@github.com:sidnevart/proof-forge.git
cd proof-forge

# 2. Поднять всё
docker compose -f infra/docker/compose.dev.yml up --build -d

# 3. Дождаться готовности (30–60 секунд)
docker compose -f infra/docker/compose.dev.yml ps
```

После запуска:

| Сервис | URL | Назначение |
|--------|-----|-----------|
| **Web (фронт)** | http://localhost:3003 | Основной интерфейс |
| **API** | http://localhost:8080 | Backend API |
| **MinIO Console** | http://localhost:59001 | S3 хранилище (login: `proofforge` / `proofforge-secret`) |
| **PostgreSQL** | localhost:55432 | БД (user: `proofforge`, pass: `proofforge`, db: `proofforge`) |
| **Redis** | localhost:56379 | Кэш |

## Что делать в браузере

1. Открой http://localhost:3003
2. **Зарегистрируйся** — введи имя и email (любые, это dev)
3. **Создай цель** — нажми "Создать первую цель", заполни:
   - Название цели
   - Описание (что считается прогрессом)
   - Имя и email партнёра (buddy)
4. **Прими инвайт за buddy** — зарегистрируй второго юзера с тем email, который указал как buddy. Перейди по ссылке из логов API (или найди токен в БД)
5. **После принятия — цель станет активной**, появится секция "Ставки"
6. **Добавь ставку** — напиши что ставишь на кон
7. **Залогинься за buddy** — он увидит кнопку "Ставка сгорела"

### Быстрый путь: принять инвайт через curl

```bash
# Найти токен инвайта
docker exec proofforge-postgres-dev psql -U proofforge -d proofforge \
  -c "SELECT acceptance_token FROM invites WHERE status = 'pending' LIMIT 1;"

# Зарегистрировать buddy (подставь email который указал при создании цели)
curl -X POST http://localhost:8080/v1/register \
  -H 'Content-Type: application/json' \
  -d '{"email": "buddy@example.com", "display_name": "Buddy"}' \
  -c /tmp/pf-buddy-cookies

# Принять инвайт (подставь токен)
curl -X POST http://localhost:8080/v1/invites/ТОКЕН_СЮДА/accept \
  -b /tmp/pf-buddy-cookies
```

## Полезные команды

```bash
# Логи всех сервисов
docker compose -f infra/docker/compose.dev.yml logs -f

# Логи только API
docker compose -f infra/docker/compose.dev.yml logs -f api

# Перебилдить после правок кода
docker compose -f infra/docker/compose.dev.yml up --build -d

# Остановить всё
docker compose -f infra/docker/compose.dev.yml down

# Остановить и удалить данные (чистый старт)
docker compose -f infra/docker/compose.dev.yml down -v

# Зайти в БД
docker exec -it proofforge-postgres-dev psql -U proofforge -d proofforge

# Посмотреть ставки в БД
docker exec proofforge-postgres-dev psql -U proofforge -d proofforge \
  -c "SELECT id, description, status FROM stakes;"
```

## Архитектура локального стека

```
Browser (http://localhost:3003)
  │
  ├── Next.js (web)        :3003  — UI
  │     └── fetch ──────────────────┐
  │                                 ▼
  ├── Go API (api)         :8080  — REST API + миграции
  ├── Go Worker (worker)          — фоновые задачи (рекапы)
  ├── PostgreSQL           :55432 — основная БД
  ├── Redis                :56379 — кэш
  └── MinIO (S3)           :59000 — файловое хранилище
```

## Если что-то не работает

**Web не запускается / билд падает:**
```bash
docker compose -f infra/docker/compose.dev.yml logs web
```
Чаще всего — ошибка npm ci. Проверь что `web/package-lock.json` в порядке.

**API не стартует:**
```bash
docker compose -f infra/docker/compose.dev.yml logs api
```
Обычно — postgres ещё не готов. Подожди и перезапусти: `docker compose -f infra/docker/compose.dev.yml restart api`

**CORS ошибки в браузере:**
Убедись что `WEB_ORIGIN=http://localhost:3003` в `.env.dev`
