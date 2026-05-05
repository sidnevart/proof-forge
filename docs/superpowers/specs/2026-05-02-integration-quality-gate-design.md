# Дизайн skill `integration-quality-gate`

## Цель
Создать project skill, который заставляет агента серьёзно проверять vertical slice ProofForge перед merge или release, а не ограничиваться happy-path тестом и общими словами про "всё прошло".

Skill должен покрывать полный quality gate:
- backend integration tests;
- frontend end-to-end coverage;
- post-deploy smoke checks;
- финальный release verdict с явными блокерами и пробелами покрытия.

## Когда skill должен срабатывать
Skill должен срабатывать на запросы вроде:
- "всё проверь";
- "серьёзно подойдём к проверке";
- "добавь интеграционные тесты";
- "закрой e2e";
- "проверь сценарии";
- "готово ли это к пушу";
- "можно ли катить на прод";
- "сделай quality gate для vertical slice".

Skill обязателен, если запрос затрагивает хотя бы одно из:
- backend integration coverage;
- frontend e2e coverage;
- release validation;
- smoke checklist;
- final go/no-go verdict перед push, merge или deploy.

## Что skill делает
Skill не просто предлагает "добавить тесты", а ведёт агента через обязательный порядок:

1. собрать scenario matrix;
2. найти текущее покрытие и пробелы;
3. закрыть backend integration coverage;
4. закрыть frontend e2e coverage;
5. сверить и обновить deploy smoke checklist;
6. прогнать верификационные команды;
7. выдать финальный release verdict.

## Границы skill
Skill покрывает:
- Go integration tests на уровне HTTP routes + test DB;
- frontend e2e на уровне пользовательских маршрутов и многошаговых flows;
- операционную smoke-проверку после деплоя;
- оценку release readiness.

Skill не покрывает:
- unit-test стратегию как основную тему;
- визуальный UI design;
- продуктовый roadmap;
- полную инфраструктурную автоматизацию CI/CD вне контекста quality gate.

## Основной workflow

### 1. Scenario matrix
До написания новых тестов агент обязан разложить slice на сценарии и отметить для каждого:
- слой проверки;
- текущее покрытие;
- требуемый тест или smoke check;
- blocker status.

Минимальные категории сценариев:
- happy path;
- permissions / authorization;
- invalid state transitions;
- regression на существующий proof loop;
- deploy smoke.

### 2. Аудит текущего покрытия
Агент обязан явно проверить:
- `backend/internal/platform/app/*_integration_test.go`;
- связанные `backend/internal/*/service_test.go`, если они влияют на картину coverage;
- frontend component tests и e2e tests;
- `docs/ops/smoke-test.md`.

После аудита каждый сценарий должен попасть в одно из состояний:
- `covered`;
- `partially covered`;
- `missing`;
- `stale`.

### 3. Backend integration first
Для ProofForge backend integration является основным слоем доверия по доменным переходам.

Skill должен требовать:
- тестировать реальные HTTP routes;
- поднимать реальный test DB через существующий harness;
- проверять status transitions и authorization;
- проверять derived read models, если UI и social loop опираются на вычисляемое состояние.

Skill должен запрещать подмену integration coverage unit-тестами.

### 4. Frontend e2e second
Frontend e2e должны покрывать только критические пользовательские маршруты.

Skill должен требовать:
- не подменять redirect/session/multi-step flow component-тестами;
- проверять настоящие пользовательские переходы для основных сценариев;
- явно называть gap, если e2e инфраструктура ещё не настроена;
- предлагать минимальный bootstrap, если без него coverage невозможно закрыть.

### 5. Deploy smoke third
Skill обязан сверять новый slice с `docs/ops/smoke-test.md`.

Если vertical slice меняет пользовательский или операторский flow, но smoke checklist не обновлён, проверка считается незавершённой.

### 6. Финальный verdict
В конце agent обязан выдать verdict в фиксированном формате:
- что покрыто;
- что не покрыто;
- какие есть blockers;
- какие команды были прогнаны;
- можно ли считать slice `ready`, `ready with known gaps` или `not ready`.

Skill должен запрещать формулировки вроде:
- "вроде всё ок";
- "должно работать";
- "основное проверили";
- "можно пушить", если нет свежих команд и явного списка пробелов.

## Минимальная матрица сценариев для текущего circles slice
Для текущего вертикального среза `circles` skill должен требовать как минимум:

### Backend integration
- `register owner -> create circle -> peer join -> goal in circle -> invite accept -> check-in -> evidence -> submit -> approve -> standings -> weekly assembly`;
- owner не может создать goal в circle без membership;
- buddy вне круга не может быть назначен в circle goal;
- не-участник не читает `GET /circles/{id}`, `standings`, `weekly-assembly`;
- invalid invite code, duplicate join, full circle;
- owner не может self-approve;
- левый пользователь не может approve/reject/request-changes;
- invalid transitions по check-in status;
- approved влияет на standings и weekly assembly;
- derived social statuses `approved`, `at_risk`, `dropped`, `comeback` проверяются явно.

