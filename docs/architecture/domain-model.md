# Доменная модель MVP

## Основные сущности

### User
Пользователь системы.

Роль в MVP:
- владелец goal
- buddy для чужого goal

Один и тот же user может быть owner в одном goal и buddy в другом.

### Goal
Серьёзное обязательство, которое пользователь хочет довести до результата.

Goal:
- всегда принадлежит owner
- всегда связан с обязательным buddy
- имеет собственный progress state
- живёт внутри accountability loop, а не как одиночный self-tracker

### Pact
Явно зафиксированная accountability-связь между owner и buddy по конкретному goal.

Pact определяет:
- кто owner
- кто buddy
- что именно подтверждается
- какой review loop действует
- активна ли связка

### Invite
Механизм присоединения buddy к pact.

Invite нужен, чтобы:
- owner не мог “назначить” buddy в одностороннем порядке без принятия
- был явный момент входа второго человека в контур

### CheckIn
Одна попытка зафиксировать прогресс по одному goal.

CheckIn:
- относится ровно к одному goal
- создаётся owner
- отправляется на review buddy
- может переживать цикл request changes

### Evidence
Набор proof-артефактов внутри одного check-in.

Evidence может включать:
- text note
- external link
- file/image artifact

Evidence не существует сама по себе вне check-in.

### ReviewDecision
Решение buddy по check-in.

Поддерживаемые решения:
- `approved`
- `changes_requested`
- `rejected`

### WeeklyRecap
Сводка подтверждённого движения за период.

AI помогает её собрать, но recap не меняет источник правды и не заменяет review decisions.

## Glossary
- `owner` — тот, чья цель и чей progress оценивается
- `buddy` — обязательный внешний reviewer для goal
- `check-in` — попытка зафиксировать движение по goal
- `evidence` — доказательства внутри check-in
- `review` — ответ buddy на конкретный check-in
- `progress` — подтверждённое движение по goal

## Инварианты
- goal не существует как полноценный accountability object без buddy
- один check-in относится только к одному goal
- progress подтверждается только через buddy approval
- AI не меняет доменные статусы
- `changes_requested` не равно `rejected`
- rejected check-in не обновляет progress
- approved check-in обновляет progress

## Domain boundaries
В доменную модель входят:
- правила статусов
- инварианты approval loop
- связи owner / buddy / goal / check-in
- логика того, что считается подтверждённым движением

В доменную модель не входят:
- transport DTO
- OpenAPI schema details
- SQL table layout
- object storage provider specifics

## Статусы верхнего уровня

### Goal
- `pending_buddy_acceptance`
- `active`
- `paused`
- `completed`
- `archived`

### Pact
- `invited`
- `active`
- `ended`

### CheckIn
- `draft`
- `submitted`
- `changes_requested`
- `approved`
- `rejected`

### WeeklyRecap
- `queued`
- `generated`
- `failed`

## Семантика review loop

### approved
- check-in принят
- progress обновляется
- goal health/streak-like metrics могут обновляться

### changes_requested
- check-in не принят, но остаётся живым
- owner может дополнить evidence и отправить повторно
- progress пока не обновляется

### rejected
- check-in закрывается как неуспешный
- progress не обновляется
- для нового шанса нужен новый check-in, а не доработка старого
