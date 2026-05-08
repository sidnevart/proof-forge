# Спек 7.3 — UI: Борд достижений + Панель "Нужна помощь"

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 7 · Frontend — Лидерборды  
**Зависимости:** Спек 4.4 (API), 7.1 (tab navigation)  
**Сложность:** S  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Борд достижений — публичный праздник milestone'ов. Панель "нужна помощь" — приватный инструмент тимлида: видит только он, только для поддержки, не для давления.

**DNA П5:** риски агрегированно и приватно. **DNA П6:** успехи публично.

---

## Маршрут

```
/circles/:id → вкладка "Борды" → таб "Достижения"
/teamspaces/:id/needs-help       ← отдельная страница, только для team_lead
```

---

## Файловая структура

```
web/app/(product)/teamspaces/[id]/needs-help/
  page.tsx

web/components/product/
  leaderboards/
    achievements-board.tsx
    achievements-board.module.css
    achievement-badge.tsx
    achievement-badge.module.css

  needs-help/
    needs-help-panel.tsx
    needs-help-panel.module.css
    needs-help-member-row.tsx
```

---

## AchievementsBoard

```
ДОСТИЖЕНИЯ

[Значок] Артём К.     🏆 Первый пруф           сегодня
[Значок] Мария И.     🔥 7 недель подряд        вчера
[Значок] Дмитрий О.   💎 10 пруфов             3 дня назад

  [Мои достижения →]
```

Борд показывает последние 10 разблокированных достижений в кругу. Это поток событий — кто что получил, в хронологическом порядке. Акцент на праздник момента.

```tsx
export function AchievementsBoard({ circleId }: { circleId: string }) {
  const { data } = useSWR(`/api/v1/circles/${circleId}/achievements-feed?limit=10`, fetcher);
  
  if (!data) return <BoardSkeleton />;
  if (data.feed.length === 0) {
    return (
      <EmptyBoard text="Первое достижение скоро появится здесь" />
    );
  }

  return (
    <div className={styles.board}>
      <div className={styles.feed}>
        {data.feed.map(item => (
          <AchievementFeedItem key={`${item.user_id}-${item.achievement_id}-${item.unlocked_at}`} item={item} />
        ))}
      </div>
      <Link href="/me/achievements" className={styles.myLink}>
        Мои достижения →
      </Link>
    </div>
  );
}

function AchievementFeedItem({ item }) {
  return (
    <div className={styles.item}>
      <AchievementBadge icon={item.icon} size={40} />
      <div className={styles.itemInfo}>
        <div className={styles.itemUser}>{item.display_name}</div>
        <div className={styles.itemName}>{item.name}</div>
      </div>
      <div className={styles.itemTime}>{formatRelativeTime(item.unlocked_at)}</div>
    </div>
  );
}
```

```css
/* achievements-board.module.css */
.board { padding-top: 4px; }

.feed { display: flex; flex-direction: column; }

.item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid var(--border);
}

.item:last-child { border-bottom: none; }

.itemInfo { flex: 1; min-width: 0; }

.itemUser {
  font-size: 13px;
  font-weight: 700;
  color: var(--ink-primary);
}

.itemName {
  font-size: 12px;
  color: var(--ink-mono);
  margin-top: 2px;
}

.itemTime {
  font-size: 11px;
  color: var(--ink-mono);
  font-family: var(--font-mono);
  white-space: nowrap;
}

.myLink {
  display: block;
  margin-top: 16px;
  font-size: 12px;
  font-weight: 700;
  color: var(--ink-mono);
  text-decoration: none;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  text-align: right;
  transition: color 140ms ease;
}

.myLink:hover { color: var(--ink-primary); }
```

---

## AchievementBadge (переиспользуемый)

```tsx
interface AchievementBadgeProps {
  icon: string;   // emoji: "🏆", "🔥", "💎", etc.
  size?: number;
  locked?: boolean;
}

export function AchievementBadge({ icon, size = 40, locked = false }: AchievementBadgeProps) {
  return (
    <div 
      className={`${styles.badge} ${locked ? styles.locked : ''}`}
      style={{ width: size, height: size, fontSize: size * 0.5 }}
    >
      {locked ? '🔒' : icon}
    </div>
  );
}
```

```css
/* achievement-badge.module.css */
.badge {
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  flex-shrink: 0;
}

.badge.locked {
  opacity: 0.4;
  filter: grayscale(1);
}
```

---

## Личные достижения — /me/achievements

Отдельная страница (не борд кругов). Показывает все достижения пользователя: разблокированные + заблокированные с прогрессом.

