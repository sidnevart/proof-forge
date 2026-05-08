# Спек 8.2 — AI: Weekly Next Step + Proof Pre-check

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 8 · AI-персонализация  
**Зависимости:** Спек 3.1 (personal metrics), 6.1 (personal dashboard)  
**Сложность:** S  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

**Next Step:** платформа видит историю пользователя — цели, пруфы, ритм — и предлагает один конкретный маленький шаг на эту неделю. Не список, не план, один шаг.

**Proof Pre-check:** перед отправкой пруфа AI проверяет черновик — есть ли конкретика, артефакт, вывод. Помогает пользователю сдать сильный пруф с первого раза.

---

## Маршруты

```
POST /v1/ai/next-step
POST /v1/ai/proof-check
```

---

## Backend: /v1/ai/next-step

### Handler

```go
type NextStepRequest struct {
    // Передаётся клиентом, собирается из /v1/me/stats и /v1/goals
    ActiveGoals []struct {
        Title      string `json:"title"`
        Mode       string `json:"mode"`
        LastProof  string `json:"last_proof_at,omitempty"`  // ISO date
        ProofCount int    `json:"proof_count"`
    } `json:"active_goals"`
    WeekStreak  int    `json:"week_streak"`
    WeeksTrend  string `json:"week_trend"` // better/same/worse/first_week
}

type NextStepResponse struct {
    Step        string `json:"step"`         // одно конкретное действие
    GoalTitle   string `json:"goal_title"`   // к какой цели относится
    WhyNow      string `json:"why_now"`      // короткое объяснение (1-2 предложения)
    TimeEstimate string `json:"time_estimate"` // "30 минут", "пара часов"
}
```

### Prompt

```go
const nextStepSystemPrompt = `Ты AI-помощник ProofForge. Помогаешь пользователю сделать следующий шаг к цели.

Правила:
- ОДИН шаг, не список
- Конкретное действие (сделать X), не расплывчатое (думать о Y)
- Учитывай ритм: если streak хороший — поддержи темп, если потерял — помоги войти обратно
- Объясни почему именно сейчас этот шаг
- Тон: как хороший коллега, не наставник
- Язык: русский
- Формат: JSON`

func buildNextStepPrompt(req NextStepRequest) string {
    goals := make([]string, len(req.ActiveGoals))
    for i, g := range req.ActiveGoals {
        lastProof := "нет пруфов"
        if g.LastProof != "" {
            lastProof = "последний: " + g.LastProof
        }
        goals[i] = fmt.Sprintf("- %s (режим: %s, пруфов: %d, %s)", 
            g.Title, g.Mode, g.ProofCount, lastProof)
    }
    
    return fmt.Sprintf(`Активные цели:
%s

Streak: %d недель. Тренд недели: %s.

Предложи один конкретный шаг на эту неделю.`,
        strings.Join(goals, "\n"),
        req.WeekStreak,
        req.WeeksTrend,
    )
}
```

### Cache

Результат кешируется на 6 часов по user_id (Redis). Если за 6 часов пользователь сдал пруф — инвалидировать кеш.

---

## Backend: /v1/ai/proof-check

### Handler

```go
type ProofCheckRequest struct {
    GoalTitle    string `json:"goal_title"`
    CheckInText  string `json:"check_in_text"`  // черновик пруфа (max 2000 chars)
    HasArtifact  bool   `json:"has_artifact"`   // есть ли прикреплённый файл/ссылка
}

type ProofCheckResponse struct {
    Score      int      `json:"score"`      // 0-100
    Verdict    string   `json:"verdict"`    // "strong" | "acceptable" | "weak"
    Strengths  []string `json:"strengths"`  // что хорошо (max 2)
    Gaps       []string `json:"gaps"`       // чего не хватает (max 2)
    Suggestion string   `json:"suggestion"` // одно конкретное улучшение
}
```

### Prompt

```go
const proofCheckSystemPrompt = `Ты рецензент ProofForge. Оцениваешь качество пруфа — доказательства прогресса.

Критерии сильного пруфа:
1. Конкретность: что именно сделано (не "разобрался", а "написал код X и показал Y")
2. Артефакт: ссылка, код, скриншот, демо — что-то материальное
3. Вывод: чему научился, что понял
4. Следующий шаг: что дальше

