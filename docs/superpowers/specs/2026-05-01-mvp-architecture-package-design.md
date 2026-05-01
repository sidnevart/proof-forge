# Дизайн MVP architecture package

## Цель
Определить архитектурный пакет для первой версии ProofForge до начала кодовой реализации.

## Выбранный системный подход
Для MVP выбирается:
- web app
- Go API
- Go worker
- PostgreSQL
- object storage для proof artifacts

Без Telegram в первой версии.

## Основные продуктовые ограничения
- buddy обязателен
- один check-in относится к одному goal
- proof включает text, link и обязательную поддержку file/image upload
- review flow: approve / request changes / reject
- AI не approve proof
- buddy approval остаётся источником правды

## Архитектурный стиль backend
- feature-first modular monolith
- отдельный worker-процесс
- domain/service/ports/adapters separation
- без business logic в handlers

## Основные доменные сущности
- User
- Goal
- Pact
- Invite
- CheckIn
- Evidence
- ReviewDecision
- WeeklyRecap

## Основные статусы
- Goal: pending_buddy_acceptance, active, paused, completed, archived
- Pact: invited, active, ended
- CheckIn: draft, submitted, changes_requested, approved, rejected
- WeeklyRecap: queued, generated, failed

## Основные документы
- `docs/architecture/system-architecture.md`
- `docs/architecture/go-backend-architecture.md`
- `docs/architecture/domain-model.md`
- `docs/architecture/status-transitions.md`
- `docs/architecture/database-schema.md`
- `docs/api/openapi-draft.md`
