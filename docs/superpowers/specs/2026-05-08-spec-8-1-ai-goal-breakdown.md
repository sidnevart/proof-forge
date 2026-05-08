# Спек 8.1 — AI: Goal-to-Proof Breakdown

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 8 · AI-персонализация  
**Зависимости:** Спек 1.3 (Proof Contract), 5.3 (Proof Contract UI)  
**Сложность:** M  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Пользователь создал цель — но что конкретно доказывать? AI помогает разбить размытую цель на 3–5 конкретных proof contract вариантов. Это снижает барьер первого шага и учит формату качественного пруфа.

**DNA П9:** AI объясняет, не управляет. Варианты — предложения, пользователь выбирает.

---

## Маршрут

```
POST /v1/ai/goal-to-proofs
```

Вызывается из UI при создании цели или просмотре страницы цели. Кнопка "Предложить 3 пруфа" — не обязательный шаг, инициируется пользователем.

---

## Backend

### Структура

```
backend/internal/ai/
  goal_breakdown.go
  goal_breakdown_prompt.go
  client.go          ← обёртка над Anthropic SDK (уже существует или создать)
```

### Handler

```go
// POST /v1/ai/goal-to-proofs
type GoalToProofsRequest struct {
    GoalID    string `json:"goal_id"`
    GoalTitle string `json:"goal_title"`
    GoalDesc  string `json:"goal_desc,omitempty"`
    Context   string `json:"context,omitempty"` // teamspace name, circle name
}

type ProofSuggestion struct {
    WhatToProve string `json:"what_to_prove"` // формулировка что именно доказывать
    HowToProve  string `json:"how_to_prove"`  // формат артефакта: код, демо, статья
    DueInDays   int    `json:"due_in_days"`   // рекомендуемый срок
    Mode        string `json:"mode"`          // single_proof / regular_rhythm
    Rationale   string `json:"rationale"`     // почему этот пруф полезен
}

type GoalToProofsResponse struct {
    Suggestions []ProofSuggestion `json:"suggestions"`
}
```

### Prompt

```go
const goalBreakdownSystemPrompt = `Ты помощник платформы ProofForge для доказательства прогресса в обучении.
Твоя задача — разбить цель пользователя на 3-5 конкретных доказуемых шагов.

Правила:
- Каждый пруф должен быть доказуемым: код, демо, статья, презентация — не "понял", а "показал"
- Срок: реалистичный (1-14 дней), не утопичный
- Формат: что конкретно создать/показать, а не процесс
- Тон: дружелюбный, без давления
- Язык: русский

Ответ в JSON:
{
  "suggestions": [
    {
      "what_to_prove": "...",
      "how_to_prove": "...",
      "due_in_days": N,
      "mode": "single_proof" | "regular_rhythm",
      "rationale": "..."
    }
  ]
}`

func buildGoalBreakdownPrompt(goal GoalToProofsRequest) string {
    return fmt.Sprintf(`Цель: %s
Описание: %s
Контекст: %s

Предложи 3-5 конкретных пруфов для этой цели.`, 
        goal.GoalTitle,
        goal.GoalDesc,
        goal.Context,
    )
}
```

### Реализация

```go
func (s *AIService) GoalToProofs(ctx context.Context, req GoalToProofsRequest) (*GoalToProofsResponse, error) {
    // Проверяем что цель принадлежит текущему пользователю
    goal, err := s.goalRepo.GetByID(ctx, req.GoalID)
    if err != nil || goal.UserID != userIDFromCtx(ctx) {
        return nil, ErrNotFound
    }

    messages := []anthropic.Message{
        {
            Role: "user",
            Content: buildGoalBreakdownPrompt(req),
        },
    }

    resp, err := s.claude.Messages(ctx, anthropic.MessagesRequest{
        Model:     "claude-haiku-4-5-20251001",
        System:    goalBreakdownSystemPrompt,
        Messages:  messages,
        MaxTokens: 1024,
    })
    if err != nil {
        return nil, fmt.Errorf("claude api: %w", err)
    }

    var result GoalToProofsResponse
    if err := json.Unmarshal([]byte(resp.Content[0].Text), &result); err != nil {
        return nil, fmt.Errorf("parse suggestions: %w", err)
    }

    // Валидация: не более 5 suggestions, not empty fields
    if len(result.Suggestions) > 5 {
        result.Suggestions = result.Suggestions[:5]
    }

    return &result, nil
}
```

Модель: `claude-haiku-4-5-20251001` (быстрее, дешевле для этого случая).

**Кеш:** результат не кешируется — goal_desc может меняться. Повторный вызов всегда свежий.

**Rate limit:** не более 3 вызовов в 10 минут на пользователя (Redis counter).

---

## Frontend

### Интеграция в goal-setup-screen.tsx и goal-detail-screen.tsx

