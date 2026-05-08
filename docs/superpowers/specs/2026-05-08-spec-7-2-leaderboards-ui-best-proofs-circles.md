# Спек 7.2 — UI: Борд лучших пруфов + Борд кругов

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 7 · Frontend — Лидерборды  
**Зависимости:** Спек 4.2, 4.3 (API), 7.1 (tab navigation)  
**Сложность:** S  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Борд лучших пруфов — витрина артефактов, не людей. Соревнуются доказательства качества, а не участники. Борд кругов — командное соревнование: сравниваются группы (круги), а не индивиды внутри них.

**DNA П4:** прогресс, а не ранжирование. Борд показывает "лучшие работы", а не "кто хуже всех".

---

## Маршрут

```
/circles/:id → вкладка "Борды" → таб "Лучшие пруфы" или "Рост"
/community-spaces/:id → таб "Круги"
```

Таб "Лучшие пруфы" и "Рост" встроены в систему из Спек 7.1.  
Таб "Круги" — отдельный борд на странице community space.

---

## Файловая структура

```
web/components/product/
  leaderboards/
    best-proofs-board.tsx
    best-proofs-board.module.css
    circles-board.tsx
    circles-board.module.css
    growth-board.tsx
    growth-board.module.css
    proof-showcase-card.tsx          ← карточка пруфа для витрины
    proof-showcase-card.module.css
```

---

## BestProofsBoard

```
ЛУЧШИЕ ПРУФЫ
Эта неделя

#1 [Аватар] Артём К. · Kotlin Coroutines
   SupervisorJob — Показал реальный сценарий...
   ✓ Одобрено buddy   ♥ 12   [Читать →]

#2 [Аватар] Мария И. · System Design
   CAP теорема — Разобрал три реальных кейса...
   ♥ 8   [Читать →]

#3 [Аватар] Дмитрий О. · AI-инструменты
   Промпт-паттерны — Собрал 5 рабочих шаблонов...
   ♥ 5   [Читать →]
```

Это витрина: показывает только топ-3 по умолчанию. Кнопка "Показать все" разворачивает до 10.

```tsx
export function BestProofsBoard({ circleId }: { circleId: string }) {
  const { data } = useSWR(`/api/v1/circles/${circleId}/best-proofs-board?limit=10`, fetcher);
  const [expanded, setExpanded] = useState(false);
  
  if (!data) return <BoardSkeleton />;
  if (data.entries.length === 0) return <EmptyBoard text="Пока нет пруфов" />;

  const entries = expanded ? data.entries : data.entries.slice(0, 3);

  return (
    <div className={styles.board}>
      <div className={styles.periodLabel}>Эта неделя</div>
      <div className={styles.entries}>
        {entries.map(entry => (
          <ProofShowcaseCard key={entry.check_in_id} entry={entry} />
        ))}
      </div>
      {data.entries.length > 3 && !expanded && (
        <button className={styles.showMore} onClick={() => setExpanded(true)}>
          Показать все {data.entries.length} пруфов
        </button>
      )}
    </div>
  );
}
```

---

## ProofShowcaseCard

```tsx
interface ProofShowcaseEntry {
  rank: number;
  check_in_id: string;
  goal_id: string;
  user_id: string;
  display_name: string;
  avatar_url?: string;
  goal_title: string;
  proof_preview: string;  // first 120 chars
  buddy_approved: boolean;
  likes_count: number;
  is_liked_by_me: boolean;
}

export function ProofShowcaseCard({ entry }: { entry: ProofShowcaseEntry }) {
  const [liked, setLiked] = useState(entry.is_liked_by_me);
  const [likes, setLikes] = useState(entry.likes_count);

  async function toggleLike() {
    const next = !liked;
    setLiked(next);
    setLikes(prev => next ? prev + 1 : prev - 1);
    await fetch(`/api/v1/check-ins/${entry.check_in_id}/like`, { method: 'POST' });
  }

  return (
    <div className={styles.card}>
      <div className={styles.rankBadge}>#{entry.rank}</div>
      <div className={styles.content}>
        <div className={styles.header}>
          <Avatar url={entry.avatar_url} name={entry.display_name} size={32} />
          <div className={styles.meta}>
            <div className={styles.authorName}>{entry.display_name}</div>
            <div className={styles.goalTag}>{entry.goal_title}</div>
          </div>
        </div>
        <div className={styles.preview}>{entry.proof_preview}…</div>
        <div className={styles.footer}>
          <div className={styles.signals}>
            {entry.buddy_approved && (
              <span className={styles.approvedBadge}>✓ Одобрено</span>
            )}
            <button className={`${styles.likeBtn} ${liked ? styles.liked : ''}`} onClick={toggleLike}>
              ♥ {likes}
            </button>
          </div>
          <Link href={`/goals/${entry.goal_id}/checkins/${entry.check_in_id}`} className={styles.readLink}>
            Читать →
          </Link>
        </div>
      </div>
    </div>
  );
}
```

