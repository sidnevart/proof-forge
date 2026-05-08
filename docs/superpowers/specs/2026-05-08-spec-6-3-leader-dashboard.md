# Спек 6.3 — Leader Dashboard (Teamspace + Community)

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 6 · Frontend — Дашборды  
**Зависимости:** Спек 3.3, 3.4 (analytics API), 4.4 (needs-help API)  
**Сложность:** M  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Тимлид видит **энергию команды** — кто движется, какие темы вызывают интерес, где нужна помощь. Лидер сообщества видит **здоровье сезона** — retention, лучшие пруфы, готовность к следующему сезону.

**DNA П5, П6:** публично вклад, риски — агрегированно и приватно. Нет публичного ранжирования участников.

---

## Маршруты

```
/teamspaces/:id/analytics          ← teamspace leader dashboard
/community-spaces/:id/analytics    ← community leader dashboard
```

Доступ: teamspace_lead / trusted_approver для teamspace; community_leader для community.

---

## Файловая структура

```
web/app/(product)/teamspaces/[id]/analytics/
  page.tsx

web/app/(product)/community-spaces/[id]/analytics/
  page.tsx

web/components/product/
  teamspace-analytics.tsx
  teamspace-analytics.module.css
  community-analytics.tsx
  community-analytics.module.css
  analytics-stat-card.tsx        ← переиспользуемый stat card
  analytics-stat-card.module.css
  activity-trend-chart.tsx       ← мини-chart активности
```

---

## Teamspace Analytics UI

### Мобайл — порядок блоков:

```
┌─────────────────────────────┐
│ ← Команда персонализации    │
│   Аналитика            [4н▾]│  period selector
├─────────────────────────────┤
│ ОБЗОР                       │
│ ┌──────────┐ ┌──────────┐   │
│ │ 7 из 9   │ │ 23 пруфа │   │  stat cards
│ │ АКТИВНЫ  │ │ ЗА ПЕРИОД│   │
│ └──────────┘ └──────────┘   │
│ ┌──────────┐ ┌──────────┐   │
│ │ 18.4ч    │ │ 12 ИПР   │   │
│ │ СР РЕВЬЮ │ │ АРТЕФАКТ │   │
│ └──────────┘ └──────────┘   │
├─────────────────────────────┤
│ АКТИВНОСТЬ                  │
│ [spark trend chart]         │
├─────────────────────────────┤
│ ПОПУЛЯРНЫЕ ТЕМЫ             │
│ ████ Kotlin        8 пруфов │
│ ███  AI-инструменты 6       │
│ ██   System Design  5       │
├─────────────────────────────┤
│ НУЖНА ПОМОЩЬ                │
│ ⚠ 2 участника без пруфа    │
│   более 2 недель            │
│ [Посмотреть подробнее →]    │
├─────────────────────────────┤
│ ЛУЧШИЕ ПРУФЫ                │
│ [ProofCard] [ProofCard]     │
└─────────────────────────────┘
```

### Десктоп — 2-col grid для stat cards, full-width chart:

Stat cards в ряд из 4. Популярные темы + needs-help side by side.

---

## Компонент: analytics-stat-card.tsx

```tsx
interface StatCardProps {
  label: string;
  value: string | number;
  sublabel?: string;
  accent?: 'win' | 'warn' | 'danger' | 'neutral';
}

export function StatCard({ label, value, sublabel, accent = 'neutral' }: StatCardProps) {
  return (
    <div className={`${styles.card} ${styles[accent]}`}>
      <div className={styles.value}>{value}</div>
      <div className={styles.label}>{label}</div>
      {sublabel && <div className={styles.sublabel}>{sublabel}</div>}
    </div>
  );
}
```

```css
/* analytics-stat-card.module.css */
.card {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  padding: 16px;
  flex: 1;
  min-width: 0;
}

.card.win { border-top: 3px solid var(--win); }
.card.warn { border-top: 3px solid var(--warn); }
.card.danger { border-top: 3px solid var(--danger); }

.value {
  font-family: var(--font-display);
  font-size: clamp(24px, 4vw, 36px);
  font-weight: 700;
  color: var(--ink-primary);
  margin-bottom: 4px;
}

.label {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: var(--ink-mono);
}

.sublabel {
  font-size: 12px;
  color: var(--ink-mono);
  margin-top: 4px;
}
```

