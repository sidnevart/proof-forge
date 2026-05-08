"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";

import { cursorToCssVars } from "@/lib/cursor-to-css-var";
import { useMagneticHover } from "@/lib/use-magnetic-hover";

import { FaqAccordion } from "./faq-accordion";
import { LiveCircleMock } from "./live-circle-mock";
import { LiveCounters } from "./live-counters";
import { ScrollProgress } from "./scroll-progress";
import { SplitTextHeading } from "./split-text-heading";
import { TelegramFeed, type TgMessage } from "./telegram-feed";
import { TiltCard } from "./tilt-card";
import styles from "./landing-page.module.css";

export function LandingPage() {
  const heroRef = useRef<HTMLElement | null>(null);
  const tiksiRef = useRef<HTMLElement | null>(null);
  const ctaRef = useRef<HTMLAnchorElement | null>(null);

  // Magnetic CTA — pulls the button toward the cursor in a 120px radius.
  const { x: ctaMagX, y: ctaMagY } = useMagneticHover(ctaRef);

  // Fire-once section reveal: adds .is-revealed to all [data-animate] sections.
  // tiksiSection is skipped here — it has its own toggle observer below.
  useEffect(() => {
    if (typeof IntersectionObserver === "undefined") return;

    const nodes = document.querySelectorAll<HTMLElement>("[data-animate]");
    const tiksi = tiksiRef.current;

    const obs = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue;
          if (entry.target === tiksi) continue; // handled separately
          entry.target.classList.add("is-revealed");
          obs.unobserve(entry.target);
        }
      },
      { threshold: 0.12 }
    );

    nodes.forEach((n) => {
      if (n !== tiksi) obs.observe(n);
    });

    return () => obs.disconnect();
  }, []);

  // Cursor spotlight on the hero — updates CSS vars via rAF.
  // Skipped entirely if the user prefers reduced motion.
  useEffect(() => {
    const hero = heroRef.current;
    if (!hero || typeof window === "undefined") return;

    const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (reduce) return;

    let rafId = 0;
    let lastEvent: MouseEvent | null = null;

    const flush = () => {
      rafId = 0;
      if (!lastEvent) return;
      const rect = hero.getBoundingClientRect();
      const { x, y } = cursorToCssVars(lastEvent, rect);
      hero.style.setProperty("--cursor-x", x);
      hero.style.setProperty("--cursor-y", y);

      // Parallax tilt for LiveCircleMock: cursor offset → max ±3°
      const rx = (parseFloat(y) / 100 - 0.5) * -6; // cursor-y → rotateX
      const ry = (parseFloat(x) / 100 - 0.5) * 6;  // cursor-x → rotateY
      hero.style.setProperty("--mock-rx", `${rx.toFixed(2)}deg`);
      hero.style.setProperty("--mock-ry", `${ry.toFixed(2)}deg`);
    };

    const onMove = (e: MouseEvent) => {
      lastEvent = e;
      if (rafId === 0) {
        rafId = requestAnimationFrame(flush);
      }
    };

    hero.addEventListener("mousemove", onMove, { passive: true });
    return () => {
      hero.removeEventListener("mousemove", onMove);
      if (rafId !== 0) cancelAnimationFrame(rafId);
    };
  }, []);

  // Toggle .is-active on the countdown section while it's visible — drives glitch.
  // Also tracks intersection ratio to vary --tiksi-pulse-speed (2.5s fast → 5s slow).
  useEffect(() => {
    const node = tiksiRef.current;
    if (!node || typeof IntersectionObserver === "undefined") return;

    const THRESHOLDS = [0, 0.25, 0.5, 0.75, 1.0];

    const obs = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          entry.target.classList.toggle("is-active", entry.isIntersecting);
          // ratio 0→1 maps pulse speed 5s→2.5s (faster when more visible)
          const speed = 5 - entry.intersectionRatio * 2.5;
          (entry.target as HTMLElement).style.setProperty(
            "--tiksi-pulse-speed",
            `${speed.toFixed(2)}s`
          );
        }
      },
      { threshold: THRESHOLDS }
    );
    obs.observe(node);
    return () => obs.disconnect();
  }, []);

  return (
    <>
    <ScrollProgress />
    <main className={styles.page}>
      {/* 1. Hero */}
      <section ref={heroRef} className={styles.hero}>
        <div className={styles.heroCopy}>
          <SplitTextHeading tag="h1" className={styles.heroH1}>
            {"СО СЛЕДУЮЩЕГО\nПОНЕДЕЛЬНИКА\nУЖЕ БЫЛО."}
          </SplitTextHeading>
          <p className={styles.heroSub}>
            Среда решает больше, чем сила воли.
            Когда рядом люди которые двигаются — ты двигаешься тоже.
            Не потому что должен. Потому что не хочется выпадать.
          </p>
          <Link
            ref={ctaRef}
            href="/workspaces/new"
            className={styles.heroCta}
            style={{
              transform: `translate(${ctaMagX}px, ${ctaMagY}px)`,
            }}
          >
            СОБРАТЬ СВОЙ КРУГ
          </Link>
          <p className={styles.heroFine}>Бесплатно. Для команд, сообществ и одиночек.</p>
          <LiveCounters />
        </div>
        <div className={styles.heroMock}>
          <LiveCircleMock />
        </div>
      </section>

      {/* 2. Три опоры */}
      <section className={styles.supportSection} data-animate>
        <div className={styles.sectionHeader}>КАК НАМЕРЕНИЕ СТАНОВИТСЯ ДВИЖЕНИЕМ.</div>
        <div className={styles.supportGrid}>
          <TiltCard className={styles.supportCard}>
            <span className={styles.supportTitle}>СИЛЬНАЯ СРЕДА</span>
            <p className={styles.supportDesc}>
              Люди рядом которые идут. Не аудитория — компания.
              Видят твоё движение, ты видишь их. Без рейтинга стыда.
            </p>
          </TiltCard>
          <TiltCard className={styles.supportCard}>
            <span className={styles.supportTitle}>БАДДИ</span>
            <p className={styles.supportDesc}>
              Видит что ты сделал первым. Отвечает — одобрить или уточнить.
              Не контролёр — партнёр в одном ритме.
            </p>
          </TiltCard>
          <TiltCard className={styles.supportCard}>
            <span className={styles.supportTitle}>AI-ДОСЬЕ</span>
            <p className={styles.supportDesc}>
              Итог периода: навыки, результаты, паттерны роста.
              Готово для 1:1 и ИПР — без выдумки раз в квартал.
            </p>
          </TiltCard>
        </div>
      </section>

      {/* 3. Как это работает */}
      <section className={styles.howSection} data-animate>
        <div className={styles.sectionHeader}>ТРИ ШАГА. ОДИН ДЕНЬ.</div>
        <div className={styles.stepsGrid}>
          <TiltCard className={styles.step}>
            <span className={styles.stepNum}>01</span>
            <span className={styles.stepTitle}>ВЫБЕРИ ДВИЖЕНИЕ</span>
            <p className={styles.stepDesc}>
              Разовый результат, регулярный ритм или рабочая инициатива.
              AI предложит 3 конкретных пути с чего начать прямо сейчас.
            </p>
          </TiltCard>
          <TiltCard className={styles.step}>
            <span className={styles.stepNum}>02</span>
            <span className={styles.stepTitle}>ЗАФИКСИРУЙ РЕЗУЛЬТАТ</span>
            <p className={styles.stepDesc}>
              Скриншот, ссылка, текст — конкретный артефакт.
              AI проверяет до отправки. Бадди подтверждает. Застрял — оформи анти-пруф.
            </p>
          </TiltCard>
          <TiltCard className={styles.step}>
            <span className={styles.stepNum}>03</span>
            <span className={styles.stepTitle}>НАРАЩИВАЙ ДИСЦИПЛИНУ</span>
            <p className={styles.stepDesc}>
              Серия недель, личный рекорд, только сравнение с собой.
              Среда тянет вперёд — не ранжирует. Досье роста готово когда нужно.
            </p>
          </TiltCard>
        </div>
      </section>

      {/* 4. ДО КОНЦА ДНЯ — data-animate здесь тоже, но reveal-observer его пропускает */}
      <section ref={tiksiRef} className={styles.tiksiSection} data-animate>
        <div className={styles.tiksiLeft}>
          <div className={styles.tiksiCountdown}>14:23:47</div>
          <div className={styles.tiksiLabel}>ДО КОНЦА ДНЯ</div>
        </div>
        <div className={styles.tiksiRight}>
          <h2 className={styles.tiksiTitle}>СРЕДА ИДЁТ В ОДНОМ РИТМЕ.</h2>
          <ul className={styles.tiksiList}>
            <li><span className={styles.tiksiTime}>&lt;4ч</span> Бот пишет: «как там сегодня?». Можешь не отвечать.</li>
            <li><span className={styles.tiksiTime} style={{ color: "var(--danger)" }}>&lt;2ч</span> Таймер ярче. Бадди уже видит, что ты в работе.</li>
            <li><span className={styles.tiksiTime} style={{ color: "var(--danger)" }}>&lt;30мин</span> Бот пинганёт ещё раз. Без капса.</li>
            <li><span className={styles.tiksiTime} style={{ color: "var(--danger)" }}>0:00</span> Заморозка. Пометка, не наказание. Завтра одного шага достаточно.</li>
          </ul>
        </div>
      </section>

      {/* 5. Telegram */}
      <section className={styles.telegramSection} data-animate>
        <div className={styles.telegramHeader}>
          <div className={styles.sectionHeader}>СРЕДА НЕ ДАЁТ ВЫПАСТЬ.</div>
        </div>
        <TelegramFeed messages={TELEGRAM_MESSAGES} />
      </section>

      {/* 6. Approval */}
      <section className={styles.approvalSection} data-animate>
        <div className={styles.approvalLeft}>
          <h2 className={styles.approvalTitle}>БАДДИ — ПАРТНЁР, НЕ СУДЬЯ.</h2>
          <p className={styles.approvalDesc}>
            Смотрит что ты сделал, тапает «✓» или пишет «не понял, перепиши».
            Молчаливого отказа не бывает — без комментария кнопка
            не работает.
          </p>
        </div>
        <div className={styles.approvalRight}>
          <ApprovalCard />
        </div>
      </section>

      {/* 7. Для кого */}
      <section className={styles.teamsSection} data-animate>
        <div className={styles.sectionHeader}>ДЛЯ КАЖДОЙ РОЛИ — СВОЁ.</div>
        <p className={styles.teamsLead}>
          Одна механика, разные среды. Личная, командная, клубная.
        </p>
        <div className={styles.teamsGrid}>
          <TiltCard className={styles.teamsCard}>
            <span className={styles.teamsCardTitle}>ДЛЯ СЕБЯ</span>
            <p className={styles.teamsCardDesc}>
              Один бадди. Свой ритм. Только сравнение
              с собой вчерашним — никакого публичного ранжирования.
              Среда подталкивает, не заставляет.
            </p>
            <Link href="/dashboard" className={styles.teamsCardCta}>
              НАЧНИ С СЕБЯ →
            </Link>
          </TiltCard>
          <TiltCard className={styles.teamsCard}>
            <span className={styles.teamsCardTitle}>ДЛЯ КОМАНДЫ</span>
            <p className={styles.teamsCardDesc}>
              Тимлид видит энергию команды, не инструмент давления.
              У каждого свой ИПР. Артефакты работы — готовая база
              для ревью и 1:1.
            </p>
            <Link href="/workspaces/new" className={styles.teamsCardCta}>
              СОЗДАТЬ WORKSPACE →
            </Link>
          </TiltCard>
          <TiltCard className={styles.teamsCard}>
            <span className={styles.teamsCardTitle}>ДЛЯ СООБЩЕСТВА</span>
            <p className={styles.teamsCardDesc}>
              Telegram-клубы, внешние сообщества. Совместный challenge.
              Лучшие результаты недели — витрина без унижения аутсайдеров.
            </p>
            <Link href="/workspaces/new" className={styles.teamsCardCta}>
              ЗАПУСТИТЬ СООБЩЕСТВО →
            </Link>
          </TiltCard>
        </div>
      </section>

      {/* 8. Словарь */}
      <section className={styles.glossarySection} data-animate>
        <div className={styles.sectionHeader}>СЛОВАРЬ</div>
        <div className={styles.glossaryGrid}>
          {GLOSSARY.map((term) => (
            <TiltCard key={term.word} className={styles.glossaryItem}>
              <div className={styles.glossaryWord}>{term.word}</div>
              <div className={styles.glossaryDef}>{term.def}</div>
            </TiltCard>
          ))}
        </div>
      </section>

      {/* 9. FAQ */}
      <section className={styles.faqSection} data-animate>
        <div className={styles.sectionHeader}>ВОПРОСЫ</div>
        <FaqAccordion items={FAQ} />
      </section>

      {/* 10. Final CTA */}
      <section className={styles.finalCta} data-animate>
        <h2 className={styles.finalTitle}>СО СЛЕДУЮЩЕГО<br />ПОНЕДЕЛЬНИКА<br />УЖЕ БЫЛО.</h2>
        <Link href="/dashboard" className={styles.finalCtaBtn}>
          ЗАФИКСИРОВАТЬ ПЕРВЫЙ ШАГ →
        </Link>
        <p className={styles.finalFine}>
          Бесплатно. Для одиночек, команд и сообществ.
          Первый шаг — за 2 минуты.
        </p>
      </section>

      {/* Footer */}
      <footer className={styles.footer}>
        <span>ProofForge © 2026</span>
      </footer>
    </main>
    </>
  );
}

