# Спек 8.4 — AI: Growth Dossier Generator

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 8 · AI-персонализация  
**Зависимости:** Спек 3.1 (личные метрики), 4.4 (достижения)  
**Сложность:** M  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Growth Dossier — персональный итог периода для 1:1 с менеджером или ИПР-встречи. AI собирает цели, пруфы, подтверждения buddy, достижения, топ-навыки и формирует структурированный документ. Экспорт в Markdown или текст.

**DNA П7:** платформа помогает пользователю выглядеть сильным — не в глазах других, а на своих встречах.

---

## Маршрут

```
POST /v1/ai/growth-dossier      ← генерация
GET  /v1/me/dossiers            ← список сохранённых
GET  /v1/me/dossiers/:id        ← конкретный
```

UI: `/me/dossier/new` — страница генерации, `/me/dossier/:id` — просмотр.

---

## Backend

### Миграция

```sql
-- 00026_dossiers.sql
CREATE TABLE dossiers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    period_from DATE NOT NULL,
    period_to   DATE NOT NULL,
    title       TEXT NOT NULL,
    content     TEXT NOT NULL,         -- Markdown
    generated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT dossiers_period CHECK (period_from < period_to)
);

CREATE INDEX idx_dossiers_user_id ON dossiers(user_id, generated_at DESC);
```

### Handler

```go
type DossierRequest struct {
    PeriodFrom string `json:"period_from"` // "2025-01-01"
    PeriodTo   string `json:"period_to"`   // "2025-03-31"
    Context    string `json:"context,omitempty"` // "для 1:1 с менеджером", "для ИПР"
}

type DossierResponse struct {
    ID          string `json:"id"`
    Title       string `json:"title"`
    Content     string `json:"content"` // Markdown
    GeneratedAt string `json:"generated_at"`
}
```

### Сбор данных

Перед запросом к Claude собираем данные за период:

```go
type DossierContext struct {
    User struct {
        Name string
    }
    Period struct {
        From, To string
    }
    Goals []struct {
        Title       string
        Mode        string
        ProofCount  int
        Status      string
        TopProofs   []string // первые 150 символов каждого
    }
    Stats struct {
        TotalProofs        int
        BuddyApprovedCount int
        WeeksActive        int
        Streak             int
        PersonalRecord     int
    }
    Achievements []struct {
        Name, Description string
    }
    TopSkills []string // теги из пруфов, топ-5 по частоте
    BuddyFeedback []string // тексты buddy feedback, первые 200 символов
}
```

```go
func (s *AIService) buildDossierContext(ctx context.Context, userID string, from, to time.Time) (*DossierContext, error) {
    var dc DossierContext
    // 1. Данные пользователя
    // 2. Цели за период + пруфы
    // 3. Статистика из stats сервиса
    // 4. Достижения разблокированные за период
    // 5. Топ-теги из goal_tags за период
    // 6. Buddy feedback snippets
    return &dc, nil
}
```

### Prompt

```go
const dossierSystemPrompt = `Ты помогаешь специалисту подготовить документ о своём росте за период.
Задача: на основе данных о целях, пруфах и достижениях написать Growth Dossier.

Структура документа:
# [Имя] — Прогресс [Период]

## Итог периода
(2-3 предложения: главное что было сделано)

## Цели и результаты
(Список целей с кратким итогом по каждой)

## Доказанные навыки
(Список навыков с конкретными примерами из пруфов)

## Лучшие моменты
(2-3 конкретных достижения с цитатами из пруфов)

## Что дальше
(Логичный следующий шаг на основе текущего прогресса)