```tsx
// Кнопка добавляется после поля описания цели
function GoalProofSuggestions({ goalId, goalTitle, goalDesc }) {
  const [suggestions, setSuggestions] = useState<ProofSuggestion[] | null>(null);
  const [loading, setLoading] = useState(false);

  async function fetchSuggestions() {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/ai/goal-to-proofs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ goal_id: goalId, goal_title: goalTitle, goal_desc: goalDesc }),
      });
      const data = await res.json();
      setSuggestions(data.suggestions);
    } finally {
      setLoading(false);
    }
  }

  if (!suggestions) {
    return (
      <button 
        className={styles.aiBtn} 
        onClick={fetchSuggestions}
        disabled={loading}
      >
        {loading ? 'Генерирую...' : '✦ Предложить 3 пруфа'}
      </button>
    );
  }

  return (
    <div className={styles.suggestions}>
      <div className={styles.suggestionsLabel}>AI предлагает:</div>
      {suggestions.map((s, i) => (
        <SuggestionCard key={i} suggestion={s} goalId={goalId} />
      ))}
      <button className={styles.regenerateBtn} onClick={fetchSuggestions}>
        Другие варианты
      </button>
    </div>
  );
}

function SuggestionCard({ suggestion, goalId }) {
  return (
    <div className={styles.suggCard}>
      <div className={styles.suggWhat}>{suggestion.what_to_prove}</div>
      <div className={styles.suggHow}>{suggestion.how_to_prove}</div>
      <div className={styles.suggMeta}>
        <span className={styles.suggDue}>~{suggestion.due_in_days} дн.</span>
        <span className={styles.suggRationale}>{suggestion.rationale}</span>
      </div>
      <Link 
        href={`/goals/${goalId}/contract/new?prefill=${encodeURIComponent(JSON.stringify(suggestion))}`}
        className={styles.useBtn}
      >
        Использовать →
      </Link>
    </div>
  );
}
```

```css
/* В goal-setup-screen.module.css */
.aiBtn {
  width: 100%;
  padding: 12px;
  background: none;
  border: 1px dashed var(--frost);
  color: var(--frost);
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
  letter-spacing: 0.06em;
  transition: background 140ms ease;
  margin-top: 12px;
}

.aiBtn:hover { background: rgba(159, 223, 255, 0.06); }

.aiBtn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.suggestions { margin-top: 12px; }

.suggestionsLabel {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--frost);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  margin-bottom: 10px;
}

.suggCard {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-left: 2px solid var(--frost);
  padding: 14px;
  margin-bottom: 8px;
}

.suggWhat {
  font-size: 14px;
  font-weight: 700;
  color: var(--ink-primary);
  margin-bottom: 4px;
}

.suggHow {
  font-size: 12px;
  color: var(--ink-mono);
  margin-bottom: 8px;
}

.suggMeta {
  display: flex;
  gap: 12px;
  margin-bottom: 10px;
}

.suggDue {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--frost);
}

.suggRationale {
  font-size: 11px;
  color: var(--ink-mono);
  flex: 1;
}

.useBtn {
  display: inline-block;
  border: 1px solid var(--frost);
  color: var(--frost);
  padding: 8px 14px;
  font-size: 11px;
  font-weight: 700;
  text-decoration: none;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  transition: background 140ms ease;
}

.useBtn:hover { background: rgba(159, 223, 255, 0.08); }

.regenerateBtn {
  background: none;
  border: none;
  color: var(--ink-mono);
  font-size: 12px;
  cursor: pointer;
  padding: 8px 0;
  margin-top: 4px;
}
```

### Prefill в contract/new

При переходе по ссылке `?prefill=...` форма Proof Contract (спек 5.3) заполняется данными из suggestion:

```tsx
// В proof-contract-screen.tsx
const searchParams = useSearchParams();
const prefill = searchParams.get('prefill');
const prefillData = prefill ? JSON.parse(decodeURIComponent(prefill)) : null;

// Используется как initialValues для формы
const [whatToProve, setWhatToProve] = useState(prefillData?.what_to_prove ?? '');
const [howToProve, setHowToProve] = useState(prefillData?.how_to_prove ?? '');
```

---

## Acceptance Criteria

- [ ] `POST /v1/ai/goal-to-proofs` возвращает 3-5 suggestions в JSON
- [ ] Rate limit: 3 вызова в 10 минут на пользователя → 429 с сообщением
- [ ] Кнопка "Предложить 3 пруфа" не показывается до ввода названия цели
- [ ] Во время генерации кнопка disabled с текстом "Генерирую..."
- [ ] Каждая suggestion card имеет кнопку "Использовать →"
- [ ] Переход "Использовать →" открывает contract/new с prefill-данными
- [ ] AI кнопка стилизована в --frost (не --win), визуально отличается от обычных действий
- [ ] "Другие варианты" повторяет запрос и заменяет текущие suggestions
- [ ] 400 если goal_id не принадлежит текущему пользователю

---

## Что нельзя делать

- Не делать AI-шаг обязательным — только кнопка по желанию
- Не сохранять suggestions на бэкенде (stateless, генерируется на лету)
- Не использовать claude-opus для этого случая (слишком дорого для suggestions)
- Не показывать внутреннее промпт-содержимое пользователю
