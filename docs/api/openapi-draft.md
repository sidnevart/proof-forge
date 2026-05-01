# Черновик OpenAPI для MVP

## Назначение
Этот документ фиксирует первый черновой API-контракт для web-only MVP. Это ещё не финальный OpenAPI YAML, а архитектурный draft, который задаёт ресурсную модель, payload boundaries и основные endpoint groups.

## Что реализовано первым vertical slice
На 1 мая 2026 года в кодовую базу для первого product slice входят:
- `POST /v1/register`
- `GET /v1/me`
- `POST /v1/goals`
- `GET /v1/goals`
- `GET /v1/dashboard`

Остальные endpoint groups ниже остаются архитектурным планом, а не уже реализованным контрактом.

## Базовые принципы
- API versioning через `/v1`
- JSON для metadata и доменных действий
- file/image proof проходит через upload flow
- API не позволяет owner self-approve check-in
- approved progress строится только на buddy review

## Основные ресурсы
- `users`
- `goals`
- `invites`
- `pacts`
- `check-ins`
- `evidence`
- `reviews`
- `weekly-recaps`

## Endpoint groups

### Auth / Session
- `POST /v1/register`
- `GET /v1/me`

### Goals
- `POST /v1/goals`
- `GET /v1/goals`
- `GET /v1/goals/{goalId}`
- `PATCH /v1/goals/{goalId}`
- `POST /v1/goals/{goalId}/archive`

### Invites / Buddy onboarding
- `POST /v1/goals/{goalId}/invite`
- `GET /v1/invites/{token}`
- `POST /v1/invites/{token}/accept`
- `POST /v1/invites/{token}/decline`

### Check-ins
- `POST /v1/goals/{goalId}/check-ins`
- `GET /v1/goals/{goalId}/check-ins`
- `GET /v1/check-ins/{checkInId}`
- `POST /v1/check-ins/{checkInId}/submit`
- `POST /v1/check-ins/{checkInId}/resubmit`

### Evidence
- `POST /v1/check-ins/{checkInId}/evidence/text`
- `POST /v1/check-ins/{checkInId}/evidence/link`
- `POST /v1/check-ins/{checkInId}/evidence/file-upload-url`
- `POST /v1/check-ins/{checkInId}/evidence/file-complete`
- `DELETE /v1/check-ins/{checkInId}/evidence/{evidenceId}`

### Reviews
- `POST /v1/check-ins/{checkInId}/approve`
- `POST /v1/check-ins/{checkInId}/request-changes`
- `POST /v1/check-ins/{checkInId}/reject`
- `GET /v1/check-ins/{checkInId}/reviews`

### Dashboard / Recaps
- `GET /v1/dashboard`
- `GET /v1/goals/{goalId}/weekly-recaps`
- `GET /v1/weekly-recaps/{recapId}`

## Session model
- web frontend работает через `HttpOnly` cookie
- успешная регистрация сразу создаёт session
- protected endpoints требуют валидную session cookie
- API по-прежнему разделяет auth boundary и domain actions; handlers не должны подменять это `X-User-ID` shortcut-ами

## Пример payload boundaries

### POST /v1/register
Request:
- `email`
- `display_name`

Response:
- `user`

Side effect:
- создаёт session cookie

### POST /v1/goals
Request:
- `title`
- `description`
- `buddy_name`
- `buddy_email`

Response:
- `goal`

Поведение:
- goal создаётся только вместе с обязательным buddy
- новый goal стартует в `pending_buddy_acceptance`
- одновременно создаются `pact` со статусом `invited` и `invite` со статусом `pending`

### GET /v1/goals
Response:
- `goals[]`

### GET /v1/dashboard
Response:
- `user`
- `summary`
- `goals[]`

`summary` в первом slice включает:
- `total_goals`
- `pending_buddy_acceptance`
- `active_goals`

### POST /v1/goals/{goalId}/check-ins
Request:
- optional `note`

Response:
- `check_in`

### POST /v1/check-ins/{checkInId}/approve
Request:
- optional `comment`

Response:
- `check_in`
- `goal_progress`

### POST /v1/check-ins/{checkInId}/request-changes
Request:
- required `comment`

Response:
- `check_in`
- `review`

### POST /v1/check-ins/{checkInId}/reject
Request:
- required `comment`

Response:
- `check_in`
- `review`

## Error model
Минимальный error envelope:

```json
{
  "error": {
    "code": "invalid_state_transition",
    "message": "Check-in cannot be approved from current state",
    "details": {}
  }
}
```

Коды ошибок нужны минимум для:
- auth required
- forbidden actor
- invite expired
- goal not active
- invalid state transition
- evidence missing
- upload not completed
- resource not found

## Важные правила API
- owner не может approve/request changes/reject свой check-in
- buddy не может review check-in по goal, к которому он не привязан
- check-in нельзя submit без meaningful evidence
- reject не должен открывать путь к повторной submit того же check-in
- request changes должен сохранять review loop живым
- registration не должна превращаться в временный dev shortcut без session boundary
- goal creation без `buddy_name` и `buddy_email` недопустим в MVP
- owner не может назначить buddy самого себе

## Что будет следующим шагом
После утверждения этого draft:
- вынести его в настоящий OpenAPI YAML
- формализовать schemas
- закрепить status enums
- закрепить actor-specific permissions
