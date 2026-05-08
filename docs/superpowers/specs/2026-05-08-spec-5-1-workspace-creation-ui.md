# Спек 5.1 — UI создания Workspace

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 5 · Frontend — Пространства  
**Зависимости:** Спек 1.1 (API workspaces), 2.1 (роли)  
**Сложность:** S  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Workspace owner или platform_admin создаёт новое пространство для организации или сообщества. Задача — сделать так, чтобы форма была быстрой и без лишних полей. Только необходимое на старте, остальное — позже в настройках.

**Принцип DNA П9:** скорость над полнотой. Создание workspace — за 3 клика.

---

## Страницы и маршруты

```
/workspaces/new   ← страница создания
/workspaces       ← список моих workspace (обновить существующий или создать)
```

---

## Файловая структура

```
web/app/(product)/workspaces/
  new/
    page.tsx                    ← страница создания
  page.tsx                      ← список workspace

web/components/product/
  workspace-setup-screen.tsx    ← основной компонент
  workspace-setup-screen.module.css
  workspace-card.tsx            ← карточка в списке
  workspace-card.module.css
```

---

## UI: страница /workspaces/new

### Мобайл (≤640px)

```
┌─────────────────────────────┐
│ ← Назад                     │  ← header, 48px
├─────────────────────────────┤
│                             │
│  СОЗДАТЬ ПРОСТРАНСТВО       │  ← eyebrow 11px mono uppercase
│                             │
│  Для кого это пространство? │  ← section title 24px Impact
│                             │
│  ┌──────────────────────┐   │
│  │ 🏢 Для команды       │   │  ← карточка type=org
│  │ Компания, департамент│   │
│  │ или направление      │   │
│  └──────────────────────┘   │
│                             │
│  ┌──────────────────────┐   │
│  │ 👥 Для сообщества    │   │  ← карточка type=community
│  │ Клуб, группа,        │   │
│  │ или экспертный круг  │   │
│  └──────────────────────┘   │
│                             │
└─────────────────────────────┘
```

После выбора типа — появляется второй экран:

```
┌─────────────────────────────┐
│ ← Назад                     │
├─────────────────────────────┤
│                             │
│  НАЗВАНИЕ                   │
│  ┌──────────────────────┐   │
│  │ T-Bank AI Stream     │   │  ← input
│  └──────────────────────┘   │
│                             │
│  АДРЕС (slug)               │
│  proofforge.io/             │
│  ┌──────────────────────┐   │
│  │ t-bank-ai            │   │  ← auto-generated from name
│  └──────────────────────┘   │
│  ✓ Адрес свободен           │  ← real-time validation
│                             │
│  ОПИСАНИЕ (необязательно)   │
│  ┌──────────────────────┐   │
│  │                      │   │
│  └──────────────────────┘   │
│                             │
│ ┌─────────────────────────┐ │
│ │    СОЗДАТЬ ПРОСТРАНСТВО │ │  ← primary CTA --win bg
│ └─────────────────────────┘ │
└─────────────────────────────┘
```

### Десктоп (>640px)

Та же логика, но в центрированном single-column layout (max-width: 480px), не fullscreen.

---

## Компонент: workspace-setup-screen.tsx

```tsx
type Step = 'type' | 'details';

interface WorkspaceSetupScreenProps {
  onSuccess: (workspace: Workspace) => void;
}

export function WorkspaceSetupScreen({ onSuccess }: WorkspaceSetupScreenProps) {
  const [step, setStep] = useState<Step>('type');
  const [type, setType] = useState<'org' | 'community' | null>(null);
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [description, setDescription] = useState('');
  const [slugAvailable, setSlugAvailable] = useState<boolean | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Auto-generate slug from name
  useEffect(() => {
    const generated = name.toLowerCase()
      .replace(/[^a-z0-9\s-]/g, '')
      .replace(/\s+/g, '-')
      .slice(0, 50);
    setSlug(generated);
  }, [name]);

  // Debounced slug availability check
  useEffect(() => {
    if (slug.length < 2) return;
    const timer = setTimeout(() => checkSlugAvailability(slug), 400);
    return () => clearTimeout(timer);
  }, [slug]);
  
  // ...handlers and render
}
```

---

## CSS (workspace-setup-screen.module.css)

