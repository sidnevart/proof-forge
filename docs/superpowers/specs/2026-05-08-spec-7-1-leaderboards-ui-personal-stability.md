# Спек 7.1 — UI: Личный лидерборд + Борд стабильности

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 7 · Frontend — Лидерборды  
**Зависимости:** Спек 4.1 (API), 6.1 (personal dashboard)  
**Сложность:** S  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Лидерборды живут на странице круга. Доступны через таб-навигацию. Личный борд — только своя история. Борд стабильности — топ-5 без дна.

---

## Маршрут

```
/circles/:id → вкладка "Борды"
```

В существующем `circle-standings.tsx` добавить таб-навигацию по типам борда.

---

## Файловая структура

```
web/components/product/
  circle-standings.tsx           ← обновить: добавить табы
  leaderboards/
    personal-board.tsx
    personal-board.module.css
    stability-board.tsx
    stability-board.module.css
    leaderboard-entry.tsx         ← переиспользуемый элемент списка
    leaderboard-entry.module.css
```

---

## UI: Табы бордов

```
┌─────────────────────────────┐
│ БОРДЫ                       │
│                             │
│ [Мой] [Стабильность] [Вклад]│  ← tabs
│ [Пруфы] [Рост] [Достиж.]    │
├─────────────────────────────┤
│ [контент активного таба]    │
└─────────────────────────────┘
```

```tsx
const BOARD_TABS = [
  { id: 'personal', label: 'Мой прогресс' },
  { id: 'stability', label: 'Стабильность' },
  { id: 'contribution', label: 'Вклад' },
  { id: 'best_proofs', label: 'Лучшие пруфы' },
  { id: 'growth', label: 'Рост' },
  { id: 'achievements', label: 'Достижения' },
];
```

Горизонтальный скролл табов на мобайле (overflow-x: auto, no scrollbar visible).

---

## PersonalBoard

```
МОЙ ПРОГРЕСС

Эта неделя                    +1 лучше чем неделю назад
2 пруфа

[Спаркчарт 8 недель]

Streak: 4 недели подряд  
Рекорд: 7 недель

ИСТОРИЯ
Вт 5 пруфов     ▼ лучший результат
Вт 3 пруфа
Вт 2 пруфа
Вт 2 пруфа  ← сейчас
```

Это дублирует PersonalProgressBar из дашборда, но в расширенном виде с историей.

```tsx
export function PersonalBoard() {
  const { data } = useSWR('/api/v1/me/leaderboard', fetcher);
  if (!data) return <BoardSkeleton />;

  return (
    <div className={styles.board}>
      <WeeklySummary
        thisWeek={data.current_week}
        streak={data.streak}
      />
      <SparkChart weeks={data.weekly_history} highlightBest />
      <WeeklyHistoryList history={data.weekly_history} personalRecord={data.streak.personal_record_weeks} />
    </div>
  );
}
```

---

## StabilityBoard

```
СТАБИЛЬНОСТЬ
Последние 4 недели

#1  [Ав] Артём К.   ████████ 95%   8 нед streak
#2  [Ав] Мария И.   ███████  88%   6 нед streak  ← ты
#3  [Ав] Дмитрий О. ██████   75%   5 нед streak
#4  [Ав] Лена С.    █████    62%   4 нед streak
#5  [Ав] Антон Р.   ████     50%   3 нед streak

  Твоя позиция: #2 из 7 участников
```

Если текущий пользователь в топ-5 — подсвечен. Если нет — строка с его позицией внизу.

```tsx
export function StabilityBoard({ circleId }) {
  const { data } = useSWR(`/api/v1/circles/${circleId}/stability-board?limit=5`, fetcher);

  return (
    <div className={styles.board}>
      <div className={styles.periodLabel}>Последние 4 недели</div>
      <div className={styles.entries}>
        {data.entries.map(entry => (
          <LeaderboardEntry
            key={entry.user_id}
            rank={entry.rank}
            user={{ id: entry.user_id, name: entry.display_name, avatarUrl: entry.avatar_url }}
            metric={`${entry.consistency_score}%`}
            submetric={`${entry.proof_streak_weeks} нед. подряд`}
            barValue={entry.consistency_score}
            isCurrentUser={entry.is_current_user}
          />
        ))}
      </div>
      {!data.entries.find(e => e.is_current_user) && data.current_user_position && (
        <div className={styles.myPosition}>
          Твоя позиция: #{data.current_user_position.rank} из {data.current_user_position.total_participants} участников
        </div>
      )}
    </div>
  );
}
```

---

## LeaderboardEntry (переиспользуемый)

```tsx
interface LeaderboardEntryProps {
  rank: number;
  user: { id: string; name: string; avatarUrl?: string };
  metric: string;       // главная метрика: "95%" или "14 ревью"
  submetric?: string;   // вторичная: "8 нед. подряд"
  barValue?: number;    // 0-100 для горизонтального бара
  isCurrentUser: boolean;
}

export function LeaderboardEntry({ rank, user, metric, submetric, barValue, isCurrentUser }) {
  return (
    <div className={`${styles.entry} ${isCurrentUser ? styles.highlighted : ''}`}>
      <div className={styles.rank}>#{rank}</div>
      <Avatar url={user.avatarUrl} name={user.name} size={36} />
      <div className={styles.info}>
        <div className={styles.name}>{user.name}</div>
        {submetric && <div className={styles.sub}>{submetric}</div>}
        {barValue !== undefined && (
          <div className={styles.barWrap}>
            <div className={styles.bar} style={{ width: `${barValue}%` }} />
          </div>
        )}
      </div>
      <div className={styles.metric}>{metric}</div>
    </div>
  );
}
```

```css
/* leaderboard-entry.module.css */
.entry {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid var(--border);
}

.entry:last-child { border-bottom: none; }

.entry.highlighted {
  background: rgba(28, 231, 131, 0.04);
  margin: 0 -16px;
  padding: 12px 16px;
  border-left: 3px solid var(--win);
}

.rank {
  font-family: var(--font-mono);
  font-size: 13px;
  font-weight: 700;
  color: var(--ink-mono);
  min-width: 28px;
}

.entry.highlighted .rank { color: var(--win); }

.info { flex: 1; min-width: 0; }

.name {
  font-size: 14px;
  font-weight: 700;
  color: var(--ink-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sub {
  font-size: 11px;
  color: var(--ink-mono);
  margin-top: 2px;
}

.barWrap {
  height: 3px;
  background: var(--border);
  margin-top: 6px;
}

.bar {
  height: 100%;
  background: var(--win);
}

.metric {
  font-family: var(--font-display);
  font-size: 16px;
  font-weight: 700;
  color: var(--ink-primary);
  text-align: right;
  min-width: 48px;
}
```

---

## Acceptance Criteria

- [ ] Табы горизонтально скроллятся на мобайле без видимого scrollbar
- [ ] PersonalBoard: загружает данные только текущего пользователя
- [ ] SparkChart показывает бары для всех 8 недель (нулевые = min height 2px)
- [ ] StabilityBoard: is_current_user подсвечивается зелёной левой границей
- [ ] StabilityBoard: "Твоя позиция" строка показывается только если не в топ-5
- [ ] Skeleton загрузчики во время fetch
- [ ] Пустой борд (0 участников) — текст "Пока нет данных"

---

## Что нельзя делать

- Не показывать в stability board участников ниже limit
- Не добавлять в personal board чужие данные