Оценивай честно, но мотивирующе. Если пруф хороший — скажи что именно хорошо.
Если слабый — скажи что одно конкретно добавить/изменить.

Ответ в JSON. Язык: русский.`

func buildProofCheckPrompt(req ProofCheckRequest) string {
    artifact := "артефакт отсутствует"
    if req.HasArtifact {
        artifact = "артефакт прикреплён"
    }
    
    return fmt.Sprintf(`Цель: %s
Артефакт: %s

Текст пруфа:
%s

Оцени качество.`,
        req.GoalTitle,
        artifact,
        req.CheckInText,
    )
}
```

**Важно:** текст пруфа обрезается до 2000 символов перед отправкой в Claude. Проверка на backend.

---

## Frontend: Next Step в дашборде

Интегрируется в `dashboard-screen.tsx` ниже NowCard. Загружается вместе с основными данными.

```tsx
// В dashboard-screen.tsx
function WeeklyNextStep() {
  const { data: stats } = useSWR('/api/v1/me/stats', fetcher);
  const { data: goals } = useSWR('/api/v1/goals?status=active', fetcher);
  
  const [step, setStep] = useState<NextStepResponse | null>(null);
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    if (!stats || !goals || dismissed) return;
    
    fetch('/api/v1/ai/next-step', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        active_goals: goals.slice(0, 3).map(g => ({
          title: g.title,
          mode: g.movement_mode,
          last_proof_at: g.last_proof_at,
          proof_count: g.proof_count,
        })),
        week_streak: stats.streak,
        week_trend: stats.week_trend,
      }),
    })
      .then(r => r.json())
      .then(setStep)
      .catch(() => {}); // Не блокируем UI если AI недоступен
  }, [stats, goals, dismissed]);

  if (!step || dismissed) return null;

  return (
    <div className={styles.nextStep}>
      <div className={styles.nextStepEyebrow}>✦ СЛЕДУЮЩИЙ ШАГ</div>
      <div className={styles.nextStepGoal}>{step.goal_title}</div>
      <div className={styles.nextStepText}>{step.step}</div>
      <div className={styles.nextStepMeta}>
        <span className={styles.nextStepTime}>{step.time_estimate}</span>
        <span className={styles.nextStepWhy}>{step.why_now}</span>
      </div>
      <button className={styles.dismissBtn} onClick={() => setDismissed(true)}>
        Понял ×
      </button>
    </div>
  );
}
```

```css
/* В dashboard-screen.module.css */
.nextStep {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-left: 3px solid var(--frost);
  padding: 16px;
  margin-bottom: 16px;
  position: relative;
}

.nextStepEyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--frost);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  margin-bottom: 8px;
}

.nextStepGoal {
  font-size: 11px;
  color: var(--ink-mono);
  margin-bottom: 4px;
  font-family: var(--font-mono);
  text-transform: uppercase;
}

.nextStepText {
  font-size: 15px;
  font-weight: 700;
  color: var(--ink-primary);
  line-height: 1.3;
  margin-bottom: 10px;
}

.nextStepMeta {
  display: flex;
  gap: 12px;
  align-items: baseline;
}

.nextStepTime {
  font-size: 11px;
  color: var(--frost);
  font-family: var(--font-mono);
  white-space: nowrap;
}

.nextStepWhy {
  font-size: 12px;
  color: var(--ink-mono);
  flex: 1;
}

.dismissBtn {
  position: absolute;
  top: 12px;
  right: 12px;
  background: none;
  border: none;
  color: var(--ink-mono);
  font-size: 11px;
  cursor: pointer;
  padding: 4px;
}
```

---

## Frontend: Proof Pre-check в check-in форме

Кнопка появляется когда введено ≥50 символов в поле пруфа.

