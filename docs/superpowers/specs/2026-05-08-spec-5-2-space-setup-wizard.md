# Спек 5.2 — Wizard настройки Space за 20 секунд

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 5 · Frontend — Пространства  
**Зависимости:** Спек 1.1, 1.2, 5.1  
**Сложность:** L  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

После создания workspace нужно создать конкретное пространство (teamspace или community space) и настроить его. Целевое время — 20 секунд. Каждый шаг — одна карточка с 2–4 вариантами, без форм.

**Принцип DNA П4 и П9:** прогрессивное раскрытие + скорость над полнотой.

---

## Маршрут

```
/spaces/new?workspace_id=:id   ← wizard создания space
/spaces/new                    ← без workspace (standalone community)
```

---

## Файловая структура

```
web/app/(product)/spaces/
  new/
    page.tsx

web/components/product/
  space-setup-wizard.tsx
  space-setup-wizard.module.css
```

---

## 6 шагов wizard

### Шаг 1: Шаблон (обязательный)

```
ВЫБЕРИ ШАБЛОН
Под что настраивается пространство?

[ИПР команды]         [Рабочая инициатива]
Цели и прогресс       Исследования и вклад
сотрудников           в рабочие проекты

[Обучение навыку]     [Community Challenge]
Курс, тема или        Групповой вызов
технология            на ограниченный срок

[Спортивный вызов]    [Научный кружок]

[Подготовка к         [Внутренний AI-стрим]
 собеседованию]
```

Шаблон определяет defaults для следующих шагов.

Шаблоны и их defaults:

| Шаблон | Ритм default | Видимость default | Режим |
|--------|-------------|-------------------|-------|
| ИПР команды | weekly | leader only | teamspace |
| Рабочая инициатива | custom | leader + members | teamspace |
| Обучение навыку | weekly | circle | teamspace/community |
| Community Challenge | challenge 28d | public витрина | community |
| Спортивный вызов | daily | public витрина | community |
| Научный кружок | biweekly | circle | community |
| Подготовка к собеседованию | weekly | buddy only | circle |
| Внутренний AI-стрим | weekly | leader + members | teamspace |

---

### Шаг 2: Тип пространства (пропускается если уже понятно из контекста)

```
ЧТО СОЗДАЁМ?

[Teamspace]            [Community Space]
Для команды в          Для сообщества,
компании               клуба или группы

[Circle]
Маленькая группа
3–8 человек
```

Если пользователь пришёл из teamspace workspace — предлагаем teamspace или circle.  
Если из community workspace — предлагаем community space или circle.

---

### Шаг 3: Ритм

```
КАК ЧАСТО СДАВАТЬ ПРУФЫ?

[Раз в день]     [Раз в неделю ✓]
                  рекомендовано

[3 раза          [Свой ритм]
 в неделю]

Или особый период:
[7 дней]  [14 дней]  [28 дней]  [42 дня]
```

Шаблон предвыбирает вариант (отмечен --win border).

---

### Шаг 4: Видимость

```
КТО ВИДИТ ПРУФЫ?

[Только я          [Только circle]
 и бадди]

[Только лидер]     [Витрина лучших
                    (публично)]

[Только            
 агрегированная   
 статистика]      
```

---

### Шаг 5: Фокус метрик

```
ЧТО ВАЖНЕЕ ИЗМЕРЯТЬ?

[Обучение]    [Работа/ИПР]
Навыки и      Вклад в
прогресс      инициативы

[Соревнование]  [Наставничество]
Лидерборды      Помощь
и витрина       другим

[Платное        
 сообщество]    
Retention и    
конверсия      
```

---

### Шаг 6: Запуск

```
ГОТОВО К ЗАПУСКУ!

Название пространства:
[_______________________]  ← единственный input в wizard

→ Создать и пригласить участников
→ Скопировать ссылку
→ Отправить анонс в Telegram (если бот подключён)
→ Запустить позже
```

---

## Компонент: space-setup-wizard.tsx

```tsx
interface WizardState {
  template: TemplateId | null;
  spaceType: 'teamspace' | 'community_space' | 'circle' | null;
  rhythm: RhythmOption | null;
  visibility: VisibilityOption | null;
  metricsMode: MetricsMode | null;
  name: string;
}

interface StepConfig {
  id: string;
  title: string;
  options: Option[];
  isSkippable: boolean;
  defaultValue?: string; // из шаблона
}

export function SpaceSetupWizard({ workspaceId }: Props) {
  const [step, setStep] = useState(0);
  const [state, setState] = useState<WizardState>(initialState);
  const steps = buildSteps(state, workspaceId); // динамически по контексту
  
  const currentStep = steps[step];
  const progress = (step / (steps.length - 1)) * 100;

  return (
    <div className={styles.wizard}>
      <ProgressBar value={progress} />
      <StepHeader title={currentStep.title} step={step + 1} total={steps.length} />
      <OptionGrid
        options={currentStep.options}
        defaultValue={currentStep.defaultValue}
        onSelect={(value) => handleSelect(value)}
      />
      {currentStep.isSkippable && (
        <button className={styles.skipBtn} onClick={() => setStep(s => s + 1)}>
          Пропустить
        </button>
      )}
    </div>
  );
}
```