```tsx
export function MyAchievementsPage() {
  const { data } = useSWR('/api/v1/me/achievements', fetcher);
  
  if (!data) return <PageSkeleton />;

  return (
    <div className={styles.page}>
      <div className={styles.eyebrow}>МОИ ДОСТИЖЕНИЯ</div>
      <div className={styles.grid}>
        {data.achievements.map(a => (
          <AchievementCard key={a.id} achievement={a} />
        ))}
      </div>
    </div>
  );
}

function AchievementCard({ achievement }) {
  return (
    <div className={`${styles.card} ${achievement.unlocked ? styles.unlocked : ''}`}>
      <AchievementBadge icon={achievement.icon} size={48} locked={!achievement.unlocked} />
      <div className={styles.cardName}>{achievement.name}</div>
      <div className={styles.cardDesc}>{achievement.description}</div>
      {!achievement.unlocked && achievement.progress !== undefined && (
        <div className={styles.progress}>
          <div className={styles.progressBar}>
            <div 
              className={styles.progressFill}
              style={{ width: `${(achievement.progress / achievement.target) * 100}%` }}
            />
          </div>
          <div className={styles.progressLabel}>
            {achievement.progress} / {achievement.target}
          </div>
        </div>
      )}
      {achievement.unlocked && achievement.unlocked_at && (
        <div className={styles.unlockedAt}>
          Получено {formatDate(achievement.unlocked_at)}
        </div>
      )}
    </div>
  );
}
```

```css
/* В achievements-board.module.css — страница достижений */
.page {
  max-width: 640px;
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
  margin-bottom: 20px;
  margin-top: 24px;
}

.grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

@media (min-width: 641px) {
  .grid { grid-template-columns: 1fr 1fr 1fr; }
}

.card {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  text-align: center;
  opacity: 0.5;
}

.card.unlocked { opacity: 1; }

.cardName {
  font-size: 13px;
  font-weight: 700;
  color: var(--ink-primary);
}

.cardDesc {
  font-size: 11px;
  color: var(--ink-mono);
  line-height: 1.4;
}

.progress { width: 100%; }

.progressBar {
  height: 2px;
  background: var(--border);
  margin-bottom: 4px;
}

.progressFill {
  height: 100%;
  background: var(--win);
}

.progressLabel {
  font-size: 10px;
  color: var(--ink-mono);
  font-family: var(--font-mono);
}

.unlockedAt {
  font-size: 10px;
  color: var(--win);
  font-family: var(--font-mono);
}
```

---

## NeedsHelpPanel (только для teamspace_lead)

```
НУЖНА ПОМОЩЬ
/teamspaces/:id/needs-help

2 участника ждут внимания

──────────────────────────────────────
Алексей С.   8 дней без пруфа    [Написать]
  Цель: System Design
  
Мария И.     Ждёт ревью 3 дня    [Перейти к ревью →]
  Kotlin Coroutines
──────────────────────────────────────

ЗАТУХАЮЩИЕ КРУГИ
⚠ Backend Club — 14 дней без активности
  [Посмотреть круг →]
```

```tsx
export function NeedsHelpPanel({ teamspaceId }: { teamspaceId: string }) {
  const { data } = useSWR(`/api/v1/teamspaces/${teamspaceId}/needs-help`, fetcher);
  
  if (!data) return <PageSkeleton />;

  const hasContent = data.members.length > 0 || data.at_risk_circles.length > 0;

  if (!hasContent) {
    return (
      <div className={styles.allGood}>
        <div className={styles.allGoodIcon}>✓</div>
        <div className={styles.allGoodText}>Все двигаются</div>
        <div className={styles.allGoodSub}>Нет участников требующих внимания</div>
      </div>
    );
  }

  return (
    <div className={styles.panel}>
      {data.members.length > 0 && (
        <section>
          <div className={styles.sectionTitle}>
            {data.members.length} участника ждут внимания
          </div>
          {data.members.map(m => (
            <NeedsHelpMemberRow key={m.user_id} member={m} />
          ))}
        </section>
      )}

      {data.at_risk_circles.length > 0 && (
        <section className={styles.circlesSection}>
          <div className={styles.sectionTitle}>ЗАТУХАЮЩИЕ КРУГИ</div>
          {data.at_risk_circles.map(c => (
            <div key={c.circle_id} className={styles.atRiskCircle}>
              <span className={styles.warnIcon}>⚠</span>
              <div className={styles.circleInfo}>
                <div className={styles.circleName}>{c.name}</div>
                <div className={styles.circleSub}>
                  {c.days_since_last_proof} дней без активности
                </div>
              </div>
              <Link href={`/circles/${c.circle_id}`} className={styles.circleLink}>
                Посмотреть →
              </Link>
            </div>
          ))}
        </section>
      )}
    </div>
  );
}
```

