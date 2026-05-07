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
            {"ТВОИ ДРУЗЬЯ\nУЖЕ РАБОТАЮТ.\nТЫ ЕЩЁ НЕТ."}
          </SplitTextHeading>
          <p className={styles.heroSub}>
            Круг из 8 человек. 28 дней. Сдавай пруф каждый день.
            Пропустил — все видят. Заморозка публичная.
          </p>
          <Link
            ref={ctaRef}
            href="/dashboard"
            className={styles.heroCta}
            style={{
              transform: `translate(${ctaMagX}px, ${ctaMagY}px)`,
            }}
          >
            СОБРАТЬ КРУГ
          </Link>
          <p className={styles.heroFine}>Бесплатно. Без приложения. Через Telegram.</p>
          <LiveCounters />
        </div>
        <div className={styles.heroMock}>
          <LiveCircleMock />
        </div>
      </section>

      {/* 2. Как это работает */}
      <section className={styles.howSection} data-animate>
        <div className={styles.sectionHeader}>КАК ЭТО РАБОТАЕТ</div>
        <div className={styles.stepsGrid}>
          <TiltCard className={styles.step}>
            <span className={styles.stepNum}>01</span>
            <span className={styles.stepTitle}>СОБЕРИ КРУГ</span>
            <p className={styles.stepDesc}>
              Пригласи 2–7 друзей. Без них платформа не работает. Давление — это они.
            </p>
          </TiltCard>
          <TiltCard className={styles.step}>
            <span className={styles.stepNum}>02</span>
            <span className={styles.stepTitle}>СДАВАЙ ПРУФ</span>
            <p className={styles.stepDesc}>
              Каждый день — одно подтверждение. Фото, текст, ссылка. Бадди проверяет.
            </p>
          </TiltCard>
          <TiltCard className={styles.step}>
            <span className={styles.stepNum}>03</span>
            <span className={styles.stepTitle}>БУДЬ В РИТМЕ</span>
            <p className={styles.stepDesc}>
              Таблица, серии, события. Кто не сдал — заморозка. Кто вернулся — камбэк.
            </p>
          </TiltCard>
        </div>
      </section>

      {/* 3. ДО ЗАМОРОЗКИ — data-animate здесь тоже, но reveal-observer его пропускает */}
      <section ref={tiksiRef} className={styles.tiksiSection} data-animate>
        <div className={styles.tiksiLeft}>
          <div className={styles.tiksiCountdown}>14:23:47</div>
          <div className={styles.tiksiLabel}>ДО ЗАМОРОЗКИ</div>
        </div>
        <div className={styles.tiksiRight}>
          <h2 className={styles.tiksiTitle}>ДАВЛЕНИЕ НАРАСТАЕТ К НОЧИ.</h2>
          <ul className={styles.tiksiList}>
            <li><span className={styles.tiksiTime}>&lt;4ч</span> Напоминание в Telegram.</li>
            <li><span className={styles.tiksiTime} style={{ color: "var(--danger)" }}>&lt;2ч</span> Таймер краснеет. Все видят твой статус.</li>
            <li><span className={styles.tiksiTime} style={{ color: "var(--danger)" }}>&lt;30мин</span> Пульс таймера. Круг следит.</li>
            <li><span className={styles.tiksiTime} style={{ color: "var(--danger)" }}>0:00</span> Заморозка. Публично.</li>
          </ul>
        </div>
      </section>

      {/* 4. Telegram */}
      <section className={styles.telegramSection} data-animate>
        <div className={styles.telegramHeader}>
          <div className={styles.sectionHeader}>БОТ В TELEGRAM ВЕДЁТ ТЕБЯ ОБРАТНО.</div>
        </div>
        <TelegramFeed messages={TELEGRAM_MESSAGES} />
      </section>

      {/* 5. Approval */}
      <section className={styles.approvalSection} data-animate>
        <div className={styles.approvalLeft}>
          <h2 className={styles.approvalTitle}>ТВОЙ БАДДИ ПРОВЕРЯЕТ. БЕЗ ОБМАНА.</h2>
          <p className={styles.approvalDesc}>
            После каждого пруфа — два тапа в Telegram. Одобрить или отклонить.
            Без обхода. Без накруток.
          </p>
        </div>
        <div className={styles.approvalRight}>
          <ApprovalCard />
        </div>
      </section>

      {/* 6. Словарь */}
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

      {/* 7. FAQ */}
      <section className={styles.faqSection} data-animate>
        <div className={styles.sectionHeader}>ВОПРОСЫ</div>
        <FaqAccordion items={FAQ} />
      </section>

      {/* 8. Final CTA */}
      <section className={styles.finalCta} data-animate>
        <h2 className={styles.finalTitle}>СОБЕРИ КРУГ.<br />НАЧНИ СЕГОДНЯ.</h2>
        <Link href="/dashboard" className={styles.finalCtaBtn}>
          СОЗДАТЬ КРУГ →
        </Link>
        <p className={styles.finalFine}>Бесплатно. Telegram-нотификации включены.</p>
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
      <div className={styles.approvalCardHeader}>АРТЁМ СДАЛ В КРУГЕ «БЕЖАТЬ 5КМ»</div>
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
    </TiltCard>
  );
}

const TELEGRAM_MESSAGES: TgMessage[] = [
  { icon: "🟢", text: "АРТЁМ ПЕРВЫЙ. ТЫ ЕЩЁ НЕТ.", kind: "win" },
  { icon: "🥶", text: "МАША ЗАМОРОЖЕНА. НЕ ПРИСОЕДИНЯЙСЯ.", kind: "danger" },
  { icon: "⏱", text: "2 ЧАСА ДО ЗАМОРОЗКИ. СДАЙ.", kind: "warn" },
  { icon: "⚡", text: "КАМБЭК. ДЕНИС ВЕРНУЛСЯ.", kind: "win" },
  { icon: "🏆", text: "ОБОГНАЛ ИЛЬЮ. РАНГ 4 → 3.", kind: "win" },
];

const GLOSSARY = [
  {
    word: "ПРУФ",
    def: "Подтверждение прогресса. Фото, видео, ссылка, скриншот. Бадди смотрит и решает — зачтено или нет.",
  },
  {
    word: "СЕРИЯ",
    def: "Количество дней подряд без пропуска. Упал до нуля — начинаешь с нуля. Рекорд виден всему кругу.",
  },
  {
    word: "РАНГ",
    def: "Место в круге по набранным очкам за неделю. Меняется при каждом одобрении и заморозке.",
  },
  {
    word: "ЗАМОРОЗКА",
    def: "Публичный статус за пропущенный день. Виден всем в круге. Один одобренный пруф снимает.",
  },
  {
    word: "БАДДИ",
    def: "Партнёр, которому ты отправляешь пруф. Он одобряет или отклоняет. Без его решения очки не засчитываются.",
  },
];

const FAQ = [
  {
    q: "ЧТО ЕСЛИ Я ВЫПАЛ?",
    a: "Заморозка видна всему кругу. Один одобренный пруф — и ты снят с заморозки. Это называется камбэк.",
  },
  {
    q: "ЭТО БЕСПЛАТНО?",
    a: "Да. Полностью. На старте — без платных функций. Просто работает.",
  },
  {
    q: "НУЖНО ПРИЛОЖЕНИЕ?",
    a: "Нет. Всё через браузер + Telegram-бот для нотификаций и одобрений.",
  },
  {
    q: "МОЙ КРУГ БУДЕТ ВИДЕН ВСЕМ?",
    a: "Только участникам твоего круга. Публичная лента — opt-in, можно не включать.",
  },
];
