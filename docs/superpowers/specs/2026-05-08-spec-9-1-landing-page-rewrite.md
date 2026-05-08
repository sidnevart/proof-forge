# Спек 9.1 — Landing Page: Переписать (новое позиционирование)

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 9 · Посадочная страница  
**Зависимости:** нет (независимый маркетинговый модуль)  
**Сложность:** M  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

ProofForge — платформа "здоровой конкуренции": соревнуются доказательства, не люди. Новая посадочная страница должна объяснить это позиционирование за 10 секунд и конвертировать три аудитории: **корпоративные команды**, **Telegram-сообщества**, **индивидуальные пользователи**.

**DNA П1:** один ответ на вопрос "зачем" — сразу, без прокрутки.

---

## Маршрут

```
/ (root) → landing page
```

---

## Файловая структура

```
web/app/(marketing)/
  page.tsx                       ← обновить (уже существует)

web/components/marketing/
  landing-hero.tsx               ← НОВЫЙ
  landing-hero.module.css
  landing-audiences.tsx          ← НОВЫЙ
  landing-audiences.module.css
  landing-how-it-works.tsx       ← НОВЫЙ
  landing-how-it-works.module.css
  landing-proof-showcase.tsx     ← НОВЫЙ (витрина реальных пруфов)
  landing-proof-showcase.module.css
  landing-cta.tsx               ← НОВЫЙ
  landing-cta.module.css
```

---

## Структура страницы

```
[HERO]
[КАК ЭТО РАБОТАЕТ — 3 шага]
[ДЛЯ КОГО — 3 аудитории]
[ВИТРИНА ПРУФОВ — реальные примеры]
[ФИНАЛЬНЫЙ CTA]
```

---

## Hero секция

```
┌──────────────────────────────────────┐
│                                      │
│  Доказывай прогресс.                 │
│  Не рассказывай.                     │
│                                      │
│  Платформа для команд, где           │
│  соревнуются пруфы, а не люди.       │
│                                      │
│  [→ НАЧАТЬ БЕСПЛАТНО]                │
│                                      │
│  ┌────────────────────────────────┐  │
│  │ СЕЙЧАС                        │  │  ← NowCard mockup
│  │ Дедлайн сегодня               │  │
│  │ «Kotlin Coroutines»           │  │
│  │ [→ СДАТЬ ПРУФ]                │  │
│  └────────────────────────────────┘  │
└──────────────────────────────────────┘
```

```tsx
export function LandingHero() {
  return (
    <section className={styles.hero}>
      <div className={styles.heroContent}>
        <h1 className={styles.headline}>
          Доказывай прогресс.<br />
          Не рассказывай.
        </h1>
        <p className={styles.subheadline}>
          Платформа для команд, где соревнуются пруфы, а не люди.
        </p>
        <Link href="/signup" className={styles.heroCta}>
          → Начать бесплатно
        </Link>
      </div>
      
      <div className={styles.heroVisual}>
        <HeroNowCardMockup />
      </div>
    </section>
  );
}

function HeroNowCardMockup() {
  return (
    <div className={styles.mockCard}>
      <div className={styles.mockEyebrow}>СЕЙЧАС</div>
      <div className={styles.mockTitle}>Дедлайн сегодня</div>
      <div className={styles.mockGoal}>«Kotlin Coroutines — демо SupervisorJob»</div>
      <div className={styles.mockCta}>→ СДАТЬ ПРУФ СЕЙЧАС</div>
    </div>
  );
}
```