### Frontend e2e
- новый пользователь проходит `dashboard -> create/join circle -> create goal in circle`;
- выбор buddy ограничен участниками круга;
- invite acceptance flow;
- owner check-in flow;
- buddy approval flow;
- dashboard показывает circle context и social surfaces;
- пустые и forbidden states не ломают маршрут.

### Deploy smoke
- readiness;
- container status;
- регистрация и session cookie;
- создание круга;
- join второго участника;
- создание goal внутри круга;
- draft check-in, evidence, submit;
- approval вторым участником;
- проверка `standings` и `weekly assembly`;
- unauthenticated rejection;
- security headers;
- rate limiting;
- webhook checks, если включены.

## Структура skill
Skill должен быть создан в:

```text
.codex/skills/integration-quality-gate/
  SKILL.md
  references/
    scenario-matrix.md
    backend-integration-rules.md
    frontend-e2e-rules.md
    deploy-smoke-rules.md
    release-verdict-template.md
```

## Назначение файлов

### `SKILL.md`
Содержит:
- trigger description;
- основной workflow;
- обязательные проверки;
- запреты на ложное завершение;
- требования к финальному verdict.

### `references/scenario-matrix.md`
Содержит:
- канонический шаблон scenario matrix;
- пример заполнения для `circles` vertical slice;
- правила классификации `covered / partially covered / missing / stale`.

### `references/backend-integration-rules.md`
Содержит:
- где размещать integration tests;
- как использовать test DB harness;
- какие переходы и права доступа обязательны;
- что нельзя считать integration test.

### `references/frontend-e2e-rules.md`
Содержит:
- какие journeys считать критическими;
- как отличать e2e от component coverage;
- как оформлять gap, если e2e bootstrap отсутствует;
- как выбирать минимальный набор сценариев для vertical slice.

### `references/deploy-smoke-rules.md`
Содержит:
- как расширять `docs/ops/smoke-test.md`;
- какие проверки обязательны для user-facing и operator-facing изменений;
- как формулировать blocker expectations.

### `references/release-verdict-template.md`
Содержит:
- обязательный шаблон итогового отчёта;
- статусы `ready`, `ready with known gaps`, `not ready`;
- правила перечисления blockers и uncovered scenarios.

## Требования к содержимому `SKILL.md`
В `SKILL.md` должны быть явно зафиксированы правила:
- не начинать с написания тестов без scenario matrix;
- backend integration идёт раньше frontend e2e;
- frontend component tests не заменяют e2e;
- локальный `go test` не равен release readiness;
- отсутствие обновления smoke checklist считается gap;
- нельзя заканчивать задачу без свежих verification commands;
- если coverage неполный, нужно назвать конкретные отсутствующие сценарии.

## Шаблон итогового verdict
Skill должен требовать такой формат закрытия:

```md
## Quality Gate Verdict

### Covered
- `<перечень покрытых сценариев>`

### Gaps
- `<перечень непокрытых или частично покрытых сценариев>`

### Blockers
- `<перечень блокеров, если они есть>`

### Commands
- `go test ./...`
- `npm run test:e2e`
- `make verify-web`
- `make verify-deploy`

### Release Verdict
- `ready` | `ready with known gaps` | `not ready`

### Why
- `<краткое объяснение, почему выбран именно этот verdict>`
```

## Initial eval prompts для проверки skill
После создания skill его нужно прогнать минимум на таких промптах:

1. "Добавь серьёзное интеграционное покрытие для нового circle flow и скажи, можно ли это катить на прод."
2. "Проверь vertical slice с invite, check-in и buddy approval. Нужен полный quality gate, не только happy-path."
3. "Перед push разложи сценарии, добей пробелы по backend и e2e, и обнови smoke checklist."

## Критерии готовности skill
Skill считается готовым, если:
- он стабильно заставляет агента строить scenario matrix до реализации;
- он не даёт завершить задачу без backend integration, frontend e2e и smoke perspective;
- он приводит к явному release verdict вместо размытого summary;
- он остаётся применимым не только к `circles`, но и к другим ProofForge vertical slices.

## Основные риски
- skill может стать слишком общим и начать звучать как generic QA checklist;
- skill может перегрузить малые задачи, если не объяснить критерии критичности slice;
- skill может дать ложное чувство полноты, если не требовать перечисления непокрытых сценариев.

## Решения по рискам
- привязать skill к ProofForge vertical slice логике, а не к абстрактному тестированию;
- явно отделить mandatory critical journeys от optional extended coverage;
- сделать список uncovered scenarios обязательной частью финального verdict.
