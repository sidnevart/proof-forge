# Решение: первый вертикальный срез кругов строится поверх текущего proof-engine

## Статус
Принято, 2026-05-02.

## Контекст
Нужно было внедрить social-layer с кругами, weekly pressure и соревновательной видимостью, не ломая текущий MVP-контур `goal -> invite -> check-in -> review`.

Полная целевая модель предполагает отдельные сущности для:
- кругов;
- участников;
- peer assignments;
- сезонов;
- standings;
- weekly assembly;
- social events.

Если пытаться внедрить всё сразу как полноценный новый домен, это резко увеличивает объём миграций, API-контрактов и риск регрессий в уже работающем proof-loop.

## Решение
Для первого среза круги реализуются как отдельный социальный слой поверх существующих `goals`, `check_ins` и `check_in_reviews`.

Принятые ограничения:
- добавлены таблицы `circles`, `circle_memberships`, `circle_seasons`;
- в `goals` добавлен nullable `circle_id`;
- источник правды по шагу по-прежнему задаётся текущей goal-моделью через `buddy`;
- `standings` и `weekly assembly` считаются на чтении из существующих goal/check-in/review данных;
- materialized standings table и отдельный event log пока не вводятся;
- отдельная таблица `peer_assignments` пока не добавляется.

## Последствия
Плюсы:
- появился рабочий end-to-end сценарий `circle -> join -> goal in circle -> approved step -> standings -> weekly assembly`;
- не сломан текущий invite/check-in/review flow;
- можно быстро проверять social mechanics в продукте.

Минусы:
- peer assignment пока привязан к `buddy` на уровне конкретной цели, а не сезона круга;
- standings и weekly assembly пока derived, а не предрасчитанные;
- season reset, richer pressure rules и event history пока ограничены первым приближением.

## Следующие шаги
- вынести назначение peer на уровень круга/сезона;
- добавить сезонный reset и архив standings;
- ввести отдельный social event log;
- расширить UI круга до полноценного weekly assembly surface и seasonal rivalry loop.