---

## Progress Bar

Тонкая полоска вверху, цвет --win, высота 2px. Показывает шаги 1-6.

```css
.progressBar {
  height: 2px;
  background: var(--border);
  position: relative;
}

.progressFill {
  height: 100%;
  background: var(--win);
  transition: width 200ms ease;
}
```

---

## CSS wizard

```css
.wizard {
  max-width: 560px;
  margin: 0 auto;
  padding: 0 16px;
}

.stepHeader {
  padding: 24px 0 20px;
}

.stepCounter {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--ink-mono);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  margin-bottom: 8px;
}

.stepTitle {
  font-family: var(--font-display);
  font-size: clamp(20px, 4vw, 28px);
  font-weight: 700;
  color: var(--ink-primary);
}

.optionGrid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-bottom: 24px;
}

/* Один шаблон в строке если их >4 */
@media (max-width: 400px) {
  .optionGrid { grid-template-columns: 1fr; }
}

.optionCard {
  border: 2px solid var(--border);
  padding: 16px;
  cursor: pointer;
  transition: border-color 140ms ease;
  min-height: 72px;
}

.optionCard:hover { border-color: var(--border-strong); }
.optionCard.selected { border-color: var(--win); }
.optionCard.recommended { border-color: var(--win); }

.optionTitle {
  font-size: 14px;
  font-weight: 700;
  color: var(--ink-primary);
  margin-bottom: 4px;
}

.optionDesc {
  font-size: 12px;
  color: var(--ink-mono);
  line-height: 1.4;
}

.recommendedBadge {
  font-size: 9px;
  font-family: var(--font-mono);
  color: var(--win);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  margin-bottom: 4px;
}

.skipBtn {
  background: none;
  border: none;
  color: var(--ink-mono);
  font-size: 13px;
  cursor: pointer;
  padding: 12px 0;
  text-decoration: underline;
  text-underline-offset: 3px;
}

.nameInput {
  width: 100%;
  background: var(--bg-elevated);
  border: 2px solid var(--border);
  color: var(--ink-primary);
  font-size: 16px;
  padding: 14px;
  outline: none;
  box-sizing: border-box;
  margin-bottom: 24px;
}

.nameInput:focus { border-color: var(--ink-primary); }

.launchActions {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.primaryLaunchBtn {
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
  width: 100%;
}

.secondaryLaunchBtn {
  background: none;
  border: 2px solid var(--border);
  color: var(--ink-primary);
  padding: 14px;
  font-size: 13px;
  cursor: pointer;
  width: 100%;
  text-align: left;
}
```

---

## Финальный шаг: launch actions

```
[→ Создать и открыть пространство]   ← --win background

[Скопировать ссылку-приглашение]     ← secondary border button
[Запустить позже]                    ← skip link
```

---

## Тексты (русский)

| Шаг | Заголовок |
|-----|-----------|
| 1 | Выбери шаблон |
| 2 | Что создаём? |
| 3 | Как часто сдавать пруфы? |
| 4 | Кто видит пруфы? |
| 5 | Что важнее измерять? |
| 6 | Придумай название |

| Кнопки | Текст |
|--------|-------|
| Primary launch | Создать пространство |
| Copy link | Скопировать ссылку приглашения |
| Skip | Пропустить |
| Later | Запустить позже |

---

## Acceptance Criteria

- [ ] Шаблон на шаге 1 предвыбирает defaults на шагах 3–5
- [ ] Клик на опцию → автопереход к следующему шагу (без кнопки "Далее")
- [ ] Шаг 2 пропускается если workspace_id уже определяет тип
- [ ] Прогресс-бар обновляется при каждом шаге
- [ ] Шаг 6: кнопка submit заблокирована если название пустое (min 2 символа)
- [ ] "Запустить позже" сохраняет пространство как draft и redirects на dashboard
- [ ] "Скопировать ссылку" → clipboard + toast "Ссылка скопирована"
- [ ] Весь wizard на мобайле — single column, карточки 2-col (или 1-col на узких)
- [ ] Полный проход всех шагов ≤ 20 секунд (UX-тест)

---

## Что нельзя делать

- Не добавлять текстовые поля (кроме имени на шаге 6)
- Не делать ни один шаг кроме шага 1 обязательным без skippable
- Не показывать более 6 опций на одном шаге