Правила:
- Конкретика: числа, примеры, цитаты из пруфов
- Тон: профессиональный но живой
- Язык: русский
- Формат: Markdown`

func buildDossierPrompt(dc *DossierContext, userContext string) string {
    // Форматируем данные в читаемый текст для Claude
    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("Имя: %s\n", dc.User.Name))
    sb.WriteString(fmt.Sprintf("Период: %s — %s\n", dc.Period.From, dc.Period.To))
    if userContext != "" {
        sb.WriteString(fmt.Sprintf("Контекст: %s\n", userContext))
    }
    sb.WriteString("\nЦели:\n")
    for _, g := range dc.Goals {
        sb.WriteString(fmt.Sprintf("- %s (%d пруфов, статус: %s)\n", g.Title, g.ProofCount, g.Status))
        for _, p := range g.TopProofs {
            sb.WriteString(fmt.Sprintf("  Пруф: %s\n", p))
        }
    }
    sb.WriteString(fmt.Sprintf("\nИтого пруфов: %d, подтверждено buddy: %d\n", dc.Stats.TotalProofs, dc.Stats.BuddyApprovedCount))
    sb.WriteString(fmt.Sprintf("Активных недель: %d, текущий streak: %d\n", dc.Stats.WeeksActive, dc.Stats.Streak))
    sb.WriteString(fmt.Sprintf("Топ навыки: %s\n", strings.Join(dc.TopSkills, ", ")))
    if len(dc.Achievements) > 0 {
        sb.WriteString("Достижения:\n")
        for _, a := range dc.Achievements {
            sb.WriteString(fmt.Sprintf("- %s: %s\n", a.Name, a.Description))
        }
    }
    sb.WriteString("\nСоздай Growth Dossier.")
    return sb.String()
}
```

### Реализация

```go
func (s *AIService) GenerateDossier(ctx context.Context, userID string, req DossierRequest) (*DossierResponse, error) {
    from, _ := time.Parse("2006-01-02", req.PeriodFrom)
    to, _ := time.Parse("2006-01-02", req.PeriodTo)
    
    if to.Sub(from) > 365*24*time.Hour {
        return nil, ErrPeriodTooLong // максимум год
    }

    dc, err := s.buildDossierContext(ctx, userID, from, to)
    if err != nil {
        return nil, err
    }

    if dc.Stats.TotalProofs == 0 {
        return nil, ErrNothingToReport
    }

    resp, err := s.claude.Messages(ctx, anthropic.MessagesRequest{
        Model:     "claude-sonnet-4-6",  // Sonnet для качественного текста
        System:    dossierSystemPrompt,
        Messages:  []anthropic.Message{{Role: "user", Content: buildDossierPrompt(dc, req.Context)}},
        MaxTokens: 2048,
    })
    if err != nil {
        return nil, err
    }

    content := resp.Content[0].Text
    title := fmt.Sprintf("%s — Прогресс %s — %s", dc.User.Name, req.PeriodFrom, req.PeriodTo)

    // Сохраняем в БД
    dossierID, err := s.dossierRepo.Save(ctx, userID, from, to, title, content)
    if err != nil {
        return nil, err
    }

    return &DossierResponse{
        ID:          dossierID,
        Title:       title,
        Content:     content,
        GeneratedAt: time.Now().Format(time.RFC3339),
    }, nil
}
```

**Модель:** `claude-sonnet-4-6` — для качественного связного текста.  
**Кеш:** не кешируется; каждый дошье уникален и сохраняется в БД.  
**Rate limit:** 2 генерации в день на пользователя.

---

## Frontend

### Страница /me/dossier/new