```css
/* landing-hero.module.css */
.hero {
  min-height: 80vh;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: clamp(40px, 8vw, 80px) clamp(20px, 5vw, 60px);
  gap: 48px;
}

@media (min-width: 641px) {
  .hero {
    flex-direction: row;
    align-items: center;
  }
  .heroContent { flex: 1; }
  .heroVisual { flex: 1; max-width: 420px; }
}

.headline {
  font-family: var(--font-display);
  font-size: clamp(36px, 6vw, 64px);
  font-weight: 700;
  color: var(--ink-primary);
  line-height: 1.05;
  margin-bottom: 20px;
}

.subheadline {
  font-size: clamp(16px, 2.5vw, 20px);
  color: var(--ink-mono);
  line-height: 1.5;
  max-width: 440px;
  margin-bottom: 32px;
}

.heroCta {
  display: inline-block;
  background: var(--win);
  color: var(--bg-base);
  padding: 16px 28px;
  font-size: 14px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  text-decoration: none;
  transition: opacity 140ms ease;
  min-height: 52px;
  line-height: 22px;
}

.heroCta:hover { opacity: 0.85; }

/* NowCard Mockup */
.mockCard {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-left: 4px solid var(--fire);
  padding: clamp(20px, 4vw, 32px);
}

.mockEyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: var(--ink-mono);
  margin-bottom: 12px;
}

.mockTitle {
  font-family: var(--font-display);
  font-size: clamp(20px, 3vw, 28px);
  font-weight: 700;
  color: var(--fire);
  margin-bottom: 8px;
}

.mockGoal {
  font-size: 14px;
  color: var(--ink-mono);
  margin-bottom: 24px;
  line-height: 1.4;
}

.mockCta {
  display: inline-block;
  background: var(--ink-primary);
  color: var(--bg-base);
  padding: 12px 20px;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
}
```

---

## Как это работает — 3 шага

```
1. Контракт        2. Пруф              3. Круг
───────────         ───────────          ───────────
Берёшь цель и      Выполнил —           Видишь чужие
обещаешь конкретный показываешь          пруфы, даёшь
результат к сроку  результат с          ревью, растёшь
                   артефактом           вместе
```

```tsx
const steps = [
  {
    number: '01',
    title: 'Контракт',
    desc: 'Берёшь цель и берёшь конкретное обязательство — что докажешь, к какому сроку, как.',
  },
  {
    number: '02',
    title: 'Пруф',
    desc: 'Выполнил — показываешь результат с артефактом: код, демо, статья. Не "занимался", а "сделал".',
  },
  {
    number: '03',
    title: 'Круг',
    desc: 'Видишь чужие пруфы, даёшь ревью, получаешь обратную связь от buddy. Растёшь вместе.',
  },
];

export function LandingHowItWorks() {
  return (
    <section className={styles.section}>
      <div className={styles.eyebrow}>КАК ЭТО РАБОТАЕТ</div>
      <div className={styles.steps}>
        {steps.map(step => (
          <div key={step.number} className={styles.step}>
            <div className={styles.stepNumber}>{step.number}</div>
            <div className={styles.stepTitle}>{step.title}</div>
            <div className={styles.stepDesc}>{step.desc}</div>
          </div>
        ))}
      </div>
    </section>
  );
}
```

```css
/* landing-how-it-works.module.css */
.section {
  padding: clamp(48px, 8vw, 80px) clamp(20px, 5vw, 60px);
  border-top: 1px solid var(--border);
}

.eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: var(--ink-mono);
  margin-bottom: 40px;
}

.steps {
  display: grid;
  grid-template-columns: 1fr;
  gap: 32px;
}

@media (min-width: 641px) {
  .steps { grid-template-columns: 1fr 1fr 1fr; gap: 24px; }
}

.step { position: relative; }

.stepNumber {
  font-family: var(--font-display);
  font-size: 48px;
  font-weight: 700;
  color: var(--border);
  line-height: 1;
  margin-bottom: 12px;
}

.stepTitle {
  font-family: var(--font-display);
  font-size: 20px;
  font-weight: 700;
  color: var(--ink-primary);
  margin-bottom: 10px;
}

.stepDesc {
  font-size: 14px;
  color: var(--ink-mono);
  line-height: 1.6;
}
```

---

## Для кого — 3 аудитории

```
КОРПОРАТИВНАЯ КОМАНДА        TELEGRAM-СООБЩЕСТВО         ДЛЯ СЕБЯ
────────────────────────     ────────────────────────     ────────────────────────
Команда доказывает рост,     Участники сезона             Строишь доказуемое
а не рассказывает на 1:1.    соревнуются пруфами,         портфолио навыков
Growth Dossier для ИПР       не болтовнёй                 для карьеры
за 2 минуты
[Для команды →]              [Для сообщества →]           [Для себя →]
```

