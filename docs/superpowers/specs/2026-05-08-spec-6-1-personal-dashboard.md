# Спек 6.1 — Персональный дашборд с умной карточкой

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 6 · Frontend — Дашборды  
**Зависимости:** Спек 3.1 (GET /v1/me/now, GET /v1/me/stats)  
**Сложность:** M  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Главный экран продукта. Пользователь должен увидеть: **что делать прямо сейчас** (умная карточка) + **свой прогресс** (мягкое сравнение с собой). Никакого публичного рейтинга.

**DNA Раздел 4 — UX-паттерн «Что сейчас»** реализуется именно здесь.

---

## Файловая структура

Изменения вносятся в существующие файлы:

```
web/components/product/
  dashboard-screen.tsx              ← обновить (добавить умную карточку и stats)
  dashboard-screen.module.css       ← обновить
  now-card.tsx                      ← НОВЫЙ компонент умной карточки
  now-card.module.css               ← НОВЫЙ
  personal-progress-bar.tsx         ← НОВЫЙ
  personal-progress-bar.module.css  ← НОВЫЙ
```

---

## Layout дашборда

### Мобайл (≤640px) — порядок блоков сверху вниз:

```
┌─────────────────────────────┐
│ ProofForge         [👤]     │  header
├─────────────────────────────┤
│ ┌─────────────────────────┐ │
│ │ СЕЙЧАС              ↗   │ │  ← NowCard (главная)
│ │                         │ │
│ │ Дедлайн завтра          │ │
│ │ «Kotlin Coroutines»     │ │
│ │ демо SupervisorJob      │ │
│ │                         │ │
│ │ [→ СДАТЬ ПРУФ СЕЙЧАС]  │ │
│ └─────────────────────────┘ │
├─────────────────────────────┤
│ МОЙ ПРОГРЕСС               │  ← PersonalProgressBar
│                             │
│ Эта неделя     ↑ Лучше     │
│ ██████░░  2 пруфа          │
│           чем неделю назад │
│                             │
│ Streak: ████ 4 недели      │
│ Рекорд: 7 недель           │
├─────────────────────────────┤
│ МОИ ЦЕЛИ                   │  ← существующий GoalCard список
│ ...                         │
└─────────────────────────────┘
[Главная] [Пруфы] [Круг] [Я]  ← BottomNav
```

### Десктоп (>640px) — 2 колонки:

```
┌──────────────────────────────────────────┐
│ ProofForge                    [Профиль]  │
├──────────┬───────────────────────────────┤
│ Sidebar  │ ┌─────────────────────────┐   │
│          │ │ СЕЙЧАС            ↗    │   │  NowCard
│          │ └─────────────────────────┘   │
│          │                               │
│          │ МОЙ ПРОГРЕСС                  │  PersonalProgressBar
│          │                               │
│          │ МОИ ЦЕЛИ                      │
│          │ [Goal] [Goal] [Goal]           │  2-col grid
└──────────┴───────────────────────────────┘
```

---

## Компонент: NowCard

```tsx
interface NowCardData {
  type: string;
  title: string;
  subtitle: string;
  urgency: 'danger' | 'fire' | 'warn' | 'win' | 'neutral';
  action: { label: string; url: string } | null;
}

export function NowCard({ data }: { data: NowCardData }) {
  const urgencyColor = {
    danger: 'var(--danger)',
    fire: 'var(--fire)',
    warn: 'var(--warn)',
    win: 'var(--win)',
    neutral: 'var(--border)',
  }[data.urgency];

  return (
    <div className={styles.card} style={{ borderLeftColor: urgencyColor }}>
      <div className={styles.eyebrow}>СЕЙЧАС</div>
      <div className={styles.title}>{data.title}</div>
      {data.subtitle && (
        <div className={styles.subtitle}>{data.subtitle}</div>
      )}
      {data.action && (
        <Link href={data.action.url} className={styles.cta}>
          → {data.action.label}
        </Link>
      )}
    </div>
  );
}
```

### CSS: now-card.module.css

```css
.card {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-left: 4px solid var(--border); /* override через style prop */
  padding: clamp(16px, 3vw, 24px);
  margin-bottom: 16px;
}

.eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: var(--ink-mono);
  margin-bottom: 10px;
}

.title {
  font-family: var(--font-display);
  font-size: clamp(18px, 3vw, 24px);
  font-weight: 700;
  color: var(--ink-primary);
  margin-bottom: 6px;
  line-height: 1.2;
}

.subtitle {
  font-size: 14px;
  color: var(--ink-mono);
  line-height: 1.5;
  margin-bottom: 16px;
}

.cta {
  display: inline-block;
  background: var(--ink-primary);
  color: var(--bg-base);
  padding: 10px 18px;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  text-decoration: none;
  transition: opacity 140ms ease;
  min-height: 40px;
  line-height: 20px;
}

.cta:hover { opacity: 0.8; }
```

---

## Компонент: PersonalProgressBar

