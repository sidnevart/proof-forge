# Схема базы данных MVP

## Основной принцип
PostgreSQL хранит доменный источник правды, а object storage хранит бинарные proof artifacts.

## Таблицы

### users
Поля:
- `id`
- `email`
- `display_name`
- `avatar_url`
- `created_at`
- `updated_at`

### user_sessions
Поля:
- `id`
- `user_id`
- `token_hash`
- `expires_at`
- `last_seen_at`
- `created_at`

Назначение:
- минимальный session-layer для web MVP
- регистрация создаёт session сразу после создания user
- frontend использует cookie-based auth, а не временный `X-User-ID` обход

### goals
Поля:
- `id`
- `owner_user_id`
- `buddy_user_id`
- `title`
- `description`
- `status`
- `current_progress_health`
- `current_streak_count`
- `created_at`
- `updated_at`
- `archived_at`

Замечание:
Хотя buddy relationship логически оформляется через pact/invite, для быстрых reads MVP допустимо держать `buddy_user_id` и в `goals`, если это упрощает dashboard queries.

### pacts
Поля:
- `id`
- `goal_id`
- `owner_user_id`
- `buddy_user_id`
- `status`
- `accepted_at`
- `ended_at`
- `created_at`
- `updated_at`

### invites
Поля:
- `id`
- `goal_id`
- `pact_id`
- `inviter_user_id`
- `invitee_user_id`
- `token_hash`
- `status`
- `expires_at`
- `accepted_at`
- `created_at`

### check_ins
Поля:
- `id`
- `goal_id`
- `owner_user_id`
- `status`
- `submitted_at`
- `approved_at`
- `rejected_at`
- `changes_requested_at`
- `created_at`
- `updated_at`

### evidence_items
Поля:
- `id`
- `check_in_id`
- `kind`
- `text_content`
- `external_url`
- `storage_key`
- `mime_type`
- `file_size_bytes`
- `created_at`

`kind`:
- `text`
- `link`
- `file`
- `image`

Одна строка — один evidence artifact.

### check_in_reviews
Поля:
- `id`
- `check_in_id`
- `reviewer_user_id`
- `decision`
- `comment`
- `created_at`

Этот слой хранит review history. Финальный статус check-in живёт в `check_ins`, а история решений — здесь.

### weekly_recaps
Поля:
- `id`
- `goal_id`
- `owner_user_id`
- `period_start`
- `period_end`
- `status`
- `summary_text`
- `model_name`
- `generated_at`
- `created_at`

## Индексы

### goals
- `(owner_user_id, status)`
- `(buddy_user_id, status)`

### user_sessions
- `(user_id, expires_at desc)`
- `(token_hash)`

### pacts
- `(goal_id)`
- `(owner_user_id, buddy_user_id, status)`

### invites
- `(invitee_user_id, status)`
- `(token_hash)`
- `(goal_id, status)`

### check_ins
- `(goal_id, created_at desc)`
- `(owner_user_id, status)`
- `(goal_id, status)`

### evidence_items
- `(check_in_id)`

### check_in_reviews
- `(check_in_id, created_at desc)`

### weekly_recaps
- `(goal_id, period_start, period_end)`
- `(owner_user_id, period_start desc)`

## Ограничения
- foreign keys обязательны между основными доменными таблицами
- `check_ins.goal_id` обязателен
- `evidence_items.check_in_id` обязателен
- `check_in_reviews.reviewer_user_id` должен совпадать с активным buddy для goal в момент review
- nullable timestamps должны отражать только реально наступившие transitions

## Почему отдельная review history table
Потому что `changes_requested` может случаться несколько раз. Нам нужна история review loop, а не только финальный snapshot.

## Что не хранить в БД
- сами бинарные файлы
- секреты object storage
- derived presentation-only fields, которые можно безопасно вычислить на read path
