# Спек 6.2 — Buddy Dashboard

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 6 · Frontend — Дашборды  
**Зависимости:** Спек 3.2 (buddy API endpoints)  
**Сложность:** S  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Buddy видит где нужна его помощь — не контролирует, а поддерживает. Дашборд должен отвечать на вопрос: "Кому мне нужно ответить прямо сейчас?"

---

## Маршрут и доступ

```
/buddy           ← страница buddy dashboard
```

Доступна только пользователям у которых есть хотя бы один активный pact (роль buddy).  
Если нет активных pacts → страница с предложением стать buddy.

---

## Файловая структура

```
web/app/(product)/buddy/
  page.tsx

web/components/product/
  buddy-dashboard.tsx
  buddy-dashboard.module.css
  buddy-queue-item.tsx
  buddy-queue-item.module.css
```

---

## UI Layout

### Мобайл

```
┌─────────────────────────────┐
│ ← Buddy                     │
├─────────────────────────────┤
│ ОЧЕРЕДЬ РЕВЬЮ               │  eyebrow
│                             │
│ ┌─────────────────────────┐ │
│ │ Мария Иванова     20ч ⏳ │ │  ← queue item (ждёт 20 часов)
│ │ Kotlin Coroutines        │ │
│ │ "Показала пример Super..." │ │
│ │ [Просмотреть →]          │ │
│ └─────────────────────────┘ │
│                             │
│ ┌─────────────────────────┐ │
│ │ Алексей Соколов   2ч ⏳  │ │
│ │ System Design            │ │
│ │ "Разобрал CAP теорему..." │ │
│ │ [Просмотреть →]          │ │
│ └─────────────────────────┘ │
├─────────────────────────────┤
│ НУЖНА ПОМОЩЬ               │  ← needs_attention секция
│                             │
│ ⚠ Алексей С. — 8 дней      │  ← compact item
│   без пруфа · [Написать]   │
│                             │
├─────────────────────────────┤
│ МОЯ СТАТИСТИКА              │
│ 3 подопечных                │
│ Среднее время ответа: 6.4ч  │
│ Дано ревью: 28              │
└─────────────────────────────┘
```

### Десктоп

Те же блоки, но queue items в 2 колонки. Sidebar навигация.

---

## Компонент: buddy-dashboard.tsx

```tsx
export function BuddyDashboard() {
  const { data: queue } = useSWR('/api/v1/buddy/queue', fetcher);
  const { data: attention } = useSWR('/api/v1/buddy/needs-attention', fetcher);
  const { data: stats } = useSWR('/api/v1/buddy/stats', fetcher);

  if (!queue?.length && !attention?.length) {
    return <BuddyEmptyState />;
  }

  return (
    <div className={styles.screen}>
      {queue?.length > 0 && (
        <section>
          <div className={styles.eyebrow}>ОЧЕРЕДЬ РЕВЬЮ</div>
          <div className={styles.queue}>
            {queue.map(item => (
              <BuddyQueueItem key={item.check_in_id} item={item} />
            ))}
          </div>
        </section>
      )}

      {attention?.length > 0 && (
        <section className={styles.attentionSection}>
          <div className={styles.eyebrow}>НУЖНА ПОМОЩЬ</div>
          {attention.map(item => (
            <BuddyAttentionItem key={item.user_id} item={item} />
          ))}
        </section>
      )}

      {stats && <BuddyStats stats={stats} />}
    </div>
  );
}
```

## Компонент: BuddyQueueItem

```tsx
export function BuddyQueueItem({ item }: { item: BuddyQueueItem }) {
  const urgencyColor = item.waiting_hours > 24 ? 'var(--warn)' : 'var(--border)';

  return (
    <div className={styles.item} style={{ borderLeftColor: urgencyColor }}>
      <div className={styles.header}>
        <div className={styles.user}>
          <Avatar url={item.user.avatar_url} name={item.user.display_name} size={32} />
          <span className={styles.userName}>{item.user.display_name}</span>
        </div>
        <div className={styles.waitTime}>
          {item.waiting_hours > 0 && (
            <span className={styles.waitBadge}>
              {humanizeHours(item.waiting_hours)} ⏳
            </span>
          )}
        </div>
      </div>
      <div className={styles.goalTitle}>{item.goal_title}</div>
      {item.preview && (
        <div className={styles.preview}>"{item.preview}..."</div>
      )}
      <Link href={`/goals/${item.goal_id}/checkins/${item.check_in_id}`} className={styles.reviewBtn}>
        Просмотреть →
      </Link>
    </div>
  );
}
```