```tsx
interface PersonalStats {
  proofsThisWeek: number;
  proofsLastWeek: number;
  weekTrend: 'better' | 'same' | 'worse' | 'first_week';
  streak: number;
  personalRecordWeeks: number;
  weeklyHistory: { week: string; proofsCount: number }[];
}

export function PersonalProgressBar({ stats }: { stats: PersonalStats }) {
  const maxProofs = Math.max(...stats.weeklyHistory.map(w => w.proofsCount), 1);
  
  return (
    <section className={styles.section}>
      <div className={styles.eyebrow}>МОЙ ПРОГРЕСС</div>
      
      <div className={styles.thisWeek}>
        <div className={styles.weekCount}>
          <span className={styles.count}>{stats.proofsThisWeek}</span>
          <span className={styles.countLabel}> пруфов на этой неделе</span>
        </div>
        <TrendBadge trend={stats.weekTrend} delta={stats.proofsThisWeek - stats.proofsLastWeek} />
      </div>
      
      {/* Спаркчарт последних 8 недель */}
      <SparkChart weeks={stats.weeklyHistory} maxValue={maxProofs} />
      
      <div className={styles.streak}>
        <StreakBar current={stats.streak} record={stats.personalRecordWeeks} />
      </div>
    </section>
  );
}

// TrendBadge — показывает "+1 лучше чем неделю назад" или "как неделю назад"
function TrendBadge({ trend, delta }) {
  if (trend === 'first_week') return null;
  const text = trend === 'better' 
    ? `+${delta} лучше чем неделю назад`
    : trend === 'same' 
    ? 'как неделю назад'
    : `${delta} меньше чем неделю назад`;
  const color = trend === 'better' ? 'var(--win)' : trend === 'worse' ? 'var(--warn)' : 'var(--ink-mono)';
  return <span style={{ color, fontSize: 12 }}>{text}</span>;
}

// SparkChart — мини-бар-чарт из 8 баров
function SparkChart({ weeks, maxValue }) {
  return (
    <div className={styles.sparkChart}>
      {weeks.map((w) => (
        <div key={w.week} className={styles.sparkBar}>
          <div 
            className={styles.sparkFill}
            style={{ height: `${(w.proofsCount / maxValue) * 100}%` }}
          />
          <div className={styles.sparkWeekLabel}>
            {formatWeekShort(w.week)} {/* "Пн" или номер недели */}
          </div>
        </div>
      ))}
    </div>
  );
}
```

### CSS: personal-progress-bar.module.css

```css
.section {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  padding: clamp(16px, 3vw, 20px);
  margin-bottom: 16px;
}

.eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: var(--ink-mono);
  margin-bottom: 12px;
}

.thisWeek {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.count {
  font-family: var(--font-display);
  font-size: clamp(32px, 5vw, 48px);
  font-weight: 700;
  color: var(--ink-primary);
}

.countLabel {
  font-size: 14px;
  color: var(--ink-mono);
}

.sparkChart {
  display: flex;
  gap: 4px;
  height: 48px;
  align-items: flex-end;
  margin-bottom: 12px;
}

.sparkBar {
  flex: 1;
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  align-items: center;
  gap: 4px;
}

.sparkFill {
  width: 100%;
  background: var(--win);
  min-height: 2px;
  transition: height 300ms ease;
}

.sparkWeekLabel {
  font-size: 9px;
  color: var(--ink-mono);
  font-family: var(--font-mono);
}

.streak {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.streakValue {
  color: var(--win);
  font-weight: 700;
}

.streakRecord {
  color: var(--ink-mono);
}
```

---

## Обновление dashboard-screen.tsx

```tsx
export function DashboardScreen() {
  const { data: nowCard } = useSWR('/api/v1/me/now', fetcher);
  const { data: stats } = useSWR('/api/v1/me/stats', fetcher);
  const { data: goals } = useSWR('/api/v1/goals?status=active', fetcher);

  return (
    <div className={styles.screen}>
      {nowCard && <NowCard data={nowCard} />}
      {stats && <PersonalProgressBar stats={mapStats(stats)} />}
      <section>
        <div className={styles.sectionEyebrow}>МОИ ЦЕЛИ</div>
        {goals?.map(goal => <GoalCircleCard key={goal.id} goal={goal} />)}
        {goals?.length === 0 && (
          <EmptyState
            text="Добавь первую цель"
            action={{ label: "Создать цель", href: "/goals/new" }}
          />
        )}
      </section>
    </div>
  );
}
```

---

## Acceptance Criteria

- [ ] `GET /v1/me/now` вызывается при загрузке дашборда
- [ ] NowCard: border-left цвет соответствует urgency (danger/fire/warn/win/neutral)
- [ ] NowCard: action кнопка ведёт на правильный URL
- [ ] PersonalProgressBar: sparkChart показывает 8 недель (пустые = нулевые бары)
- [ ] TrendBadge: показывает "+N лучше" если trend=better, скрыт при first_week
- [ ] При `type=on_track` и `action=null` — NowCard без кнопки CTA
- [ ] Дашборд загружается без ошибок даже если у пользователя 0 целей
- [ ] Мобайл: NowCard и ProgressBar выше списка целей

---

## Что нельзя делать

- Не показывать других пользователей или их данные на персональном дашборде
- Не добавлять публичный рейтинг в персональный прогресс
- Не убирать существующий GoalCard компонент — только дополнять