```tsx
export function DossierNewPage() {
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [context, setContext] = useState('');
  const [generating, setGenerating] = useState(false);
  const router = useRouter();

  // Пресеты периода
  const presets = [
    { label: 'Этот месяц', from: startOfMonth(), to: today() },
    { label: 'Последние 3 месяца', from: monthsAgo(3), to: today() },
    { label: 'Этот квартал', from: startOfQuarter(), to: today() },
    { label: 'Полгода', from: monthsAgo(6), to: today() },
  ];

  async function generate() {
    setGenerating(true);
    try {
      const res = await fetch('/api/v1/ai/growth-dossier', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ period_from: from, period_to: to, context }),
      });
      const data = await res.json();
      if (res.ok) {
        router.push(`/me/dossier/${data.id}`);
      }
    } finally {
      setGenerating(false);
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.eyebrow}>GROWTH DOSSIER</div>
      <div className={styles.title}>Документ для 1:1 и ИПР</div>
      <div className={styles.desc}>
        AI соберёт твои цели, пруфы и достижения в готовый документ о росте
      </div>

      <div className={styles.section}>
        <div className={styles.label}>Период</div>
        <div className={styles.presets}>
          {presets.map(p => (
            <button
              key={p.label}
              className={`${styles.preset} ${from === p.from && to === p.to ? styles.active : ''}`}
              onClick={() => { setFrom(p.from); setTo(p.to); }}
            >
              {p.label}
            </button>
          ))}
        </div>
        <div className={styles.dateRange}>
          <input type="date" value={from} onChange={e => setFrom(e.target.value)} className={styles.dateInput} />
          <span>—</span>
          <input type="date" value={to} onChange={e => setTo(e.target.value)} className={styles.dateInput} />
        </div>
      </div>

      <div className={styles.section}>
        <div className={styles.label}>Для чего (необязательно)</div>
        <input
          className={styles.contextInput}
          placeholder="Например: для встречи с менеджером, для ИПР..."
          value={context}
          onChange={e => setContext(e.target.value)}
        />
      </div>

      <button
        className={styles.generateBtn}
        onClick={generate}
        disabled={!from || !to || generating}
      >
        {generating ? (
          <span>Собираю дошье...</span>
        ) : (
          <span>✦ Сгенерировать дошье</span>
        )}
      </button>

      {generating && (
        <div className={styles.generatingHint}>
          Это занимает 10–20 секунд
        </div>
      )}
    </div>
  );
}
```

### Страница /me/dossier/:id — просмотр и экспорт

```tsx
export function DossierViewPage({ params }) {
  const { data } = useSWR(`/api/v1/me/dossiers/${params.id}`, fetcher);
  
  if (!data) return <PageSkeleton />;

  function downloadMarkdown() {
    const blob = new Blob([data.content], { type: 'text/markdown' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${data.title}.md`;
    a.click();
    URL.revokeObjectURL(url);
  }

  function copyToClipboard() {
    navigator.clipboard.writeText(data.content);
  }

  return (
    <div className={styles.page}>
      <div className={styles.header}>
        <div className={styles.title}>{data.title}</div>
        <div className={styles.actions}>
          <button className={styles.actionBtn} onClick={copyToClipboard}>Копировать</button>
          <button className={styles.actionBtn} onClick={downloadMarkdown}>Скачать .md</button>
        </div>
      </div>
      
      <div className={styles.content}>
        <MarkdownRenderer content={data.content} />
      </div>
      
      <div className={styles.footer}>
        <Link href="/me/dossier/new" className={styles.newBtn}>
          Создать новый →
        </Link>
      </div>
    </div>
  );
}
```

```css
/* dossier.module.css */
.page {
  max-width: 720px;
  margin: 0 auto;
  padding: 24px 16px;
  padding-bottom: calc(80px + env(safe-area-inset-bottom));
}

.eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: var(--frost);
  margin-bottom: 8px;
}

.title {
  font-family: var(--font-display);
  font-size: clamp(22px, 4vw, 32px);
  font-weight: 700;
  color: var(--ink-primary);
  margin-bottom: 8px;
}

.desc {
  font-size: 14px;
  color: var(--ink-mono);
  line-height: 1.5;
  margin-bottom: 32px;
}

.section { margin-bottom: 24px; }

.label {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--ink-mono);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  margin-bottom: 8px;
}

