# Переходы статусов MVP

## Goal transitions

### `pending_buddy_acceptance -> active`
Происходит, когда buddy принимает invite.

### `active -> paused`
Происходит, когда owner или системное правило временно останавливает goal.

### `paused -> active`
Происходит при возобновлении работы.

### `active -> completed`
Происходит, когда цель достигнута и закрыта как завершённая.

### `active|paused|completed -> archived`
Происходит, когда goal убирается из активного operational слоя.

## Pact transitions

### `invited -> active`
Происходит после принятия invite buddy.

### `active -> ended`
Происходит, когда relationship по goal завершён или разорван.

## CheckIn transitions

### `draft -> submitted`
Только owner может отправить check-in на review.
На момент перехода должны существовать:
- связанный goal
- активный pact
- обязательный buddy
- хотя бы один meaningful evidence artifact

### `submitted -> changes_requested`
Только buddy может запросить доработку.
Progress не меняется.
Check-in остаётся редактируемым в пределах allowed resubmission policy.

### `changes_requested -> submitted`
Owner дополняет proof и отправляет заново.

### `submitted -> approved`
Только buddy может approve.
Этот переход обновляет progress goal.

### `submitted -> rejected`
Только buddy может reject.
Check-in закрывается без обновления progress.

### `changes_requested -> rejected`
Допустимо, если после запроса доработок owner не предоставил достаточный proof, и buddy принимает финальное отрицательное решение.

### Запрещённые переходы
- `approved -> submitted`
- `approved -> changes_requested`
- `rejected -> submitted`
- `rejected -> approved`

Для новой попытки после reject нужен новый check-in.

## WeeklyRecap transitions

### `queued -> generated`
Worker успешно собрал recap.

### `queued -> failed`
Worker не смог собрать recap.

### `failed -> queued`
Допустим retry.

## Инварианты переходов
- approve/reject/request changes доступны только buddy
- owner не может self-approve progress
- AI не участвует в статусных переходах check-in
- approved — единственный transition, который обновляет подтверждённый progress
- changes_requested — не провал, а открытый review loop
- rejected — финальное решение по конкретному check-in
