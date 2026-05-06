# Goal Helper Design

## Контекст

В `current-plan.md` для волны `1B доп.` зафиксирован Goal Helper «УТОЧНИ ЦЕЛЬ»: на `/goals/new` пользователь должен получать 3 SMART-варианта цели и примеры того, что считать пруфом. Полная продуктовая версия предполагает вызов реального LLM-провайдера, но на текущем этапе бюджет на провайдера отсутствует.

Нужно реализовать shipping-версию фичи так, чтобы:
- UX и API-контракт были уже production-shaped;
- runtime-cost оставался нулевым;
- подключение реального LLM позже не требовало переписывать frontend, endpoint, cache и rate-limit;
- решение не ломало уже начатый `circles-first vertical slice`.

## Цель

Добавить в продукт Goal Helper, который:
- принимает черновик цели;
- возвращает категорию и 3 SMART-варианта;
- сохраняет выбранные `proof_examples` и `category` в `goals`;
- работает сейчас через deterministic fake provider;
- в будущем переключается на real provider по конфигу без изменения публичного API.

## Архитектурное решение

### 1. Общий AI seam

Добавляется новый пакет `backend/internal/ai/llm_provider.go`, который вводит продуктовый интерфейс:

```go
type RefineProvider interface {
    RefineGoal(ctx context.Context, draftText string) (GoalRefineResult, error)
}
```

Это намеренно не OpenAI-specific abstraction. Пакет задаёт:
- `GoalRefineResult`
- `GoalRefineVariant`
- интерфейс `RefineProvider`
- fake-реализацию по умолчанию

### 2. Fake-by-default, real-by-flag

На этой итерации реальным источником ответов является `FakeGoalRefineProvider`.

Wiring:
- если `OPENAI_API_KEY` отсутствует, приложение использует `FakeGoalRefineProvider`;
- если позже будет добавлен `OpenAIRefineProvider` и ключ появится, wiring переключится на реальный provider;
- frontend и endpoint `POST /v1/goals/refine` при этом не меняются.

### 3. Граница ответственности

`goals` остаётся владельцем:
- endpoint `POST /v1/goals/refine`;
- cache и rate-limit orchestration;
- валидации входа и выхода;
- сохранения выбранных `proof_examples` и `category` в `goals`.

`ai` отвечает только за генерацию результата refinement по черновику.

## API-контракт

### Endpoint

`POST /v1/goals/refine`

Request:

```json
{
  "draft_text": "учить англ"
}
```

Response:

```json
{
  "category": "учёба",
  "variants": [
    {
      "title": "Подтянуть разговорный английский за 6 недель",
      "smart": "В течение 6 недель 5 раз в неделю проходить 20 минут разговорной практики и публиковать подтверждение каждого занятия.",
      "proof_examples": [
        "Скриншот пройденного урока",
        "Ссылка на запись разговорной практики",
        "Короткий текст с новыми фразами после занятия"
      ]
    }
  ]
}
```

Правила:
- `draft_text` обязателен;
- после `trim` минимум `3` символа;
- максимум `240` символов;
- в ответе всегда ровно `3` варианта;
- каждый вариант содержит `title`, `smart`, `proof_examples`;
- `proof_examples` всегда длины `3`;
- `category` ограничивается `32` символами.

Ошибки:
- `400 invalid_input`
- `429 refine_rate_limited`
- `500 internal_error`

## Backend design

### 1. Новые типы в goals

`goals/domain.go` расширяется:
- `CreateInput` получает:
  - `ProofExamples string`
  - `Category string`
- добавляются типы:
  - `RefineInput`
  - `RefineResponse`

Для первой версии `proof_examples` хранится как обычный `text`, а не JSON-массив. На клиенте три строки собираются в один нормализованный block text, например через `\n- ...`.

### 2. Миграция

Добавляется миграция, расширяющая goals:

```sql
ALTER TABLE goals
  ADD COLUMN proof_examples text,
  ADD COLUMN category text;

CREATE TABLE goal_refine_cache (
  hash text PRIMARY KEY,
  response jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE goal_refine_requests (
  id bigserial PRIMARY KEY,
  user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  draft_hash text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_goal_refine_requests_user_created
  ON goal_refine_requests(user_id, created_at DESC);
```

### 3. Cache

Cache-key:
- `sha256(normalized_draft_text)`
- берётся prefix `16` hex chars

TTL:
- `30` дней

