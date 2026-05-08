# Спек 6.4 — Platform Admin Panel

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 6 · Frontend — Дашборды  
**Зависимости:** Спек 3.5 (admin API), 2.1 (platform_admin role)  
**Сложность:** M  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Platform admin управляет всей платформой: видит workspace, здоровье пространств, назначает роли, замораживает/восстанавливает. Это **внутренний инструмент**, не клиентский UI.

**Принцип DNA П10:** опасные действия (freeze, make-admin) требуют явного подтверждения.

---

## Маршрут и доступ

```
/admin           ← только для is_platform_admin = true
/admin/workspaces
/admin/users
```

Если `is_platform_admin = false` → redirect на `/dashboard`.

---

## Файловая структура

```
web/app/(admin)/
  layout.tsx                ← admin layout с sidebar
  page.tsx                  ← redirect to /admin/workspaces
  workspaces/page.tsx
  users/page.tsx

web/components/admin/
  admin-layout.tsx
  admin-layout.module.css
  workspace-table.tsx
  workspace-table.module.css
  user-table.tsx
  health-score-badge.tsx
  freeze-confirm-modal.tsx
```

---

## Admin Layout

### Десктоп (основной режим для admin)

```
┌──────────┬───────────────────────────────────────┐
│ ADMIN    │ [заголовок страницы]                  │
│──────────│                                       │
│Workspace │ [основной контент]                    │
│Пользоват │                                       │
│          │                                       │
│──────────│                                       │
│ Выйти    │                                       │
└──────────┴───────────────────────────────────────┘
```

### Мобайл

Sidebar collapsed — только иконки. Контент на всю ширину. Admin panel не оптимизирован для мобайла, но функционален.

```tsx
export function AdminLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className={styles.layout}>
      <aside className={styles.sidebar}>
        <div className={styles.logo}>ADMIN</div>
        <nav>
          <AdminNavLink href="/admin/workspaces" label="Workspace" />
          <AdminNavLink href="/admin/users" label="Пользователи" />
        </nav>
        <div className={styles.sidebarFooter}>
          <Link href="/dashboard" className={styles.exitLink}>← В продукт</Link>
        </div>
      </aside>
      <main className={styles.main}>{children}</main>
    </div>
  );
}
```

```css
/* admin-layout.module.css */
.layout {
  display: grid;
  grid-template-columns: 220px 1fr;
  min-height: 100vh;
  background: var(--bg-base);
}

@media (max-width: 640px) {
  .layout { grid-template-columns: 60px 1fr; }
}

.sidebar {
  border-right: 1px solid var(--border);
  padding: 24px 16px;
  display: flex;
  flex-direction: column;
}

.logo {
  font-family: var(--font-display);
  font-size: 18px;
  font-weight: 700;
  color: var(--danger);
  margin-bottom: 32px;
  letter-spacing: 0.1em;
}

.main {
  padding: clamp(20px, 3vw, 40px);
  max-width: 1100px;
}
```

---

## /admin/workspaces — Список workspace

### UI

```
WORKSPACE                                    [+ Создать workspace]

[Поиск workspace...]

┌──────────────────────────────────────────────────────────┐
│ Название         │ Тип  │ Health │ Участ │ Риск  │ Действ │
├──────────────────────────────────────────────────────────┤
│ T-Bank AI Stream │ org  │ ████82 │ 80    │ LOW   │ [···]  │
│ ML Community     │ comm │ ████41 │ 45    │ HIGH  │ [···]  │
│ Backend Club     │ comm │ ████67 │ 23    │ MED   │ [···]  │
└──────────────────────────────────────────────────────────┘
```

Health score — цветная полоска: ≥70 зелёная, 40-69 жёлтая, <40 красная.

Действия (···) — dropdown:
- Открыть workspace
- Редактировать
- **Заморозить** (красный, с подтверждением)
- **Разморозить** (если frozen)

### Freeze Confirm Modal