```css
/* proof-showcase-card.module.css */
.card {
  display: flex;
  gap: 12px;
  padding: 16px 0;
  border-bottom: 1px solid var(--border);
}

.card:last-child { border-bottom: none; }

.rankBadge {
  font-family: var(--font-mono);
  font-size: 13px;
  font-weight: 700;
  color: var(--ink-mono);
  min-width: 28px;
  padding-top: 2px;
}

.content { flex: 1; min-width: 0; }

.header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.authorName {
  font-size: 13px;
  font-weight: 700;
  color: var(--ink-primary);
}

.goalTag {
  font-size: 11px;
  color: var(--ink-mono);
  font-family: var(--font-mono);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.preview {
  font-size: 13px;
  color: var(--ink-secondary);
  line-height: 1.5;
  margin-bottom: 10px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.signals {
  display: flex;
  align-items: center;
  gap: 10px;
}

.approvedBadge {
  font-size: 11px;
  color: var(--win);
  font-weight: 700;
}

.likeBtn {
  background: none;
  border: none;
  font-size: 12px;
  color: var(--ink-mono);
  cursor: pointer;
  padding: 4px 0;
  transition: color 140ms ease;
}

.likeBtn:hover { color: var(--ink-primary); }
.likeBtn.liked { color: var(--fire); }

.readLink {
  font-size: 12px;
  font-weight: 700;
  color: var(--ink-primary);
  text-decoration: none;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}
```

```css
/* best-proofs-board.module.css */
.board { padding-top: 4px; }

.periodLabel {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--ink-mono);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  margin-bottom: 4px;
}

.showMore {
  width: 100%;
  padding: 12px;
  background: none;
  border: 1px solid var(--border);
  color: var(--ink-mono);
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  cursor: pointer;
  margin-top: 12px;
  transition: border-color 140ms ease, color 140ms ease;
}

.showMore:hover {
  border-color: var(--ink-primary);
  color: var(--ink-primary);
}
```

---

## GrowthBoard

```
РОСТ
Последние 4 недели

#1 [Аватар] Артём К.   +4 пруфа   ↑ Активный рост
#2 [Аватар] Мария И.   +2 пруфа   ↑ Растёт
#3 [Аватар] Дмитрий О. +1 пруф    → Держит ритм
```

Показывает только позитивный прирост (delta > 0). Строки с delta ≤ 0 скрыты.

```tsx
export function GrowthBoard({ circleId }: { circleId: string }) {
  const { data } = useSWR(`/api/v1/circles/${circleId}/growth-board`, fetcher);
  
  if (!data) return <BoardSkeleton />;
  if (data.entries.length === 0) {
    return <EmptyBoard text="Пока недостаточно данных для роста" />;
  }

  return (
    <div className={styles.board}>
      <div className={styles.periodLabel}>Последние 4 недели</div>
      {data.entries.map(entry => (
        <div key={entry.user_id} className={`${styles.entry} ${entry.is_current_user ? styles.highlighted : ''}`}>
          <div className={styles.rank}>#{entry.rank}</div>
          <Avatar url={entry.avatar_url} name={entry.display_name} size={36} />
          <div className={styles.info}>
            <div className={styles.name}>{entry.display_name}</div>
            <div className={styles.growthLabel}>{entry.growth_label}</div>
          </div>
          <div className={styles.delta}>+{entry.proof_delta}</div>
        </div>
      ))}
    </div>
  );
}
```

```css
/* growth-board.module.css */
.board { padding-top: 4px; }

.periodLabel {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--ink-mono);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  margin-bottom: 8px;
}

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
}

.growthLabel {
  font-size: 11px;
  color: var(--win);
  margin-top: 2px;
}

.delta {
  font-family: var(--font-display);
  font-size: 16px;
  font-weight: 700;
  color: var(--win);
  text-align: right;
  min-width: 48px;
}
```

---

## CirclesBoard (community-spaces)

Отдельный компонент для страницы community space. Сравнивает круги, не людей.

```
КРУГИ СООБЩЕСТВА

#1 Kotlin Backend   ████████ 87%   24 участника  → ЖИВЁТ
#2 System Design    ██████   72%   18 участника  → АКТИВНЫЙ
#3 AI-инструменты   ████     45%   31 участник   ⚠ ПОД УГРОЗОЙ
```

