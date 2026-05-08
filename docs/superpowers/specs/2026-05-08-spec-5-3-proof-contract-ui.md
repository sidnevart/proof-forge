# Спек 5.3 — Proof Contract UI (первый пруф за 2 минуты)

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 5 · Frontend — Пространства  
**Зависимости:** Спек 1.3 (proof_contracts API), 1.4 (movement modes)  
**Сложность:** M  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

Proof contract — главная механика v2. Пользователь должен создать свой первый контракт за ≤2 минуты. Три пути: разовый пруф, регулярный ритм, подключение к инициативе. Минимум вопросов.

**Принцип DNA П2, П3, П9:** контекст вместо инструкций, одно действие, скорость.

---

## Маршруты

```
/goals/:goalID/contract/new   ← создать новый контракт под цель
/goals/:goalID/contract/:id   ← просмотр контракта
```

---

## Файловая структура

```
web/app/(product)/goals/[goalID]/contract/
  new/page.tsx
  [contractID]/page.tsx

web/components/product/
  proof-contract-screen.tsx
  proof-contract-screen.module.css
  proof-contract-card.tsx
  proof-contract-card.module.css
```

---

## UI: /goals/:goalID/contract/new

### Шаг 0: Выбор пути (только если у цели нет movement_mode или он free_goal)

```
┌─────────────────────────────┐
│ ← [Название цели]           │
├─────────────────────────────┤
│                             │
│  ЧТО ХОЧЕШЬ СДЕЛАТЬ?       │  eyebrow
│                             │
│  ┌──────────────────────┐   │
│  │ → Сдать разовый пруф │   │  ← primary card (--win border)
│  │ Один конкретный      │   │
│  │ результат            │   │
│  └──────────────────────┘   │
│                             │
│  ┌──────────────────────┐   │
│  │ → Настроить ритм     │   │
│  │ Сдавать регулярно    │   │
│  │ по расписанию        │   │
│  └──────────────────────┘   │
│                             │
│  ┌──────────────────────┐   │
│  │ → Подключиться к     │   │
│  │   инициативе         │   │
│  │ Рабочие проекты      │   │
│  └──────────────────────┘   │
└─────────────────────────────┘
```

Если у цели movement_mode = `single_proof` → сразу шаг 1 (пропустить шаг 0).

---

### Путь А: Разовый пруф

**Шаг 1: Что докажешь**

```
ЧТО ТЫ ДОКАЖЕШЬ?

[textarea — свободный текст]
Например: "Покажу мини-пример с SupervisorJob 
           и объясню где применимо в нашем сервисе"

AI: Предложить формулировку →  (кнопка --frost цвет)
```

**Шаг 2: Чем докажешь (тип пруфа)**

```
ЧЕМ ДОКАЖЕШЬ?

[Артефакт]    [Заметка]
 MR, файл,    Текстовый
 документ     отчёт

[Демо]        [Эксперимент]
 Видео,       Результат
 скрин        исследования
```

**Шаг 3: Когда**

```
КОГДА ПОКАЖЕШЬ?

[Сегодня]     [Завтра]
[Эта пятница] [Выбрать дату →]
```

**Шаг 4: Кто подтвердит**

```
КТО ПОДТВЕРДИТ?

[Мой бадди — Артём]   ← если есть активный pact
[Другой человек]
[Без подтверждения]
```

**Финал — CTA:**

```
┌─────────────────────────────┐
│     СОЗДАТЬ КОНТРАКТ        │  ← --win background
└─────────────────────────────┘

Краткое резюме:
"До пятницы: демо SupervisorJob · Артём подтвердит"
```

---

### Путь Б: Регулярный ритм

**Шаг 1: Цель** (если не определена)

```
КАК ЧАСТО?

[Раз в день]     [Раз в неделю ✓]
[Раз в          [По пятницам]
 две недели]
```

**Шаг 2:** Первый пруф-контракт — те же шаги что в пути А.

---

## Компонент: proof-contract-screen.tsx

```tsx
type ContractPath = 'single' | 'rhythm' | 'initiative';

interface Props {
  goalId: string;
  goal: Goal;
  existingBuddy?: User;
}

export function ProofContractScreen({ goalId, goal, existingBuddy }: Props) {
  const [path, setPath] = useState<ContractPath | null>(
    goal.movement_mode === 'single_proof' ? 'single' : null
  );
  const [formStep, setFormStep] = useState(0);
  const [contract, setContract] = useState<Partial<ContractFormData>>({});

  // Шаги формы определяются выбранным путём
  const steps = getStepsForPath(path, goal, existingBuddy);
  
  const handleSubmit = async () => {
    const res = await apiClient.post(`/v1/goals/${goalId}/contracts`, {
      what_to_prove: contract.whatToProve,
      how_to_prove: contract.howToProve,
      proof_type: contract.proofType,
      due_at: contract.dueAt,
      buddy_id: contract.buddyId ?? null,
    });
    router.push(`/goals/${goalId}`);
  };

  // ...render
}
```

