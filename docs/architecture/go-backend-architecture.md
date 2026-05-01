# Go backend архитектура MVP

## Архитектурный стиль
Для Go backend выбирается `feature-first modular monolith`.

Это означает:
- код группируется по доменным срезам
- внутри каждого среза есть разделение на domain / service / ports / adapters
- API и worker используют одни и те же доменные use cases

Не делаем:
- папку `utils` как свалку всего подряд
- business logic в HTTP handlers
- repository-driven architecture, где handler сам решает бизнес-переходы

## Предлагаемая структура

```text
backend/
  cmd/
    api/
    worker/
  internal/
    platform/
      config/
      logger/
      postgres/
      storage/
      jobs/
      httpx/
    auth/
    users/
    goals/
    pacts/
    invites/
    checkins/
    approvals/
    reports/
```

## Внутренний паттерн feature slice
Для каждого основного feature slice нужен явный набор файлов:
- `domain.go`
- `service.go`
- `ports.go`
- `postgres_repository.go`
- `http_handler.go`
- `service_test.go`
- `http_test.go`

Если slice разрастётся, допустимы дополнительные файлы, но только по ясной ответственности.

## Что где живёт

### domain.go
Содержит:
- сущности
- value objects
- инварианты
- status transitions
- доменные ошибки

Не содержит:
- SQL
- HTTP
- storage SDK calls

### service.go
Содержит:
- orchestration use cases
- транзакционные бизнес-flow
- coordination between repositories and storage ports

Не содержит:
- транспортную сериализацию
- framework-specific code

### ports.go
Содержит интерфейсы для:
- repositories
- storage
- jobs
- external services

### postgres_repository.go
Содержит:
- SQL queries
- persistence mapping
- tx-aware database operations

Не содержит:
- бизнес-решения по status transitions

### http_handler.go
Содержит:
- request parsing
- auth boundary checks
- вызов service layer
- response mapping

Не содержит:
- SQL
- сложные business branches
- принятие решений про approve/reject/request changes

## Boundary rules
- `context.Context` обязателен на границах service/repository/storage
- доступ к env только через `config` package
- domain status transitions определяются в domain/service, а не в handler/repository
- worker использует service layer, а не лезет напрямую в таблицы “по-быстрому”

## Почему worker отдельно
API должен обслуживать:
- goal creation
- invite acceptance
- proof submission
- review actions
- dashboard reads

Worker должен обслуживать:
- recap generation
- future reminders
- async proof processing

Такой split защищает request path от долгих фоновых задач и упрощает операционную диагностику.

## Архитектурные запреты
- business logic в handler
- package-level utils dump
- прямой вызов repository из handler там, где нужен service
- обход domain transitions “ради простоты”
- отдельные ad-hoc модели для API и worker, если речь об одном и том же доменном состоянии

## Что считать успехом
Go backend считается архитектурно здоровым, если:
- каждый feature slice можно понять независимо
- HTTP layer тонкий
- бизнес-правила читаются в domain/service
- storage и DB детали не протекают в доменную модель
- API и worker разделены по runtime-роли, но не дублируют доменную логику