```
┌──────────────────────────────┐
│ Заморозить workspace?        │
│                              │
│ «T-Bank AI Stream» будет     │
│ заморожен. Пользователи не   │
│ смогут создавать новые цели  │
│ и пруфы. Данные сохранятся.  │
│                              │
│ Причина заморозки:           │
│ [________________________]   │
│                              │
│ [Отмена]    [Заморозить ▼]   │
└──────────────────────────────┘
```

Кнопка "Заморозить" — `background: var(--danger)`.

```tsx
function FreezeConfirmModal({ workspace, onConfirm, onClose }) {
  const [reason, setReason] = useState('');
  return (
    <Modal onClose={onClose}>
      <div className={styles.title}>Заморозить workspace?</div>
      <div className={styles.desc}>
        «{workspace.name}» будет заморожен. Пользователи не смогут создавать 
        новые цели и пруфы. Данные сохранятся.
      </div>
      <input
        className={styles.input}
        placeholder="Причина заморозки"
        value={reason}
        onChange={e => setReason(e.target.value)}
      />
      <div className={styles.actions}>
        <button className={styles.cancelBtn} onClick={onClose}>Отмена</button>
        <button 
          className={styles.dangerBtn} 
          onClick={() => onConfirm(reason)}
          disabled={!reason.trim()}
        >
          Заморозить
        </button>
      </div>
    </Modal>
  );
}
```

---

## /admin/users — Список пользователей

```
ПОЛЬЗОВАТЕЛИ

[🔍 Поиск по имени или email...]

┌──────────────────────────────────────────────────────────┐
│ Пользователь     │ Email          │ Admin │ Активность  │
├──────────────────────────────────────────────────────────┤
│ [Ав] Иван Петров │ ivan@...       │ ✓     │ 7 мая       │
│ [Ав] Мария Ив.   │ maria@...      │       │ 6 мая       │
└──────────────────────────────────────────────────────────┘
                                              Стр 1/62 → 
```

Клик на строку пользователя → боковая панель с детальной информацией:
- Профиль
- Кнопка "Назначить admin" / "Снять права admin" (с подтверждением)
- Список workspace и ролей

---

## HealthScoreBadge

```tsx
export function HealthScoreBadge({ score }: { score: number }) {
  const color = score >= 70 ? 'var(--win)' : score >= 40 ? 'var(--warn)' : 'var(--danger)';
  return (
    <div className={styles.badge}>
      <div className={styles.bar}>
        <div 
          className={styles.fill}
          style={{ width: `${score}%`, background: color }}
        />
      </div>
      <span className={styles.value} style={{ color }}>{score}</span>
    </div>
  );
}
```

---

## Тексты (русский)

| Элемент | Текст |
|---------|-------|
| Nav workspaces | Workspace |
| Nav users | Пользователи |
| Exit link | ← В продукт |
| Create btn | + Создать workspace |
| Search placeholder | Поиск workspace... |
| Risk low | НИЗКИЙ |
| Risk medium | СРЕДНИЙ |
| Risk high | ВЫСОКИЙ |
| Freeze btn | Заморозить |
| Unfreeze btn | Разморозить |
| Modal title | Заморозить workspace? |
| Modal reason | Причина заморозки |
| Cancel | Отмена |
| Make admin | Назначить администратора |
| Remove admin | Снять права администратора |

---

## Acceptance Criteria

- [ ] `/admin/*` redirect на `/dashboard` для не-platform_admin
- [ ] Workspace таблица загружает данные из `GET /v1/admin/workspaces`
- [ ] HealthScoreBadge: цвет корректен (зелёный ≥70, жёлтый 40–69, красный <40)
- [ ] Freeze action: открывает modal, кнопка confirm неактивна пока не заполнена причина
- [ ] После freeze → строка workspace обновляется (is_active = false, статус "Заморожен")
- [ ] Поиск по workspace — client-side фильтрация по name
- [ ] Users: поиск дёргает `GET /v1/admin/users?q=...` с дебаунсом 400ms
- [ ] Make-admin: требует подтверждения в modal
- [ ] Keyboard: Esc закрывает modal

---

## Что нельзя делать

- Не показывать email пользователей в общем списке (только в детальном просмотре)
- Не давать admin удалять workspace или пользователей (только freeze)
- Не убирать confirmation modal для destructive actions
