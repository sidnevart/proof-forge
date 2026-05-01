# Системная архитектура MVP

## Кратко
Для первой версии ProofForge выбирается архитектура `web app + Go API + Go worker + PostgreSQL + object storage`, без Telegram в core-MVP.

Это не микросервисная система. Это модульный монолит с отдельным worker-процессом для фоновых задач. Такой подход даёт простой старт, но не смешивает synchronous user flows и asynchronous processing в одну бесформенную кучу.

## Границы системы
В MVP входят:
- web frontend
- Go API
- Go worker
- PostgreSQL
- object storage для proof-файлов и изображений
- AI weekly recap как фоновая задача

В MVP пока не входят:
- Telegram flows
- мобильные клиенты
- сложная multi-tenant B2B логика
- публичная соцсеть достижений

## Компоненты

### 1. Web app
Отвечает за:
- создание goal
- приглашение buddy
- отправку check-in
- загрузку proof
- просмотр review state
- просмотр dashboard и recap

Web app не содержит бизнес-правил домена. Он использует API как источник истины.

### 2. Go API
Отвечает за:
- аутентификацию и user session boundary
- CRUD и domain workflows
- review flow
- выдачу upload intents / upload coordination
- чтение dashboard state
- orchestration domain use cases

Go API — это единственная онлайн-точка принятия бизнес-решений.

### 3. Go worker
Отвечает за:
- генерацию weekly recaps
- post-processing proof artifacts при необходимости
- scheduled reminders и future async jobs
- повторные задачи, которые не должны тормозить API request path

Worker не должен дублировать доменные решения API. Он исполняет фоновые use cases через те же доменные сервисы и репозиторные контракты.

### 4. PostgreSQL
Отвечает за:
- users
- goals
- buddy pacts
- invites
- check-ins
- approvals
- recap metadata
- audit-like domain events, если они нужны для наблюдаемости и истории

PostgreSQL хранит основной источник правды по доменному состоянию.

### 5. Object storage
Отвечает за:
- file/image proof artifacts
- metadata-bound storage paths
- безопасное хранение и выдачу ссылок

Сами бинарные файлы не должны храниться в PostgreSQL.

## Основной поток данных
1. Пользователь создаёт goal.
2. Goal сразу предполагает обязательный buddy relationship.
3. Buddy принимает invite.
4. Пользователь отправляет check-in по одному конкретному goal.
5. Check-in содержит proof: text, link, file/image upload.
6. API фиксирует check-in в статусе ожидания review.
7. Buddy делает одно из действий: approve / request changes / reject.
8. Если approve, обновляется progress state goal.
9. Если request changes, check-in остаётся живым и ждёт дополнения proof.
10. Если reject, check-in закрывается как неуспешная попытка.
11. Worker позже собирает weekly recap на основе подтверждённого движения и review history.

## Почему не микросервисы
Для MVP микросервисы только поднимут стоимость:
- больше operational surface
- больше integration points
- больше failure modes
- хуже удерживается доменная модель

Модульный монолит с worker-процессом даёт достаточную структурность без раннего раздробления.

## Основные архитектурные принципы
- buddy approval — источник правды о подтверждённом прогрессе
- AI не approve proof и не меняет доменный статус check-in
- один check-in относится только к одному goal
- proof upload — first-class citizen MVP, а не “потом добавим”
- async work вынесен из request path
- backend должен оставаться feature-first, а не utils-first

## Failure modes, которые надо учитывать
- upload file завершился, а check-in не был зафиксирован
- check-in создан, но buddy review долго не происходит
- buddy запрашивает changes несколько раз подряд
- worker недоступен, и weekly recap не собрался вовремя
- object storage доступно частично или временно недоступно
- повторная отправка proof пользователем создаёт race по review state

Эти сценарии должны быть явно учтены на уровне доменной модели и API-контрактов.
