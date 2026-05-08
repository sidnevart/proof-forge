# ProofForge

**ProofForge** — приложение для достижения целей через систему взаимной ответственности. Вы ставите цель, берёте «бадди» (партнёра по ответственности), регулярно сдаёте доказательства прогресса, а бадди их одобряет. Всё работает внутри кругов — небольших групп людей с общими целями.

---

## Возможности

- **Цели с бадди** — каждая цель требует партнёра, который одобряет ваши чекины
- **Круги** — группы до 8 человек, сезоны длиной 7 дней, общая лента прогресса
- **Доказательства** — текст, ссылки, загрузка файлов (S3/MinIO)
- **Лента** — публичная лента успехов своего круга и похожих целей
- **AI-помощник** — уточняет формулировку цели при создании (OpenAI, опционально)
- **Telegram-бот** — уведомления и напоминания (опционально)
- **Еженедельные рекапы** — автоматический сводный отчёт по неделе
- **Мобильный UI** — адаптивный интерфейс с нижним навбаром

---

## Стек

| Слой | Технологии |
|---|---|
| Backend | Go 1.24, chi, pgx v5, Goose (миграции) |
| Frontend | Next.js 15 (App Router), TypeScript, CSS Modules |
| БД | PostgreSQL 16 + pgvector |
| Кэш / очередь | Redis 7 |
| Хранилище файлов | MinIO (S3-совместимое) |
| Контейнеры | Docker + Compose |
| CI/CD | GitHub Actions |

---

## Быстрый старт (локально)

### Требования
- Docker Desktop / Docker Engine с Compose plugin
- Go 1.24+ (только для разработки без Docker)
- Node.js 22+ (только для разработки без Docker)

### 1. Клонировать репозиторий

```bash
git clone https://github.com/sidnevart/proof-forge.git
cd proof-forge
```

### 2. Создать dev-окружение

```bash
cp infra/docker/.env.dev.example infra/docker/.env.dev
# Отредактируйте .env.dev при необходимости (MinIO, Telegram, OpenAI — опционально)
```

### 3. Запустить полный стек

```bash
docker compose -f infra/docker/compose.dev.yml --env-file infra/docker/.env.dev up --build -d
```

Или через скрипт:

```bash
bash scripts/local/start_dev.sh
```

### 4. Загрузить моковые данные (опционально)

```bash
docker compose -f infra/docker/compose.dev.yml exec -T postgres \
  psql -U proofforge -d proofforge < infra/docker/seed.sql
```

### 5. Открыть приложение

| Сервис | URL |
|---|---|
| **Приложение** | http://localhost:3003 |
| API | http://localhost:8080 |
| MinIO Console | http://localhost:59001 (логин: `proofforge` / `proofforge123`) |

---

## Структура проекта

```
proof-forge/
├── backend/          # Go API-сервер и воркер
│   ├── cmd/api/      # Точка входа API
│   ├── cmd/worker/   # Точка входа воркера (рекапы, уведомления)
│   ├── internal/     # Доменная логика (goals, circles, checkins, …)
│   └── migrations/   # SQL-миграции (Goose)
├── web/              # Next.js 15 фронтенд
│   ├── app/          # App Router — страницы
│   └── components/   # UI-компоненты
├── infra/
│   ├── docker/       # Compose-файлы и env-шаблоны
│   ├── nginx/        # Конфиг nginx для prod
│   └── scripts/      # Скрипты деплоя
└── docs/             # Архитектурные решения, ранбуки
```

---

## Переменные окружения

Полный список с описанием — в `.env.example` (корень) и `infra/docker/.env.dev.example`.

Обязательные для запуска:

| Переменная | Описание |
|---|---|
| `DATABASE_URL` | PostgreSQL DSN |
| `APP_NAME` | Идентификатор приложения |
| `WEB_ORIGIN` | URL фронтенда (для CORS) |

Опциональные (функции отключаются если не заданы):

| Переменная | Функция |
|---|---|
| `OPENAI_API_KEY` | AI-уточнение целей и рекапы |
| `TELEGRAM_BOT_TOKEN` | Telegram-бот |
| `S3_BUCKET` | Загрузка файлов как доказательств |
| `SMTP_HOST` | Email-уведомления |

---

## Деплой в production

Подробный ранбук: [`docs/runbooks/vps-deploy-proof-forge.md`](docs/runbooks/vps-deploy-proof-forge.md)

Краткая схема:
1. Настроить VPS, SSH-ключ для GitHub Actions
2. Запустить `infra/scripts/server/bootstrap_production_host.sh`
3. Заполнить `/opt/proofforge-prod/.env.prod` по шаблону `infra/docker/.env.prod.example`
4. Запустить деплой из GitHub Actions (`deploy.yml`)

---

## Лицензия

MIT — см. [LICENSE](LICENSE)