---

## Популярные темы — горизонтальный bar chart

```tsx
function TopicsChart({ topics }: { topics: Topic[] }) {
  const max = topics[0]?.proof_count ?? 1;
  return (
    <div className={styles.topics}>
      {topics.slice(0, 5).map(t => (
        <div key={t.tag} className={styles.topicRow}>
          <div className={styles.topicName}>{t.tag}</div>
          <div className={styles.topicBarWrap}>
            <div 
              className={styles.topicBar}
              style={{ width: `${(t.proof_count / max) * 100}%` }}
            />
          </div>
          <div className={styles.topicCount}>{t.proof_count}</div>
        </div>
      ))}
    </div>
  );
}
```

```css
.topicRow {
  display: grid;
  grid-template-columns: 120px 1fr 40px;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.topicBarWrap {
  height: 6px;
  background: var(--bg-elevated);
}

.topicBar {
  height: 100%;
  background: var(--win);
  transition: width 400ms ease;
}

.topicCount {
  font-size: 12px;
  color: var(--ink-mono);
  text-align: right;
}
```

---

## Нужна помощь — агрегированный блок

```tsx
function NeedsHelpSummary({ summary, teamspaceId }) {
  if (summary.members_needing_attention === 0) return null;
  
  return (
    <div className={styles.attention}>
      <span className={styles.attentionIcon}>⚠</span>
      <div>
        <div className={styles.attentionText}>
          {summary.members_needing_attention} участника без пруфа более 2 недель
        </div>
        <Link href={`/teamspaces/${teamspaceId}/needs-help`} className={styles.attentionLink}>
          Посмотреть подробнее →
        </Link>
      </div>
    </div>
  );
}
```

**Важно:** блок показывает только число, без имён. Имена только на странице `/needs-help` (спек 7.3).

---

## Community Analytics UI

Аналогичная структура, но с другими метриками:

- Stat cards: всего участников, retention %, пруфов за период, completion %
- Circle engagement таблица (название, статус, completion)
- Лучшие пруфы недели (с кнопкой "Добавить в витрину")
- Potential mentors список

```tsx
export function CommunityAnalytics({ communityId }) {
  const { data } = useSWR(`/api/v1/community-spaces/${communityId}/analytics`, fetcher);
  
  return (
    <div className={styles.screen}>
      <StatGrid>
        <StatCard label="УЧАСТНИКОВ" value={data.summary.total_members} />
        <StatCard label="RETENTION" value={`${data.summary.retention_pct}%`} accent="win" />
        <StatCard label="ПРУФОВ" value={data.summary.total_proofs_this_season} />
        <StatCard label="ЗАВЕРШИЛИ" value={`${data.summary.completion_rate_pct}%`} />
      </StatGrid>
      
      <CircleEngagementTable circles={data.circle_engagement} />
      <BestProofsList proofs={data.best_proofs_week} communityId={communityId} />
      <PotentialMentorsList mentors={data.potential_mentors} />
    </div>
  );
}
```

---

## Period Selector

Dropdown в header: "Эта неделя" / "4 недели" / "12 недель"

```tsx
// Меняет ?period= в URL, SWR перезапрашивает
const periods = [
  { value: 'last_week', label: 'Эта неделя' },
  { value: 'last_4_weeks', label: '4 недели' },
  { value: 'last_12_weeks', label: '12 недель' },
];
```

---

## Acceptance Criteria

- [ ] 403 для не-лидеров при доступе к analytics
- [ ] Stat cards рендерятся с правильными значениями из API
- [ ] Topics bar chart нормализован по максимальному значению
- [ ] NeedsHelpSummary показывает только число, без имён
- [ ] Period selector меняет данные без перезагрузки страницы
- [ ] Community: кнопка "Добавить в витрину" вызывает POST `/best-proofs/:id/feature`
- [ ] Мобайл: stat cards 2×2 grid; десктоп: 4 в ряд
- [ ] Пустые данные (0 proofs) — корректные нули, не ошибки

---

## Что нельзя делать

- Не показывать имена участников в "нужна помощь" секции (только число)
- Не добавлять рейтинг участников по активности
- Не давать trusted_approver доступ к `/members` детальной аналитике