---

## CSS (buddy-dashboard.module.css)

```css
.screen {
  max-width: 720px;
  margin: 0 auto;
  padding: 0 16px;
  padding-bottom: calc(80px + env(safe-area-inset-bottom));
}

.eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: var(--ink-mono);
  margin-bottom: 12px;
  margin-top: 24px;
}

.queue {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

@media (min-width: 641px) {
  .queue {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }
}
```

```css
/* buddy-queue-item.module.css */
.item {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-left: 4px solid var(--border);
  padding: 16px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.user {
  display: flex;
  align-items: center;
  gap: 8px;
}

.userName {
  font-size: 14px;
  font-weight: 700;
  color: var(--ink-primary);
}

.waitBadge {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--warn);
}

.goalTitle {
  font-size: 13px;
  color: var(--ink-mono);
  margin-bottom: 8px;
}

.preview {
  font-size: 13px;
  color: var(--ink-primary);
  font-style: italic;
  margin-bottom: 14px;
  line-height: 1.4;
}

.reviewBtn {
  display: inline-block;
  border: 2px solid var(--border);
  color: var(--ink-primary);
  padding: 8px 14px;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  text-decoration: none;
  transition: border-color 140ms ease;
  min-height: 40px;
  line-height: 22px;
}

.reviewBtn:hover { border-color: var(--ink-primary); }
```

---

## Empty State (нет подопечных)

```tsx
function BuddyEmptyState() {
  return (
    <div className={styles.emptyState}>
      <div className={styles.emptyTitle}>У тебя пока нет подопечных</div>
      <div className={styles.emptyDesc}>
        Стань buddy для кого-то — помоги не слиться с целью
      </div>
      <Link href="/goals" className={styles.emptyBtn}>
        Посмотреть цели где нужен buddy →
      </Link>
    </div>
  );
}
```

---

## Тексты (русский)

| Элемент | Текст |
|---------|-------|
| Section queue | ОЧЕРЕДЬ РЕВЬЮ |
| Section attention | НУЖНА ПОМОЩЬ |
| Section stats | МОЯ СТАТИСТИКА |
| Wait: hours | "{N}ч ⏳" |
| Wait: days | "{N}д ⏳" |
| Review btn | Просмотреть → |
| Attention item | "{Имя} — {N} дней без пруфа" |
| Write btn | Написать |
| Stats: buddies | "{N} подопечных" |
| Stats: avg time | "Среднее время ответа: {N}ч" |
| Stats: reviews | "Дано ревью: {N}" |
| Empty title | У тебя пока нет подопечных |
| Empty desc | Стань buddy для кого-то — помоги не слиться с целью |

---

## Acceptance Criteria

- [ ] Страница доступна по `/buddy`
- [ ] Если нет активных pacts — показывается EmptyState, не пустой экран
- [ ] Queue items упорядочены по `waiting_hours` DESC (самые долгие первые)
- [ ] WaitBadge: цвет `--warn` если > 24ч, `--ink-mono` если < 24ч
- [ ] Attention section показывается только если есть items
- [ ] Stats section скрыт если нет данных (null avg_response_hours)
- [ ] Мобайл: queue в 1 колонку; десктоп: 2 колонки
- [ ] Кнопка "Написать" в attention → открывает telegram или email если подключены
- [ ] SWR revalidation каждые 60 секунд (очередь актуальна)

---

## Что нельзя делать

- Не показывать личные блокеры или описания целей подопечных
- Не добавлять кнопку "отклонить" прямо в queue item (только через детальный экран)
- Не показывать stats если нет ни одного completed review