```tsx
const audiences = [
  {
    tag: 'КОМАНДЫ',
    headline: 'Команда доказывает рост',
    subheadline: 'Не рассказывает на ежеквартальных 1:1, а показывает реальные результаты.',
    bullets: [
      'Тимлид видит энергию, а не отчёты',
      'Growth Dossier для ИПР за 2 минуты',
      'Без публичного рейтинга и демотивации',
    ],
    cta: { label: 'Для команды →', href: '/for-teams' },
    accent: 'var(--win)',
  },
  {
    tag: 'СООБЩЕСТВА',
    headline: 'Сезонные челленджи с пруфами',
    subheadline: 'Участники Telegram-клуба соревнуются доказательствами, а не болтовнёй.',
    bullets: [
      'Публичная витрина лучших пруфов',
      'Buddy-система взаимной поддержки',
      'Ведущий видит здоровье сезона',
    ],
    cta: { label: 'Для сообщества →', href: '/for-communities' },
    accent: 'var(--frost)',
  },
  {
    tag: 'ЛИЧНО',
    headline: 'Доказуемое портфолио навыков',
    subheadline: 'Каждый пруф — конкретный артефакт. Собери доказательства для следующего шага.',
    bullets: [
      'Streak и личные рекорды без публичного давления',
      'AI помогает не слиться с целью',
      'История пруфов как живое резюме',
    ],
    cta: { label: 'Для себя →', href: '/signup' },
    accent: 'var(--fire)',
  },
];

export function LandingAudiences() {
  return (
    <section className={styles.section}>
      <div className={styles.eyebrow}>ДЛЯ КОГО</div>
      <div className={styles.cards}>
        {audiences.map(a => (
          <AudienceCard key={a.tag} audience={a} />
        ))}
      </div>
    </section>
  );
}

function AudienceCard({ audience }) {
  return (
    <div className={styles.card} style={{ borderTopColor: audience.accent }}>
      <div className={styles.cardTag} style={{ color: audience.accent }}>{audience.tag}</div>
      <div className={styles.cardHeadline}>{audience.headline}</div>
      <div className={styles.cardSub}>{audience.subheadline}</div>
      <ul className={styles.cardList}>
        {audience.bullets.map(b => (
          <li key={b} className={styles.cardItem}>{b}</li>
        ))}
      </ul>
      <Link href={audience.cta.href} className={styles.cardCta} style={{ color: audience.accent, borderColor: audience.accent }}>
        {audience.cta.label}
      </Link>
    </div>
  );
}
```

```css
/* landing-audiences.module.css */
.section {
  padding: clamp(48px, 8vw, 80px) clamp(20px, 5vw, 60px);
  border-top: 1px solid var(--border);
}

.eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: var(--ink-mono);
  margin-bottom: 40px;
}

.cards {
  display: grid;
  grid-template-columns: 1fr;
  gap: 16px;
}

@media (min-width: 641px) {
  .cards { grid-template-columns: 1fr 1fr 1fr; gap: 20px; }
}

.card {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-top: 3px solid var(--border);
  padding: clamp(20px, 3vw, 28px);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.cardTag {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.12em;
}

.cardHeadline {
  font-family: var(--font-display);
  font-size: clamp(18px, 2.5vw, 22px);
  font-weight: 700;
  color: var(--ink-primary);
  line-height: 1.2;
}

.cardSub {
  font-size: 14px;
  color: var(--ink-mono);
  line-height: 1.5;
}

.cardList {
  list-style: none;
  padding: 0;
  margin: 0;
  flex: 1;
}

.cardItem {
  font-size: 13px;
  color: var(--ink-secondary);
  padding: 4px 0;
  padding-left: 16px;
  position: relative;
}

.cardItem::before {
  content: '→';
  position: absolute;
  left: 0;
  color: var(--ink-mono);
}

.cardCta {
  display: inline-block;
  border: 1px solid;
  padding: 10px 16px;
  font-size: 12px;
  font-weight: 700;
  text-decoration: none;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  align-self: flex-start;
  transition: opacity 140ms ease;
}

.cardCta:hover { opacity: 0.75; }
```

---

## Витрина пруфов (социальное доказательство)

Статически захардкоженные примеры реальных пруфов (анонимизированные / с разрешения).