.presets {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.preset {
  background: none;
  border: 1px solid var(--border);
  color: var(--ink-mono);
  padding: 8px 12px;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all 140ms ease;
}

.preset:hover { border-color: var(--ink-primary); color: var(--ink-primary); }
.preset.active { border-color: var(--frost); color: var(--frost); }

.dateRange {
  display: flex;
  align-items: center;
  gap: 8px;
}

.dateInput {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  color: var(--ink-primary);
  padding: 8px 10px;
  font-size: 13px;
  font-family: var(--font-mono);
  flex: 1;
}

.contextInput {
  width: 100%;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  color: var(--ink-primary);
  padding: 10px;
  font-size: 13px;
  font-family: inherit;
}

.generateBtn {
  width: 100%;
  background: var(--frost);
  color: var(--bg-base);
  border: none;
  padding: 16px;
  font-size: 14px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  cursor: pointer;
  transition: opacity 140ms ease;
}

.generateBtn:disabled { opacity: 0.5; cursor: not-allowed; }

.generatingHint {
  text-align: center;
  font-size: 12px;
  color: var(--ink-mono);
  margin-top: 12px;
}

/* View page */
.header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 24px;
  flex-wrap: wrap;
}

.actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.actionBtn {
  background: none;
  border: 1px solid var(--border);
  color: var(--ink-mono);
  padding: 8px 14px;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all 140ms ease;
}

.actionBtn:hover { border-color: var(--ink-primary); color: var(--ink-primary); }

/* Markdown контент */
.content {
  border: 1px solid var(--border);
  padding: 24px;
  background: var(--bg-surface);
  line-height: 1.7;
  font-size: 14px;
  color: var(--ink-primary);
}

.content h1, .content h2, .content h3 {
  font-family: var(--font-display);
  color: var(--ink-primary);
  margin: 24px 0 12px;
}

.content h1 { font-size: 22px; }
.content h2 { font-size: 17px; border-bottom: 1px solid var(--border); padding-bottom: 8px; }
.content h3 { font-size: 14px; color: var(--ink-mono); }

.content ul { padding-left: 20px; }
.content li { margin-bottom: 6px; }
.content strong { color: var(--ink-primary); }
.content em { color: var(--ink-mono); }

.footer { margin-top: 24px; text-align: right; }

.newBtn {
  font-size: 13px;
  color: var(--ink-mono);
  text-decoration: none;
  font-weight: 700;
}
```

### Ссылка из дашборда

В `dashboard-screen.tsx` добавляется ненавязчивая ссылка — только если у пользователя ≥3 пруфа:

```tsx
{stats && stats.totalProofs >= 3 && (
  <Link href="/me/dossier/new" className={styles.dossierLink}>
    ✦ Создать дошье для 1:1 →
  </Link>
)}
```

---

## Acceptance Criteria

- [ ] `POST /v1/ai/growth-dossier` возвращает Markdown-документ и сохраняет в БД
- [ ] Rate limit: 2 генерации в день → 429 с текстом "Дошье можно создавать 2 раза в день"
- [ ] Период не может превышать 365 дней → 400
- [ ] Если нет пруфов за период → 400 "Нет данных за этот период"
- [ ] Пресеты автоматически заполняют даты (корректные диапазоны)
- [ ] Во время генерации: кнопка disabled + подсказка "Это занимает 10–20 секунд"
- [ ] Кнопка "Копировать" копирует полный Markdown в буфер
- [ ] Кнопка "Скачать .md" скачивает файл с правильным именем
- [ ] Ссылка на /me/dossier/new появляется в дашборде только при ≥3 пруфах
- [ ] Список `/me/dossiers` показывает историю дошье (дата, название)

---

## Что нельзя делать

- Не включать в дошье данные других пользователей
- Не использовать claude-haiku для генерации (нужно качество — только Sonnet)
- Не делать генерацию синхронной с >30 сек таймаутом (рассмотреть async через polling)
- Не показывать raw prompt пользователю
