# Голосовая система ProofForge

Этот документ переводит утверждённый voice package в рабочие правила для копирайта, интерфейсных сообщений и коротких продуктовых формулировок.

## Роль голоса
Голос ProofForge звучит как спокойная, серьёзная система ответственности с человеческим peer-слоем. Он не должен напоминать коучинг, терапию, мотивационный шум или публичную соцсеть.

## Базовые принципы
- Пиши ясно и коротко.
- Ставь статус раньше эмоции.
- Называй `proof`, `review`, `approval`, `progress` прямо, без эвфемизмов.
- Формулируй интерфейс как систему наблюдения и подтверждения, а не как персональный дневник.
- Звучание должно быть дисциплинированным, слегка сухим и уверенным.
- Допустим человеческий тон, но без мягкой обволакивающей интонации.

## Паттерны фраз
Используй следующие шаблоны как основу для UI copy, уведомлений и статусов.

| Паттерн | Когда использовать | Пример |
| --- | --- | --- |
| `Статус + объект` | Для компактных статусов | `Proof received.` / `Proof submitted.` |
| `Статус + ожидание` | Когда действие завершено, но нужен следующий шаг | `Waiting for review.` |
| `Объект + состояние` | Для карточек и заголовков секций | `Pact active.` |
| `Действие + дедлайн` | Для чек-инов и напоминаний | `Next check-in due Friday.` |
| `Статус + источник правды` | Для пояснения ролей | `Buddy approval required.` |
| `Короткое подтверждение` | Для success-состояний | `Checkpoint complete.` |

## Рекомендуемая структура сообщений
Для большинства поверхностей используй порядок:
1. Что произошло.
2. Что дальше.
3. Кто подтверждает или на что влияет статус.

Пример:
- `Proof uploaded. Waiting for buddy review.`
- `Checkpoint complete. Next check-in due Friday.`

## Анти-паттерны
Избегай фраз, которые:
- чрезмерно хвалят пользователя
- звучат как wellness-app
- превращают действие в эмоциональный праздник
- размывают статус
- говорят о прогрессе как о чувстве, а не как о наблюдаемом факте

Плохие направления:
- `Amazing job, keep going!`
- `You crushed it today!`
- `Build better habits with friends.`
- `Share your wins with the world.`
- `Every small step counts on your journey.`

## UI copy examples
Ниже приведены короткие примеры, которыми можно пользоваться как каноном для интерфейса.

### Статусы
- `Proof received.`
- `Waiting for review.`
- `Approved by buddy.`
- `Needs another look.`
- `Checkpoint complete.`

### Карточки и секции
- `Pact active.`
- `Buddy status: ready.`
- `Progress pending approval.`
- `Weekly recap ready.`
- `Next check-in due Friday.`

### Кнопки и действия
- `Upload proof`
- `Send for review`
- `Mark complete`
- `Request approval`
- `View recap`

### Пустые состояния
- `No proof yet.`
- `Your next checkpoint will appear here.`
- `Invite a buddy to start the review loop.`

### Ошибки
- `Upload failed. Try again.`
- `Could not load review state.`
- `Something broke in the proof pipeline. Refresh to retry.`

### Уведомления
- `Your buddy approved the latest proof.`
- `A new check-in is due Friday.`
- `Weekly recap is ready for review.`

## Контроль качества
Перед публикацией любого текста проверь:
- понятен ли следующий шаг без догадок
- есть ли явный статус
- нет ли мотивационного шума
- не звучит ли текст как соцсеть или habit tracker
- не переусложнена ли фраза