```tsx
// В check-in-form.tsx (существующий компонент)
function ProofChecker({ goalTitle, text, hasArtifact }) {
  const [result, setResult] = useState<ProofCheckResponse | null>(null);
  const [checking, setChecking] = useState(false);

  // Показывать кнопку только если есть что проверять
  if (text.length < 50) return null;

  async function checkProof() {
    setChecking(true);
    try {
      const res = await fetch('/api/v1/ai/proof-check', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          goal_title: goalTitle,
          check_in_text: text.slice(0, 2000),
          has_artifact: hasArtifact,
        }),
      });
      setResult(await res.json());
    } finally {
      setChecking(false);
    }
  }

  return (
    <div className={styles.checker}>
      {!result ? (
        <button className={styles.checkBtn} onClick={checkProof} disabled={checking}>
          {checking ? 'Проверяю...' : '✦ Проверить перед отправкой'}
        </button>
      ) : (
        <ProofCheckResult result={result} onReset={() => setResult(null)} />
      )}
    </div>
  );
}

function ProofCheckResult({ result, onReset }) {
  const verdictColor = result.verdict === 'strong' 
    ? 'var(--win)' 
    : result.verdict === 'acceptable' 
    ? 'var(--warn)' 
    : 'var(--danger)';
  
  const verdictText = {
    strong: 'Сильный пруф',
    acceptable: 'Можно лучше',
    weak: 'Доработай',
  }[result.verdict];

  return (
    <div className={styles.result}>
      <div className={styles.resultScore} style={{ color: verdictColor }}>
        {result.score}/100 — {verdictText}
      </div>
      {result.strengths.length > 0 && (
        <div className={styles.strengths}>
          {result.strengths.map((s, i) => (
            <div key={i} className={styles.strength}>✓ {s}</div>
          ))}
        </div>
      )}
      {result.gaps.length > 0 && (
        <div className={styles.gaps}>
          {result.gaps.map((g, i) => (
            <div key={i} className={styles.gap}>→ {g}</div>
          ))}
        </div>
      )}
      {result.suggestion && (
        <div className={styles.suggestion}>{result.suggestion}</div>
      )}
      <button className={styles.resetBtn} onClick={onReset}>
        Проверить снова
      </button>
    </div>
  );
}
```

```css
/* В check-in-form.module.css */
.checker { margin-top: 10px; }

.checkBtn {
  width: 100%;
  padding: 10px;
  background: none;
  border: 1px dashed var(--frost);
  color: var(--frost);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  letter-spacing: 0.06em;
  transition: background 140ms ease;
}

.checkBtn:hover { background: rgba(159, 223, 255, 0.06); }
.checkBtn:disabled { opacity: 0.5; cursor: not-allowed; }

.result {
  border: 1px solid var(--border);
  border-left: 2px solid var(--frost);
  padding: 14px;
  background: var(--bg-elevated);
}

.resultScore {
  font-family: var(--font-display);
  font-size: 18px;
  font-weight: 700;
  margin-bottom: 10px;
}

.strengths, .gaps { margin-bottom: 8px; }

.strength {
  font-size: 12px;
  color: var(--win);
  margin-bottom: 4px;
}

.gap {
  font-size: 12px;
  color: var(--ink-mono);
  margin-bottom: 4px;
}

.suggestion {
  font-size: 13px;
  color: var(--ink-primary);
  border-top: 1px solid var(--border);
  padding-top: 10px;
  margin-top: 8px;
  line-height: 1.5;
}

.resetBtn {
  background: none;
  border: none;
  color: var(--ink-mono);
  font-size: 11px;
  cursor: pointer;
  padding: 8px 0 0;
  text-decoration: underline;
}
```

---

## Acceptance Criteria

**Next Step:**
- [ ] Загружается автоматически на дашборде (без клика пользователя)
- [ ] Не блокирует загрузку дашборда (отдельный async запрос)
- [ ] "Понял ×" скрывает блок на всю сессию (не persistent)
- [ ] Кеш 6ч на backend: повторные запросы не дёргают Claude API
- [ ] Если AI недоступен (таймаут/ошибка) — блок просто не отображается

**Proof Pre-check:**
- [ ] Кнопка появляется только при ≥50 символах в поле
- [ ] Текст обрезается до 2000 символов перед отправкой
- [ ] Score отображается числом и цветом: ≥70 win, 50-69 warn, <50 danger
- [ ] "Проверить снова" сбрасывает результат для повторной проверки
- [ ] Не блокирует отправку пруфа — это рекомендация, не валидация

---

## Что нельзя делать

- Не делать проверку пруфа обязательной — пользователь может сдать без неё
- Не показывать внутренние оценки Claude (raw API response)
- Не кешировать proof-check (каждая проверка на свежий черновик)
- Не добавлять Auto-check при каждом keystroke (только по кнопке)