// ── Local components ────────────────────────────────────────────────────────

function ApprovalCard() {
  const [hoverState, setHoverState] = useState<"" | "approving" | "rejecting">("");

  return (
    <TiltCard
      className={[
        styles.approvalCard,
        hoverState === "approving" ? styles["approvalCard--approving"] : "",
        hoverState === "rejecting" ? styles["approvalCard--rejecting"] : "",
      ]
        .filter(Boolean)
        .join(" ")}
    >
      <div className={styles.approvalCardHeader}>АРТЁМ ЗАВЕРШИЛ В КРУГЕ «БЕЖАТЬ 5КМ»</div>
      <div className={styles.approvalCardBody}>Скриншот из Strava. 5.2 км. 28:14.</div>
      <div className={styles.approvalBtns}>
        <div
          className={styles.approvalBtnWin}
          onMouseEnter={() => setHoverState("approving")}
          onMouseLeave={() => setHoverState("")}
        >
          ✅ ОДОБРИТЬ
        </div>
        <div
          className={styles.approvalBtnDanger}
          onMouseEnter={() => setHoverState("rejecting")}
          onMouseLeave={() => setHoverState("")}
        >
          ❌ ОТКЛОНИТЬ
        </div>
        <div className={styles.approvalBtnMuted}>✏️ ДОПИСАТЬ</div>
      </div>
      <div className={styles.approvalHint}>
        ОТКЛОНИТЬ → ПОПРОСИТ КОММЕНТАРИЙ
      </div>
    </TiltCard>
  );
}