Поведение:
- сначала проверяем cache;
- если cache-hit свежий, отдаём сохранённый JSON и не расходуем provider;
- если cache stale/missing, вызываем provider, валидируем output, сохраняем в cache и отдаём клиенту.

### 4. Rate limit

Лимит:
- `5` запросов на пользователя за последние `24` часа.

Поведение:
- rate-limit проверяется до provider call;
- cache-hit тоже считается запросом пользователя, потому что фича интерактивная и нам важен именно usage limit, а не стоимость provider;
- при превышении отдаётся `429 refine_rate_limited`.

### 5. Fake provider behavior

Fake provider детерминированный и без случайности.

Категории по ключевым словам:
- `англ`, `учёб`, `курс`, `экзам` -> `учёба`
- `трен`, `бег`, `зал`, `вес` -> `фитнес`
- `код`, `релиз`, `лендинг`, `продукт`, `проект` -> `работа`
- `рис`, `музык`, `пиш`, `твор` -> `творчество`
- default -> `развитие`

Три варианта строятся по шаблонам:
- outcome-driven
- cadence-driven
- evidence-driven

Цель fake provider:
- правдоподобный UX;
- стабильные snapshot-like тесты;
- тот же контракт, что будет у live provider.

## Frontend design

### 1. Goal setup screen

На `/goals/new` под полем названия цели появляется кнопка:

`[⚡] УТОЧНИТЬ ЦЕЛЬ`

Правила:
- disabled, если в title меньше `3` символов;
- при клике открывает drawer `GoalRefineSheet`.

### 2. Drawer

Новый компонент:
- `web/components/product/goal-refine-sheet.tsx`

Состояния:
- closed
- loading
- loaded
- error

Loaded-state показывает 3 карточки. Каждая карточка содержит:
- короткий `title`
- полный `smart`
- список `proof_examples`
- кнопки `ВЗЯТЬ` и `ИЗМЕНИТЬ`

### 3. Упрощение первой версии

Для первой shipping-версии:
- `ВЗЯТЬ` подставляет выбранный вариант в форму;
- `ИЗМЕНИТЬ` делает то же самое, после чего пользователь редактирует уже основную форму руками;
- отдельный inline editor внутри drawer не строится.

Это сознательное сокращение scope, чтобы не раздувать UI при сохранении главной ценности фичи.

### 4. Client state

`goal-setup-screen.tsx` хранит:
- `refineResult`
- `selectedVariant`
- `selectedProofExamples`
- `selectedCategory`

После выбора варианта:
- `title` и `description` заполняются;
- под формой появляется read-only блок:
  - «Что считается пруфом»
  - 3 bullet points из `proof_examples`

При submit `createGoal` получает:
- `proof_examples`
- `category`

## Границы первой реализации

### Входит

- fake provider;
- config-driven seam под future real provider;
- `POST /v1/goals/refine`;
- cache;
- rate limit;
- schema changes;
- drawer на `/goals/new`;
- сохранение `proof_examples` и `category` в `goals`.

### Не входит

- live OpenAI integration;
- telemetry/analytics;
- similarity reuse для inspiration;
- сложный inline editor в drawer;
- полное отображение `proof_examples` во всех review surfaces продукта.

## Тестирование

### Backend

- unit tests для fake provider;
- service tests:
  - invalid input
  - cache miss
  - cache hit
  - rate limit exceeded
  - validation of malformed provider result
- handler tests:
  - `400`
  - `429`
  - `200`

### Frontend

- кнопка refine disabled при коротком title;
- открытие drawer;
- успешный выбор варианта;
- submit формы с `proof_examples` и `category`.

### Verification

- `go test ./...`
- `npm test`
- `npm run lint`
- `npm run build`

## Риски и компромиссы

- Fake provider не даёт реальной LLM-ценности, но позволяет shipping UX и контракта без бюджета.
- Хранение `proof_examples` как `text` упрощает первую волну, но позже может потребовать миграции в `jsonb`, если examples станут редактируемыми по одному.
- Cache-hit считается в rate limit. Это консервативное поведение и может быть пересмотрено после появления real provider и анализа usage.

## Definition of Done

- Пользователь на `/goals/new` может уточнить цель через drawer;
- backend отдаёт 3 стабильных SMART-варианта через `POST /v1/goals/refine`;
- выбранный вариант сохраняет `proof_examples` и `category` в `goals`;
- фича работает без реального AI provider;
- backend seam позволяет позже включить live provider через конфиг, не ломая публичный контракт.
