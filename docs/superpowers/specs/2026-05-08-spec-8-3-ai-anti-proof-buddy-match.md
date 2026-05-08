# Спек 8.3 — AI: Anti-Proof Assistant + Buddy Matching

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 8 · AI-персонализация  
**Зависимости:** Спек 1.3 (Proof Contract), 3.2 (buddy metrics)  
**Сложность:** S  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

**Anti-Proof:** пользователь застрял и не может сдать пруф — AI помогает честно зафиксировать блокер и найти маленький шаг вперёд. Это не провал, это часть процесса.

**Buddy Match:** AI предлагает подходящего buddy по цели, стеку и ритму. Снижает барьер для поиска пары поддержки.

**DNA П8:** "застрял" — нормально. Помогаем, не осуждаем.

---

## Маршруты

```
POST /v1/ai/anti-proof
POST /v1/ai/buddy-match
```

---

## Backend: /v1/ai/anti-proof

### Handler

```go
type AntiProofRequest struct {
    GoalTitle    string `json:"goal_title"`
    ContractWhat string `json:"contract_what"` // что планировал доказать
    Blocker      string `json:"blocker"`        // что мешает (от пользователя)
    Tried        string `json:"tried"`          // что уже пробовал
}

type AntiProofResponse struct {
    Acknowledgment string   `json:"acknowledgment"` // "Понятно, это реальный стопор"
    MicroStep      string   `json:"micro_step"`     // один маленький шаг прямо сейчас
    ReframeOptions []string `json:"reframe_options"` // 2-3 переформулировки цели
    AntiProofText  string   `json:"anti_proof_text"` // готовый текст анти-пруфа
}
```

### Prompt

```go
const antiProofSystemPrompt = `Ты помощник ProofForge. Пользователь застрял и не смог сдать пруф по плану.

Твоя задача:
1. Признать что блокер реальный (не обесценивать)
2. Предложить один микро-шаг — самое маленькое действие которое можно сделать прямо сейчас
3. Предложить 2-3 варианта переформулировки цели/пруфа (если нужно скорректировать)
4. Написать текст анти-пруфа — честная фиксация что застрял и почему

Тон: без осуждения, как хороший коллега. Не "ты должен", а "можно попробовать".
Язык: русский.
Ответ в JSON.`
```

### Реализация

```go
func (s *AIService) AntiProof(ctx context.Context, req AntiProofRequest) (*AntiProofResponse, error) {
    prompt := fmt.Sprintf(`Цель: %s
Что планировал доказать: %s
Блокер: %s
Что уже пробовал: %s

Помоги разобраться.`,
        req.GoalTitle,
        req.ContractWhat,
        req.Blocker,
        req.Tried,
    )

    resp, err := s.claude.Messages(ctx, anthropic.MessagesRequest{
        Model:     "claude-haiku-4-5-20251001",
        System:    antiProofSystemPrompt,
        Messages:  []anthropic.Message{{Role: "user", Content: prompt}},
        MaxTokens: 800,
    })
    if err != nil {
        return nil, err
    }

    var result AntiProofResponse
    if err := json.Unmarshal([]byte(resp.Content[0].Text), &result); err != nil {
        return nil, err
    }
    if len(result.ReframeOptions) > 3 {
        result.ReframeOptions = result.ReframeOptions[:3]
    }
    return &result, nil
}
```

**Кеш:** не кешируется. Контекст каждый раз разный.  
**Rate limit:** 5 вызовов в час на пользователя.

---

## Backend: /v1/ai/buddy-match

### Handler

```go
type BuddyMatchRequest struct {
    GoalTitle string `json:"goal_title"`
    GoalDesc  string `json:"goal_desc,omitempty"`
    Tags      []string `json:"tags"` // теги цели
    Rhythm    string `json:"rhythm"` // daily/weekly/biweekly
}

type BuddyCandidate struct {
    UserID      string   `json:"user_id"`
    DisplayName string   `json:"display_name"`
    AvatarURL   string   `json:"avatar_url,omitempty"`
    CommonTags  []string `json:"common_tags"`  // общие интересы
    ReviewCount int      `json:"review_count"` // сколько ревью дал другим
    MatchReason string   `json:"match_reason"` // почему подходит
}

type BuddyMatchResponse struct {
    Candidates []BuddyCandidate `json:"candidates"` // top 3
}
```