```css
.screen {
  max-width: 480px;
  margin: 0 auto;
  padding: clamp(20px, 5vw, 48px) 16px;
}

.eyebrow {
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: var(--ink-mono);
  margin-bottom: 12px;
}

.title {
  font-family: var(--font-display);
  font-size: clamp(22px, 5vw, 32px);
  font-weight: 700;
  color: var(--ink-primary);
  margin-bottom: 32px;
}

.typeCard {
  border: 2px solid var(--border);
  padding: 20px;
  cursor: pointer;
  transition: border-color 140ms ease;
  margin-bottom: 12px;
}

.typeCard:hover { border-color: var(--border-strong); }
.typeCard.selected { border-color: var(--win); }

.typeCardTitle {
  font-size: 15px;
  font-weight: 700;
  color: var(--ink-primary);
  margin-bottom: 6px;
}

.typeCardDesc {
  font-size: 13px;
  color: var(--ink-mono);
  line-height: 1.5;
}

.field { margin-bottom: 20px; }

.fieldLabel {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: var(--ink-mono);
  margin-bottom: 8px;
  display: block;
}

.input {
  width: 100%;
  background: var(--bg-elevated);
  border: 2px solid var(--border);
  color: var(--ink-primary);
  font-family: var(--font-body);
  font-size: 14px;
  padding: 12px 14px;
  outline: none;
  transition: border-color 140ms ease;
  box-sizing: border-box;
}

.input:focus { border-color: var(--ink-primary); }

.slugRow {
  display: flex;
  align-items: center;
  gap: 8px;
}

.slugPrefix {
  font-size: 12px;
  color: var(--ink-mono);
  white-space: nowrap;
}

.slugStatus {
  font-size: 12px;
  margin-top: 6px;
}
.slugStatus.available { color: var(--win); }
.slugStatus.taken { color: var(--danger); }

.submitBtn {
  width: 100%;
  background: var(--win);
  color: #000;
  border: none;
  padding: 16px;
  font-family: var(--font-body);
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  cursor: pointer;
  transition: opacity 140ms ease;
  min-height: 52px;
}

.submitBtn:disabled { opacity: 0.4; cursor: not-allowed; }
```

---

## Поведение

1. Пользователь выбирает тип (org/community) → клик на карточку
2. Автоматически переходит к следующему шагу (без кнопки "Далее")
3. При вводе названия — slug генерируется автоматически
4. Slug можно редактировать вручную
5. После изменения slug — через 400ms дебаунс запрос на проверку доступности
6. Если slug занят → показать ошибку и заблокировать submit
7. Submit → `POST /v1/workspaces` → redirect на `/workspaces/:slug`

### API вызовы

```ts
// Проверка slug (не требует отдельного endpoint — реализовать через существующий GET /v1/workspaces/:slug)
async function checkSlugAvailability(slug: string): Promise<boolean> {
  const res = await fetch(`/api/v1/workspaces/${slug}`);
  return res.status === 404; // 404 = свободен
}

// Создание
async function createWorkspace(data: CreateWorkspaceInput): Promise<Workspace> {
  const res = await apiClient.post('/v1/workspaces', data);
  return res.data;
}
```

---

## Тексты (русский)

| Элемент | Текст |
|---------|-------|
| Eyebrow | СОЗДАТЬ ПРОСТРАНСТВО |
| Title step 1 | Для кого это пространство? |
| Card org title | Для команды или компании |
| Card org desc | Отдел, направление или рабочая группа |
| Card community title | Для сообщества или клуба |
| Card community desc | Закрытая группа, клуб по интересам |
| Field НАЗВАНИЕ | НАЗВАНИЕ |
| Field АДРЕС | АДРЕС |
| Slug available | ✓ Адрес свободен |
| Slug taken | ✗ Этот адрес уже занят |
| CTA | Создать пространство |
| Success toast | Пространство создано |

---

## Acceptance Criteria

- [ ] Шаг 1: два варианта типа, клик переключает на шаг 2 без лишних кнопок
- [ ] Slug автогенерируется из названия в реальном времени
- [ ] Slug валидируется на паттерн `^[a-z0-9-]{2,50}$` на клиенте
- [ ] Проверка доступности slug — дебаунс 400ms
- [ ] Submit заблокирован если slug занят или название пустое
- [ ] После успеха — redirect на страницу workspace
- [ ] Мобайл: кнопка submit в нижней части экрана (thumb zone), 52px высота
- [ ] Десктоп: форма max-width 480px, центрирована

---

## Что нельзя делать

- Не добавлять поля аватара, цвета, настроек в этот флоу (всё в settings)
- Не показывать advanced настройки на этапе создания
- Не делать multi-step с прогресс-баром — только 2 шага, переключение автоматическое
