# Спек 4.2 — Борд вклада + Борд лучших пруфов

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 4 · Лидерборды Backend  
**Зависимости:** Существующие check_in_reviews, check_ins  
**Сложность:** S  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

**Борд вклада** — соревнование в помощи другим, а не в личной силе. Это здоровая конкуренция: кто больше поддержал, тот виден. Мотивирует культуру взаимопомощи.

**Борд лучших пруфов** — соревнуются артефакты, а не люди. Снижает токсичность: "мой конспект понравился" — не "я лучше тебя".

---

## API

### GET /v1/circles/:id/contribution-board

```
GET /v1/circles/:id/contribution-board?period=last_4_weeks&limit=5
Требует: участник круга
```

**Response 200:**
```json
{
  "data": {
    "board_type": "contribution",
    "period": "last_4_weeks",
    "entries": [
      {
        "rank": 1,
        "user_id": "uuid",
        "display_name": "Артём Козлов",
        "avatar_url": "...",
        "reviews_given": 14,
        "helpful_marks": 11,
        "newcomers_supported": 2,
        "contribution_score": 91,
        "is_current_user": false
      }
    ],
    "current_user_position": {
      "rank": 4,
      "reviews_given": 3,
      "contribution_score": 42,
      "total_participants": 7
    }
  }
}
```

**contribution_score формула:**
```
(reviews_given * 5 + helpful_marks * 3 + newcomers_supported * 10) / max_possible * 100
```

`helpful_marks` — кол-во раз когда получатель ревью нажал "👍 полезно". Если лайки не реализованы — = reviews_given × 0.8 (placeholder).

`newcomers_supported` — кол-во участников, для которых это был первый проверенный check-in.

---

### GET /v1/circles/:id/best-proofs-board

Лучшие артефакты круга за период. Соревнуются пруфы, не люди.

```
GET /v1/circles/:id/best-proofs-board?period=this_week&limit=5
Требует: участник круга
```

**Response 200:**
```json
{
  "data": {
    "board_type": "best_proofs",
    "period": "this_week",
    "entries": [
      {
        "rank": 1,
        "check_in_id": "uuid",
        "goal_title": "Kotlin Coroutines",
        "proof_preview": "Показал демо SupervisorJob — вот код и объяснение где применимо...",
        "proof_url": "https://...",
        "submitted_at": "2026-05-06T14:00:00Z",
        "author": {
          "user_id": "uuid",
          "display_name": "Мария Иванова",
          "avatar_url": "..."
        },
        "likes_count": 5,
        "buddy_approved": true,
        "proof_type": "artifact"
      }
    ]
  }
}
```

**Как ранжируются пруфы:**
1. `buddy_approved = true` — приоритет
2. `likes_count DESC`
3. `submitted_at DESC` (свежие выше при равных лайках)

---

### POST /v1/check-ins/:id/like

Лайк пруфу. Один пользователь — один лайк.

```
POST /v1/check-ins/:id/like
```

**Response 200:** `{ "data": { "check_in_id": "uuid", "likes_count": 6, "liked_by_me": true } }`

**Response 200 (повторный — toggle off):** `{ "data": { "check_in_id": "uuid", "likes_count": 5, "liked_by_me": false } }`

### SQL: таблица лайков

```sql
-- Добавить в миграцию 00024_proof_likes.sql
CREATE TABLE proof_likes (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  check_in_id  UUID NOT NULL REFERENCES check_ins(id) ON DELETE CASCADE,
  user_id      UUID NOT NULL REFERENCES users(id),
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(check_in_id, user_id)
);

CREATE INDEX proof_likes_checkin_idx ON proof_likes(check_in_id);
```

---

## Структура файлов

```
backend/internal/leaderboards/
  contribution.go   ← GetContributionBoard()
  best_proofs.go    ← GetBestProofsBoard()

backend/internal/checkins/
  http_handler.go   ← добавить POST /v1/check-ins/:id/like (toggle)

backend/migrations/
  00024_proof_likes.sql
```

---

## Acceptance Criteria

- [ ] contribution board: только участники данного круга
- [ ] contribution board: топ-N, аутсайдеры не показываются
- [ ] `current_user_position` заполнен всегда, даже если не в топе
- [ ] best-proofs board: только check-ins участников данного круга за период
- [ ] best-proofs board: `buddy_approved: true` приоритет над лайками
- [ ] POST /like: повторный вызов — toggle (снять лайк), не 409
- [ ] likes_count на check-in корректно обновляется при like/unlike
- [ ] 403 для не-участников круга

---

## Что нельзя делать

- Не показывать в contribution board участников с низким вкладом (только топ)
- Не публиковать лайки анонимно — `liked_by_me` всегда правдив
- Не давать лайкать собственные пруфы (проверка `user_id != check_in.user_id`)