### Логика

Buddy Match — **не чистый AI**: это гибридный подход.

1. **SQL-запрос** находит кандидатов из того же круга/community с похожими тегами:

```sql
SELECT u.id, u.display_name, u.avatar_url,
       COUNT(DISTINCT ci.id) AS review_count,
       ARRAY_AGG(DISTINCT gt.tag) AS user_tags
FROM users u
JOIN check_ins ci ON ci.buddy_user_id = u.id AND ci.buddy_feedback IS NOT NULL
JOIN goals g ON g.user_id = u.id
JOIN goal_tags gt ON gt.goal_id = g.id
WHERE gt.tag = ANY($1)  -- теги запроса
  AND u.id != $2        -- не текущий пользователь
  AND u.id IN (         -- в том же community/circle
    SELECT user_id FROM circle_members 
    WHERE circle_id IN (SELECT circle_id FROM circle_members WHERE user_id = $2)
  )
GROUP BY u.id, u.display_name, u.avatar_url
ORDER BY review_count DESC
LIMIT 10
```

2. **Claude** получает топ-10 кандидатов + цель пользователя → выбирает топ-3 и пишет `match_reason` для каждого:

```go
const buddyMatchSystemPrompt = `Тебе дана цель пользователя и список кандидатов в buddy.
Выбери топ-3 наиболее подходящих кандидата и для каждого напиши одну фразу почему он подходит.
Учитывай: общие теги, опыт ревью, совместимость ритма.
Язык: русский. Ответ в JSON: массив из 3 объектов {user_id, match_reason}.`
```

3. Обогащаем ответ данными из SQL-запроса.

---

## Frontend: Anti-Proof Flow

Кнопка "Я застрял" добавляется в:
- страницу цели (goal-detail-screen.tsx) 
- страницу check-in создания (check-in-form.tsx)

```tsx
function ImStuckFlow({ goalId, goalTitle, contractWhat }) {
  const [step, setStep] = useState<'idle' | 'form' | 'result'>('idle');
  const [blocker, setBlocker] = useState('');
  const [tried, setTried] = useState('');
  const [result, setResult] = useState<AntiProofResponse | null>(null);
  const [submitting, setSubmitting] = useState(false);

  if (step === 'idle') {
    return (
      <button className={styles.stuckBtn} onClick={() => setStep('form')}>
        Я застрял
      </button>
    );
  }

  if (step === 'form') {
    return (
      <div className={styles.stuckForm}>
        <div className={styles.stuckTitle}>Что мешает?</div>
        <textarea
          className={styles.stuckInput}
          placeholder="Опиши что не получается..."
          value={blocker}
          onChange={e => setBlocker(e.target.value)}
          rows={3}
        />
        <textarea
          className={styles.stuckInput}
          placeholder="Что уже пробовал? (необязательно)"
          value={tried}
          onChange={e => setTried(e.target.value)}
          rows={2}
        />
        <div className={styles.stuckActions}>
          <button className={styles.cancelBtn} onClick={() => setStep('idle')}>Отмена</button>
          <button 
            className={styles.submitBtn}
            disabled={!blocker.trim() || submitting}
            onClick={async () => {
              setSubmitting(true);
              const res = await fetch('/api/v1/ai/anti-proof', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                  goal_title: goalTitle,
                  contract_what: contractWhat,
                  blocker,
                  tried,
                }),
              }).then(r => r.json());
              setResult(res);
              setStep('result');
              setSubmitting(false);
            }}
          >
            {submitting ? 'Думаю...' : 'Помогите →'}
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.stuckResult}>
      <div className={styles.acknowledge}>{result!.acknowledgment}</div>
      
      {result!.micro_step && (
        <div className={styles.microStep}>
          <div className={styles.microStepLabel}>Один шаг прямо сейчас:</div>
          <div className={styles.microStepText}>{result!.micro_step}</div>
        </div>
      )}
      
      {result!.anti_proof_text && (
        <div className={styles.antiProofSection}>
          <div className={styles.antiProofLabel}>Готовый текст анти-пруфа:</div>
          <div className={styles.antiProofText}>{result!.anti_proof_text}</div>
          <Link
            href={`/goals/${goalId}/checkins/new?prefill_anti=${encodeURIComponent(result!.anti_proof_text)}`}
            className={styles.useAntiProofBtn}
          >
            Сдать анти-пруф →
          </Link>
        </div>
      )}
      
      <button className={styles.resetBtn} onClick={() => setStep('idle')}>Закрыть</button>
    </div>
  );
}
```