const TELEGRAM_MESSAGES: TgMessage[] = [
  { icon: "🟢", text: "АРТЁМ В РИТМЕ. ВТОРОЙ ДЕНЬ ПОДРЯД.", kind: "win" },
  { icon: "💬", text: "МАША МОЛЧИТ ВТОРОЙ ДЕНЬ. БАДДИ НАПИСАЛ.", kind: "warn" },
  { icon: "⏱", text: "2 ЧАСА ДО ПОЛУНОЧИ. ОДНА СТРОЧКА.", kind: "warn" },
  { icon: "⚡", text: "ДЕНИС ВЕРНУЛСЯ. ПЕРВЫЙ ШАГ ЗА ДВЕ НЕДЕЛИ.", kind: "win" },
  { icon: "📈", text: "У ИЛЬИ 14 ДНЕЙ ПОДРЯД.", kind: "win" },
];

const GLOSSARY = [
  {
    word: "ПРУФ",
    def: "Конкретный артефакт сделанной работы: ссылка, скриншот, текст. Бадди смотрит и подтверждает.",
  },
  {
    word: "PROOF CONTRACT",
    def: "Конкретная договорённость с собой и бадди: что сделаешь, каким артефактом, к какой дате.",
  },
  {
    word: "АНТИ-ПРУФ",
    def: "Честная фиксация попытки которая не получилась. Это не провал — это данные для следующего шага.",
  },
  {
    word: "СИЛЬНАЯ СРЕДА",
    def: "Люди рядом которые двигаются. Их присутствие создаёт естественное давление — без стыда и рейтинга.",
  },
  {
    word: "БАДДИ",
    def: "Партнёр который видит твои результаты первым. Не контролёр — поддержка. Без молчаливого отказа.",
  },
  {
    word: "ДОСЬЕ РОСТА",
    def: "AI-итог периода: навыки, результаты, паттерны, следующий фокус. Для 1:1 и ИПР.",
  },
];