```tsx
const SHOWCASE_PROOFS = [
  {
    goalTag: 'Kotlin',
    goalTitle: 'Coroutines',
    text: 'Реализовал SupervisorJob с обработкой ошибок в продакшн-сервисе. Вот ссылка на PR и объяснение почему именно этот паттерн...',
    buddyApproved: true,
    author: 'Разработчик из команды T-Bank',
  },
  {
    goalTag: 'System Design',
    goalTitle: 'CAP теорема',
    text: 'Разобрал три реальных кейса где мы выбирали между CP и AP в нашей системе. Включая неочевидный момент с Cassandra...',
    buddyApproved: true,
    author: 'Senior engineer, ML Community',
  },
  {
    goalTag: 'AI-инструменты',
    goalTitle: 'Промпт-паттерны',
    text: 'Собрал 5 рабочих шаблонов промптов для code review и написал почему каждый работает с точки зрения психологии LLM...',
    buddyApproved: false,
    author: 'Тимлид, Backend Club',
  },
];

export function LandingProofShowcase() {
  return (
    <section className={styles.section}>
      <div className={styles.eyebrow}>ПРИМЕРЫ ПРУФОВ</div>
      <div className={styles.subtitle}>Что значит "доказать прогресс"</div>
      <div className={styles.cards}>
        {SHOWCASE_PROOFS.map((p, i) => (
          <ShowcaseProofCard key={i} proof={p} />
        ))}
      </div>
    </section>
  );
}
```

---

## Финальный CTA

```tsx
export function LandingCTA() {
  return (
    <section className={styles.section}>
      <div className={styles.content}>
        <div className={styles.headline}>Готов доказывать?</div>
        <div className={styles.sub}>
          Первый пруф — за 2 минуты. Без кредитки.
        </div>
        <Link href="/signup" className={styles.cta}>
          → Начать бесплатно
        </Link>
        <div className={styles.hint}>или</div>
        <Link href="/demo" className={styles.secondaryCta}>
          Посмотреть демо →
        </Link>
      </div>
    </section>
  );
}
```

```css
/* landing-cta.module.css */
.section {
  padding: clamp(60px, 10vw, 100px) clamp(20px, 5vw, 60px);
  border-top: 1px solid var(--border);
  text-align: center;
}

.content { max-width: 480px; margin: 0 auto; }

.headline {
  font-family: var(--font-display);
  font-size: clamp(28px, 5vw, 48px);
  font-weight: 700;
  color: var(--ink-primary);
  margin-bottom: 16px;
}

.sub {
  font-size: 16px;
  color: var(--ink-mono);
  margin-bottom: 32px;
}

.cta {
  display: inline-block;
  background: var(--win);
  color: var(--bg-base);
  padding: 18px 36px;
  font-size: 14px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  text-decoration: none;
  transition: opacity 140ms ease;
}

.cta:hover { opacity: 0.85; }

.hint {
  font-size: 13px;
  color: var(--ink-mono);
  margin: 16px 0 8px;
}

.secondaryCta {
  display: inline-block;
  font-size: 13px;
  font-weight: 700;
  color: var(--ink-mono);
  text-decoration: none;
  border-bottom: 1px solid var(--border);
  transition: color 140ms ease;
}

.secondaryCta:hover { color: var(--ink-primary); }
```

---

## page.tsx — сборка

```tsx
export default function LandingPage() {
  return (
    <main>
      <LandingHero />
      <LandingHowItWorks />
      <LandingAudiences />
      <LandingProofShowcase />
      <LandingCTA />
    </main>
  );
}
```

---

## Acceptance Criteria

- [ ] Hero: headline видна без скролла на любом экране ≥320px
- [ ] HeroNowCardMockup: border-left цвет --fire, текст понятен без контекста
- [ ] "Начать бесплатно" → `/signup`, "Для команды →" → `/for-teams`
- [ ] 3-шаговая секция: мобайл 1-col, десктоп 3-col
- [ ] AudienceCard: border-top соответствует accent цвету аудитории
- [ ] ProofShowcase: 3 карточки видны без горизонтального скролла на мобайле
- [ ] FinCTA: две кнопки (primary + secondary)
- [ ] Страница не использует JS для первоначального рендера (static, no client components)
- [ ] Open Graph meta: title "ProofForge — Доказывай прогресс", description ≤ 160 символов

---

## Что нельзя делать

- Не добавлять цену/тарифы на эту страницу (отдельный /pricing)
- Не добавлять видео — только статический контент
- Не использовать изображения/иллюстрации — только текст и mockup-компоненты
- Не добавлять форму на главной — только кнопки для перехода