```css
/* В goal-detail-screen.module.css */
.stuckBtn {
  background: none;
  border: 1px solid var(--border);
  color: var(--ink-mono);
  padding: 10px 16px;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  cursor: pointer;
  transition: border-color 140ms ease;
}

.stuckBtn:hover { border-color: var(--warn); color: var(--warn); }

.stuckForm {
  border: 1px solid var(--border);
  border-left: 3px solid var(--warn);
  padding: 16px;
  background: var(--bg-elevated);
}

.stuckTitle {
  font-size: 14px;
  font-weight: 700;
  color: var(--ink-primary);
  margin-bottom: 12px;
}

.stuckInput {
  width: 100%;
  background: var(--bg-base);
  border: 1px solid var(--border);
  color: var(--ink-primary);
  padding: 10px;
  font-size: 13px;
  resize: vertical;
  margin-bottom: 8px;
  font-family: inherit;
}

.stuckActions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  margin-top: 4px;
}

.cancelBtn {
  background: none;
  border: 1px solid var(--border);
  color: var(--ink-mono);
  padding: 8px 14px;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
}

.submitBtn {
  background: var(--ink-primary);
  border: none;
  color: var(--bg-base);
  padding: 8px 16px;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  cursor: pointer;
}

.submitBtn:disabled { opacity: 0.5; cursor: not-allowed; }

.stuckResult {
  border: 1px solid var(--border);
  border-left: 3px solid var(--frost);
  padding: 16px;
  background: var(--bg-elevated);
}

.acknowledge {
  font-size: 14px;
  color: var(--ink-primary);
  line-height: 1.5;
  margin-bottom: 14px;
}

.microStepLabel, .antiProofLabel {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--ink-mono);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  margin-bottom: 6px;
}

.microStep { margin-bottom: 14px; }

.microStepText {
  font-size: 14px;
  font-weight: 700;
  color: var(--ink-primary);
}

.antiProofSection { margin-top: 12px; }

.antiProofText {
  font-size: 13px;
  color: var(--ink-mono);
  line-height: 1.5;
  border: 1px solid var(--border);
  padding: 10px;
  margin-bottom: 10px;
  background: var(--bg-base);
}

.useAntiProofBtn {
  display: inline-block;
  border: 1px solid var(--win);
  color: var(--win);
  padding: 8px 14px;
  font-size: 12px;
  font-weight: 700;
  text-decoration: none;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.resetBtn {
  background: none;
  border: none;
  color: var(--ink-mono);
  font-size: 12px;
  cursor: pointer;
  margin-top: 12px;
  padding: 0;
}
```

---

## Frontend: Buddy Match

Кнопка "Найти buddy" добавляется в `goal-detail-screen.tsx` если у цели нет активного buddy.