```tsx
export function CirclesBoard({ communityId }: { communityId: string }) {
  const { data } = useSWR(`/api/v1/community-spaces/${communityId}/circles-board`, fetcher);
  
  if (!data) return <BoardSkeleton />;
  if (data.circles.length === 0) return <EmptyBoard text="Пока нет кругов" />;

  return (
    <div className={styles.board}>
      {data.circles.map(circle => (
        <CircleRow key={circle.circle_id} circle={circle} />
      ))}
    </div>
  );
}

function CircleRow({ circle }) {
  const statusColor = circle.status === 'thriving' 
    ? 'var(--win)' 
    : circle.status === 'active' 
    ? 'var(--ink-primary)' 
    : 'var(--warn)';
  
  const statusLabel = {
    thriving: 'ЖИВЁТ',
    active: 'АКТИВНЫЙ',
    at_risk: 'ПОД УГРОЗОЙ',
  }[circle.status];

  return (
    <Link href={`/circles/${circle.circle_id}`} className={styles.circleRow}>
      <div className={styles.circleRank}>#{circle.rank}</div>
      <div className={styles.circleInfo}>
        <div className={styles.circleName}>{circle.name}</div>
        <div className={styles.barWrap}>
          <div className={styles.bar} style={{ width: `${circle.completion_pct}%` }} />
        </div>
      </div>
      <div className={styles.circleMeta}>
        <div className={styles.circleMembers}>{circle.member_count} уч.</div>
        <div className={styles.circleStatus} style={{ color: statusColor }}>
          {circle.status === 'at_risk' ? '⚠ ' : '→ '}{statusLabel}
        </div>
      </div>
      <div className={styles.circleScore}>{circle.completion_pct}%</div>
    </Link>
  );
}
```

```css
/* circles-board.module.css */
.board { padding-top: 4px; }

.circleRow {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 0;
  border-bottom: 1px solid var(--border);
  text-decoration: none;
  transition: background 120ms ease;
}

.circleRow:last-child { border-bottom: none; }

.circleRow:hover { background: var(--bg-elevated); margin: 0 -16px; padding: 14px 16px; }

.circleRank {
  font-family: var(--font-mono);
  font-size: 13px;
  font-weight: 700;
  color: var(--ink-mono);
  min-width: 28px;
}

.circleInfo { flex: 1; min-width: 0; }

.circleName {
  font-size: 14px;
  font-weight: 700;
  color: var(--ink-primary);
  margin-bottom: 6px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.barWrap {
  height: 3px;
  background: var(--border);
}

.bar {
  height: 100%;
  background: var(--win);
}

.circleMeta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  min-width: 88px;
}

.circleMembers {
  font-size: 11px;
  color: var(--ink-mono);
}

.circleStatus {
  font-size: 10px;
  font-weight: 700;
  font-family: var(--font-mono);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.circleScore {
  font-family: var(--font-display);
  font-size: 16px;
  font-weight: 700;
  color: var(--ink-primary);
  min-width: 40px;
  text-align: right;
}
```

---

## Интеграция табов (обновление circle-standings.tsx)

Два новых таба добавляются к системе из Спек 7.1:

```tsx
// В circle-standings.tsx рендер контента таба:
{activeTab === 'best_proofs' && <BestProofsBoard circleId={circleId} />}
{activeTab === 'growth' && <GrowthBoard circleId={circleId} />}
```

CirclesBoard рендерится на странице community space отдельно — не через таб-систему кругов.

---

## Acceptance Criteria

- [ ] BestProofsBoard: по умолчанию показывает топ-3, "Показать все" разворачивает до 10
- [ ] ProofShowcaseCard: кнопка лайка оптимистично обновляет счётчик без ожидания API
- [ ] Нет дублирования лайков: повторный клик убирает лайк (toggle)
- [ ] `buddy_approved` badge показывается только если `buddy_approved === true`
- [ ] GrowthBoard: показывает только участников с delta > 0
- [ ] GrowthBoard: если все delta ≤ 0 — EmptyBoard "Пока недостаточно данных"
- [ ] CirclesBoard: статус `at_risk` окрашивается в --warn, `thriving` в --win
- [ ] CirclesBoard: строки кликабельны и ведут на `/circles/:id`
- [ ] Skeleton лоадеры во время загрузки
- [ ] Пустые борды — текст "Пока нет данных" (не пустой экран)

---

## Что нельзя делать

- Не показывать в growth board участников с отрицательным или нулевым приростом
- Не добавлять кнопку "антипруф" или "нужна помощь" в витрину — это борд достижений, не контроль
- Не позволять ставить лайк своему же пруфу (проверка на backend, UI не показывает кнопку)
