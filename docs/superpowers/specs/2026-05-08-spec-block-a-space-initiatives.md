# Spec: Space Initiatives — Block A

**Date:** 2026-05-08  
**Status:** Approved  
**Scope:** Teamspaces and Community Spaces

---

## Overview

Space Initiatives — совместные задачи с единым критерием, которые могут создавать как лидеры так и обычные участники пространства. Любой участник пространства может свободно вступить. Пруфы аппрувят сами участники по peer review модели.

---

## Ключевые решения

| Вопрос | Решение |
|--------|---------|
| Что такое инициатива | Совместная цель: одна инициатива, несколько участников, пруфы в общий котёл |
| Кто аппрувает | Любой другой участник инициативы (peer review) — не автор пруфа |
| Как вступить | Самостоятельно, без одобрения создателя |
| Критерий пруфа | Единый для всех, задаётся при создании инициативы |
| Архитектурный подход | Initiative как обёртка над личными goals — максимум переиспользования |

---

## Модель данных

### Новая таблица: `initiatives`

```sql
CREATE TABLE initiatives (
    id                  BIGSERIAL PRIMARY KEY,
    space_type          TEXT NOT NULL CHECK (space_type IN ('teamspace', 'community')),
    teamspace_id        BIGINT REFERENCES teams(id) ON DELETE CASCADE,
    community_space_id  BIGINT REFERENCES community_spaces(id) ON DELETE CASCADE,
    creator_id          BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title               TEXT NOT NULL CHECK (length(trim(title)) BETWEEN 3 AND 120),
    description         TEXT NOT NULL DEFAULT '',
    proof_criteria      TEXT NOT NULL CHECK (length(trim(proof_criteria)) >= 10),
    status              TEXT NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active', 'archived')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- XOR: принадлежит ровно одному пространству
    CONSTRAINT initiatives_space_xor CHECK (
        (teamspace_id IS NOT NULL AND community_space_id IS NULL) OR
        (teamspace_id IS NULL AND community_space_id IS NOT NULL)
    )
);

CREATE INDEX idx_initiatives_teamspace ON initiatives(teamspace_id) WHERE teamspace_id IS NOT NULL;
CREATE INDEX idx_initiatives_community ON initiatives(community_space_id) WHERE community_space_id IS NOT NULL;
```

### Изменение: `goals` (+1 поле)

```sql
ALTER TABLE goals ADD COLUMN initiative_id BIGINT REFERENCES initiatives(id) ON DELETE SET NULL;
CREATE INDEX idx_goals_initiative ON goals(initiative_id) WHERE initiative_id IS NOT NULL;
```

Инвариант: если `initiative_id IS NOT NULL` — это инициативная цель. Buddy не назначается, buddy-guard в checkins обходится.

### Новая таблица: `initiative_approvals`

```sql
CREATE TABLE initiative_approvals (
    id           BIGSERIAL PRIMARY KEY,
    checkin_id   BIGINT NOT NULL REFERENCES check_ins(id) ON DELETE CASCADE,
    approver_id  BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    comment      TEXT NOT NULL DEFAULT '',
    approved_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT initiative_approvals_one_per_checkin UNIQUE (checkin_id)
);
```

Инвариант, проверяемый в сервисе: `approver_id ≠ check_in.owner_user_id`.

---

## Доменная логика

### Создание инициативы

- Создаёт любой авторизованный участник пространства (member или leader).
- Валидация: `title` 3–120 символов, `proof_criteria` ≥ 10 символов.
- Статус сразу `active`.

### Join (вступление)

`POST /v1/initiatives/{id}/join`:

1. Проверить что пользователь — участник родительского пространства.
2. Проверить что у пользователя нет активного goal с этим `initiative_id` (идемпотентность).
3. Создать Goal:
   - `owner_user_id` = actor
   - `title` = initiative.title
   - `description` = initiative.proof_criteria
   - `initiative_id` = initiative.id
   - `status` = `active` (минуя buddy acceptance flow)
   - `buddy_email` = `""` (не требуется)

### Сдача пруфа

Через существующий `POST /v1/check_ins` против initiative-goal. Сервис checkins добавляет ветку:

```go
if goal.InitiativeID != nil {
    // пропустить проверку buddy_required
    // статус сразу 'submitted', не 'pending_buddy'
}
```

### Peer-аппрув

`POST /v1/initiatives/{id}/proofs/{checkinId}/approve`:

1. Проверить что approver — участник инициативы (есть goal с этим initiative_id).
2. Проверить что approver ≠ автор check_in.
3. Вставить запись в `initiative_approvals`.
4. Обновить `check_ins.status = 'approved'`.

### Просмотр pending-пруфов

`GET /v1/initiatives/{id}/pending-proofs`:

- Возвращает check_ins со статусом `submitted`, linked к goals с `initiative_id = id`, исключая собственные пруфы текущего пользователя.

---

## API Endpoints

```
POST /v1/spaces/{type}/{spaceId}/initiatives         — создать инициативу
GET  /v1/spaces/{type}/{spaceId}/initiatives         — список инициатив пространства
GET  /v1/initiatives/{id}                            — детали + прогресс участников
POST /v1/initiatives/{id}/join                       — вступить
GET  /v1/initiatives/{id}/pending-proofs             — пруфы ждущие аппрува
POST /v1/initiatives/{id}/proofs/{checkinId}/approve — peer-аппрув (+ comment)
```

`{type}` = `teamspace` | `community`.

---

## UX-поток

### Вкладка «Инициативы» в пространстве

- Список карточек инициатив: заголовок, proof_criteria, кол-во участников, активность сегодня, кол-во pending-пруфов.
- Состояние карточки зависит от членства:
  - Не вступил → кнопка «ПРИСОЕДИНИТЬСЯ»
  - Уже участвую → badge «✓ ВЫ УЧАСТВУЕТЕ» + своя серия + кол-во пруфов ждущих аппрува
- Любой участник пространства может создать инициативу (кнопка «+ СОЗДАТЬ ИНИЦИАТИВУ» внизу списка).

### Страница инициативы

Два раздела:

**Peer review очередь** (жёлтый header «⚡ N ПРУФОВ ЖДУТ ВАШЕГО АППРУВА»):
- Карточки пруфов от других участников: имя, время, текст, вложение.
- Кнопки «✓ ОДОБРИТЬ» и «✏ УТОЧНИТЬ» (УТОЧНИТЬ требует комментарий, как у buddy).
- Собственные пруфы не показываются в этом разделе.

**Прогресс участников**:
- Список с прогресс-барами. Метрика — кол-во одобренных пруфов.
- Нет публичного ранжирования аутсайдеров — бары показывают абсолютный прогресс, не место в рейтинге.

---

## Инварианты и граничные случаи

| Случай | Поведение |
|--------|-----------|
| Пользователь вступает повторно | 200 OK, возвращает существующий goal — join идемпотентен |
| Автор пытается аппрувить свой пруф | 403 Forbidden |
| Не-участник пытается аппрувить | 403 Forbidden |
| Инициатива архивируется | Новые join и сдача пруфов блокируются, старые данные сохраняются |
| Пользователь покидает пространство | Его goal остаётся, но он больше не видит инициативы |

---

## Что не меняется

- Таблицы `check_ins`, `goals`, `pacts`, `invites` — без изменений схемы (кроме +1 поля в goals).
- Дашборд участника — initiative-goals отображаются как обычные goals.
- Аналитика, стрики, досье — работают автоматически через существующие goals/checkins.
- Buddy flow для обычных (не инициативных) goals — без изменений.

---

## Файлы реализации

```
backend/migrations/00026_space_initiatives.sql
backend/internal/initiatives/
  domain.go
  ports.go
  service.go
  service_test.go
  http_handler.go
  postgres_repository.go
backend/internal/checkins/service.go        — ветка initiative bypass
backend/internal/platform/app/api.go        — регистрация роутов
web/app/(product)/spaces/[type]/[id]/       — страница пространства с вкладкой Инициативы
web/components/product/initiative-list.tsx
web/components/product/initiative-detail.tsx
web/components/product/initiative-peer-review.tsx
web/lib/types.ts                            — типы Initiative, InitiativeProof
web/lib/api.ts                              — новые fetch-функции
```