const FAQ = [
  {
    q: "ЧТО ТАКОЕ PROOF CONTRACT?",
    a: "Конкретная договорённость: что именно сделаешь, каким артефактом это зафиксируешь, к какой дате. Создаётся за 2 минуты. AI предложит 3 варианта если не знаешь с чего начать.",
  },
  {
    q: "ЧТО ЕСЛИ ЗАСТРЯЛ И НЕ ЗНАЕШЬ КАК ДВИГАТЬСЯ?",
    a: "Оформи «анти-пруф» — честную фиксацию попытки. Это не провал, это данные. AI поможет сформулировать что пробовал, где блокер, каким будет следующий шаг.",
  },
  {
    q: "ЭТО БЕСПЛАТНО?",
    a: "Да. Полностью. На старте — без платных функций. Просто работает.",
  },
  {
    q: "МОИ ДАННЫЕ ВИДЯТ ВСЕ?",
    a: "Нет. Каждый видит только то что ты разрешил. Тимлид видит агрегированную картину команды, не инструмент давления. Нет публичного ранжирования аутсайдеров.",
  },
  {
    q: "А ДЛЯ КОМАНДЫ ЭТО РАБОТАЕТ?",
    a: "Да. Создай Workspace, добавь Teamspace — и тимлид видит энергию команды: кто движется, где блокеры, популярные направления. Артефакты работы собираются автоматически в базу для ревью и 1:1.",
    anchorId: "team-faq",
  },
  {
    q: "КАК ЗАПУСТИТЬ СООБЩЕСТВО?",
    a: "Создай Community Space — для Telegram-клубов и внешних сообществ. Совместный challenge, лучшие результаты недели как витрина, аналитика вовлечённости. Без публичного унижения участников.",
  },
];