---

## CSS (proof-contract-screen.module.css)

```css
.screen {
  max-width: 480px;
  margin: 0 auto;
  padding: 0 16px;
  padding-bottom: calc(80px + env(safe-area-inset-bottom));
}

.pathCard {
  border: 2px solid var(--border);
  padding: 18px 20px;
  cursor: pointer;
  transition: border-color 140ms ease;
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 10px;
  min-height: 64px;
}

.pathCard:hover { border-color: var(--border-strong); }
.pathCard.primary { border-color: var(--win); }

.pathArrow {
  color: var(--win);
  font-size: 18px;
  flex-shrink: 0;
}

.pathTitle {
  font-size: 15px;
  font-weight: 700;
  color: var(--ink-primary);
}

.pathDesc {
  font-size: 12px;
  color: var(--ink-mono);
}

.textarea {
  width: 100%;
  background: var(--bg-elevated);
  border: 2px solid var(--border);
  color: var(--ink-primary);
  font-family: var(--font-body);
  font-size: 14px;
  line-height: 1.6;
  padding: 14px;
  outline: none;
  resize: none;
  min-height: 120px;
  box-sizing: border-box;
  transition: border-color 140ms ease;
}

.textarea:focus { border-color: var(--ink-primary); }

.aiBtn {
  background: none;
  border: 1px solid var(--frost);
  color: var(--frost);
  padding: 8px 14px;
  font-size: 12px;
  cursor: pointer;
  margin-top: 8px;
  transition: opacity 140ms ease;
}

.aiBtn:hover { opacity: 0.7; }

.summaryBox {
  background: var(--bg-surface);
  border-left: 3px solid var(--win);
  padding: 14px;
  font-size: 13px;
  color: var(--ink-primary);
  margin-bottom: 16px;
}

/* Sticky submit на мобайле */
.submitBar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 12px 16px;
  padding-bottom: calc(12px + env(safe-area-inset-bottom));
  background: var(--bg-base);
  border-top: 1px solid var(--border);
}

.submitBtn {
  width: 100%;
  background: var(--win);
  color: #000;
  border: none;
  padding: 16px;
  font-weight: 700;
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  cursor: pointer;
  min-height: 52px;
}

.submitBtn:disabled { opacity: 0.4; }
```

---

## Дата-выбор (quick-pick)

Вместо date-picker — quick-pick кнопки:

```tsx
const quickDates = [
  { label: 'Сегодня', value: endOfToday() },
  { label: 'Завтра', value: addDays(endOfToday(), 1) },
  { label: 'Эта пятница', value: nextFriday() },
  { label: 'Через неделю', value: addDays(today, 7) },
];
// + кнопка "Другая дата" → native <input type="date">
```

---

## Тексты (русский)

| Элемент | Текст |
|---------|-------|
| Eyebrow шаг 0 | ЧТО ХОЧЕШЬ СДЕЛАТЬ? |
| Path single | → Сдать разовый пруф |
| Path single desc | Один конкретный результат |
| Path rhythm | → Настроить регулярный ритм |
| Path initiative | → Подключиться к инициативе |
| Step: what | ЧТО ТЫ ДОКАЖЕШЬ? |
| Step: how | ЧЕМ ДОКАЖЕШЬ? |
| Step: when | КОГДА ПОКАЖЕШЬ? |
| Step: who | КТО ПОДТВЕРДИТ? |
| AI button | Предложить формулировку |
| Submit | Создать контракт |
| Without buddy | Без подтверждения |

---

## Acceptance Criteria

- [ ] Цель с `movement_mode=single_proof` → шаг 0 пропускается, сразу форма
- [ ] Каждый шаг занимает весь экран на мобайле
- [ ] После выбора типа пруфа → авто-переход к следующему шагу
- [ ] Quick-date кнопки: "Сегодня", "Завтра", "Пятница", "Через неделю"
- [ ] Если есть активный buddy → он предлагается первым вариантом на шаге "Кто подтвердит"
- [ ] Submit sticky-bar на мобайле над safe area
- [ ] Кнопка "Предложить формулировку" — вызывает AI endpoint (спек 8.1), результат подставляется в textarea
- [ ] После успеха → redirect на `/goals/:goalID` с toast "Контракт создан"
- [ ] Полный флоу ≤ 2 минуты (UX-тест)

---

## Что нельзя делать

- Не добавлять поле "описание" — только what_to_prove и how_to_prove
- Не делать buddy обязательным
- Не показывать date-picker по умолчанию — только quick-pick кнопки