```tsx
function BuddyMatchSection({ goalId, goalTitle, goalTags, goalRhythm, currentBuddyId }) {
  if (currentBuddyId) return null;
  
  const [candidates, setCandidates] = useState<BuddyCandidate[] | null>(null);
  const [loading, setLoading] = useState(false);

  async function findBuddy() {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/ai/buddy-match', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          goal_title: goalTitle,
          tags: goalTags,
          rhythm: goalRhythm,
        }),
      });
      const data = await res.json();
      setCandidates(data.candidates);
    } finally {
      setLoading(false);
    }
  }

  if (!candidates) {
    return (
      <button className={styles.buddyBtn} onClick={findBuddy} disabled={loading}>
        {loading ? 'Ищу...' : '✦ Найти buddy'}
      </button>
    );
  }

  return (
    <div className={styles.buddyCandidates}>
      <div className={styles.candidatesLabel}>Подходящие buddy:</div>
      {candidates.map(c => (
        <BuddyCandidateCard key={c.user_id} candidate={c} goalId={goalId} />
      ))}
    </div>
  );
}

function BuddyCandidateCard({ candidate, goalId }) {
  const [sent, setSent] = useState(false);
  
  async function inviteBuddy() {
    await fetch(`/api/v1/goals/${goalId}/buddy-invite`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ buddy_user_id: candidate.user_id }),
    });
    setSent(true);
  }

  return (
    <div className={styles.candidateCard}>
      <Avatar url={candidate.avatar_url} name={candidate.display_name} size={36} />
      <div className={styles.candidateInfo}>
        <div className={styles.candidateName}>{candidate.display_name}</div>
        <div className={styles.candidateReason}>{candidate.match_reason}</div>
        {candidate.common_tags.length > 0 && (
          <div className={styles.candidateTags}>
            {candidate.common_tags.slice(0, 3).map(t => (
              <span key={t} className={styles.tag}>{t}</span>
            ))}
          </div>
        )}
      </div>
      <button 
        className={styles.inviteBtn}
        onClick={inviteBuddy}
        disabled={sent}
      >
        {sent ? 'Запрос отправлен' : 'Позвать →'}
      </button>
    </div>
  );
}
```

```css
/* В goal-detail-screen.module.css */
.buddyBtn {
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
}

.buddyBtn:hover { background: rgba(159, 223, 255, 0.06); }

.candidatesLabel {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--frost);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  margin-bottom: 10px;
}

.candidateCard {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--border);
  margin-bottom: 8px;
  background: var(--bg-elevated);
}

.candidateInfo { flex: 1; min-width: 0; }

.candidateName {
  font-size: 14px;
  font-weight: 700;
  color: var(--ink-primary);
}

.candidateReason {
  font-size: 12px;
  color: var(--ink-mono);
  margin-top: 2px;
  line-height: 1.4;
}

.candidateTags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 6px;
}

.tag {
  font-size: 10px;
  color: var(--frost);
  border: 1px solid var(--frost);
  padding: 2px 6px;
  font-family: var(--font-mono);
}

.inviteBtn {
  border: 1px solid var(--border);
  background: none;
  color: var(--ink-primary);
  padding: 8px 12px;
  font-size: 11px;
  font-weight: 700;
  cursor: pointer;
  white-space: nowrap;
  transition: border-color 140ms ease;
}

.inviteBtn:hover { border-color: var(--ink-primary); }
.inviteBtn:disabled { opacity: 0.6; cursor: default; }
```

---

## Acceptance Criteria

**Anti-Proof:**
- [ ] Кнопка "Я застрял" видна на странице цели и в форме check-in
- [ ] Форма: поле "блокер" обязательно, поле "что пробовал" опционально
- [ ] "Помогите →" disabled пока blocker.trim() === ''
- [ ] Результат показывает acknowledgment, micro_step, anti_proof_text
- [ ] Кнопка "Сдать анти-пруф →" открывает /checkins/new с prefill-текстом
- [ ] Rate limit 5/час: 429 → показать "Слишком много запросов, подожди"

**Buddy Match:**
- [ ] Раздел "Найти buddy" скрыт если у цели уже есть buddy
- [ ] Показывает 3 кандидата максимум
- [ ] Кнопка "Позвать →" меняется на "Запрос отправлен" после клика (оптимистично)
- [ ] Common tags отображаются в стиле --frost тегов
- [ ] Если кандидатов не нашлось (0 из SQL) — текст "Пока нет кандидатов в твоём круге"

---

## Что нельзя делать

- Не делать anti-proof обязательным шагом перед сдачей
- Не показывать другим пользователям что кто-то застрял (private)
- Не отправлять buddy invite автоматически — только явный клик
- Не добавлять в anti-proof кнопку "Отказаться от цели" — это другой флоу