```tsx
// needs-help-member-row.tsx
export function NeedsHelpMemberRow({ member }) {
  const isStuck = member.attention_reason === 'no_proof_over_5_days';
  const isWaiting = member.attention_reason === 'awaiting_buddy_review';

  return (
    <div className={styles.row}>
      <Avatar url={member.avatar_url} name={member.display_name} size={36} />
      <div className={styles.info}>
        <div className={styles.name}>{member.display_name}</div>
        <div className={styles.goalTitle}>{member.goal_title}</div>
        <div className={styles.reason}>
          {isStuck && `${member.days_without_proof} дней без пруфа`}
          {isWaiting && `Ждёт ревью ${member.days_waiting} дня`}
        </div>
      </div>
      <div className={styles.action}>
        {isStuck && member.contact_url && (
          <a href={member.contact_url} className={styles.contactBtn} target="_blank" rel="noreferrer">
            Написать
          </a>
        )}
        {isWaiting && (
          <Link href={`/goals/${member.goal_id}/checkins/${member.check_in_id}`} className={styles.reviewBtn}>
            К ревью →
          </Link>
        )}
      </div>
    </div>
  );
}
```

```css
/* needs-help-panel.module.css */
.panel {
  max-width: 640px;
  margin: 0 auto;
  padding: 0 16px;
  padding-bottom: calc(80px + env(safe-area-inset-bottom));
}

.sectionTitle {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: var(--ink-mono);
  margin: 24px 0 12px;
}

.allGood {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60px 16px;
  gap: 8px;
  text-align: center;
}

.allGoodIcon {
  font-size: 32px;
  color: var(--win);
  margin-bottom: 8px;
}

.allGoodText {
  font-family: var(--font-display);
  font-size: 20px;
  font-weight: 700;
  color: var(--ink-primary);
}

.allGoodSub {
  font-size: 14px;
  color: var(--ink-mono);
}

.circlesSection { margin-top: 8px; }

.atRiskCircle {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 0;
  border-bottom: 1px solid var(--border);
}

.atRiskCircle:last-child { border-bottom: none; }

.warnIcon {
  font-size: 16px;
  color: var(--warn);
  flex-shrink: 0;
}

.circleInfo { flex: 1; }

.circleName {
  font-size: 14px;
  font-weight: 700;
  color: var(--ink-primary);
}

.circleSub {
  font-size: 12px;
  color: var(--warn);
  margin-top: 2px;
}

.circleLink {
  font-size: 12px;
  font-weight: 700;
  color: var(--ink-mono);
  text-decoration: none;
  white-space: nowrap;
}
```

```css
/* needs-help-member-row.module.css */
.row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 0;
  border-bottom: 1px solid var(--border);
}

.row:last-child { border-bottom: none; }

.info { flex: 1; min-width: 0; }

.name {
  font-size: 14px;
  font-weight: 700;
  color: var(--ink-primary);
}

.goalTitle {
  font-size: 12px;
  color: var(--ink-mono);
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.reason {
  font-size: 12px;
  color: var(--warn);
  margin-top: 4px;
}

.contactBtn,
.reviewBtn {
  display: inline-block;
  border: 2px solid var(--border);
  color: var(--ink-primary);
  padding: 8px 14px;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  text-decoration: none;
  white-space: nowrap;
  min-height: 40px;
  line-height: 22px;
  transition: border-color 140ms ease;
}

.contactBtn:hover,
.reviewBtn:hover {
  border-color: var(--ink-primary);
}
```

---

## Роутинг /teamspaces/[id]/needs-help

```tsx
// web/app/(product)/teamspaces/[id]/needs-help/page.tsx
import { requireTeamspaceLead } from '@/lib/auth-guards';

export default async function NeedsHelpPage({ params }) {
  await requireTeamspaceLead(params.id);
  return <NeedsHelpPanel teamspaceId={params.id} />;
}
```

На frontend: если API вернул 403 (не тимлид) — redirect на `/teamspaces/:id`.

---

## Acceptance Criteria

- [ ] AchievementsBoard: показывает последние 10 достижений в кругу в хронологическом порядке
- [ ] AchievementBadge: locked-состояние — серый + замок-иконка
- [ ] MyAchievementsPage: 2-col grid мобайл, 3-col десктоп
- [ ] Прогресс-бар для заблокированных достижений отображает текущий прогресс
- [ ] NeedsHelpPanel: "Все двигаются" empty state если нет ни members, ни circles
- [ ] NeedsHelpMemberRow: показывает "Написать" только если есть contact_url
- [ ] Страница /needs-help: 403 → redirect на /teamspaces/:id (не ошибка на экране)
- [ ] Затухающие круги — ссылки работают
- [ ] SWR revalidation через 60 секунд на needs-help (данные оперативные)

---

## Что нельзя делать

- Не показывать имена и цели участников в "нужна помощь" никому кроме тимлида
- Не добавлять в achievements-board кнопки действий — только просмотр
- Не показывать locked достижения без прогресс-бара (если прогресс есть в API)
- Не добавлять в panel возможность написать комментарий или оценку участнику
