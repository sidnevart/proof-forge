"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { getDashboard, getWeeklyAssembly, listCircles } from "@/lib/api";
import type { CircleDetail, DashboardResponse, WeeklyAssembly } from "@/lib/types";
import { pluralizeRu } from "@/lib/plural";
import styles from "./page.module.css";

type PageState =
  | { kind: "loading" }
  | { kind: "error" }
  | {
      kind: "ready";
      dashboard: DashboardResponse;
      circles: CircleDetail[];
    };

export default function MePage() {
  const [state, setState] = useState<PageState>({ kind: "loading" });
  const [activeCircleId, setActiveCircleId] = useState<number | null>(null);
  const [assembly, setAssembly] = useState<WeeklyAssembly | null>(null);
  const [assemblyError, setAssemblyError] = useState(false);

  useEffect(() => {
    async function load() {
      try {
        const [dashboard, circlesRes] = await Promise.all([
          getDashboard(),
          listCircles(),
        ]);
        const circles = circlesRes.circles ?? [];
        setState({ kind: "ready", dashboard, circles });
        if (circles.length > 0) {
          setActiveCircleId(circles[0].circle.id);
        }
      } catch {
        setState({ kind: "error" });
      }
    }
    void load();
  }, []);

  // Fetch assembly whenever active circle changes
  useEffect(() => {
    if (activeCircleId === null) return;
    setAssembly(null);
    setAssemblyError(false);
    getWeeklyAssembly(activeCircleId)
      .then(setAssembly)
      .catch(() => setAssemblyError(true));
  }, [activeCircleId]);

  if (state.kind === "loading") {
    return (
      <div className={`page-shell ${styles.page}`}>
        <div className={styles.loading}>ЗАГРУЖАЕМ ДОСЬЕ...</div>
      </div>
    );
  }

  if (state.kind === "error") {
    return (
      <div className={`page-shell ${styles.page}`}>
        <div className={styles.loading}>ОШИБКА. <Link href="/dashboard">← НАЗАД</Link></div>
      </div>
    );
  }

  const { dashboard, circles } = state;
  const { user } = dashboard;
  const goals = dashboard.goals ?? [];

  const activeCircle = circles.find((c) => c.circle.id === activeCircleId) ?? null;
  const me = assembly?.standings.find((s) => s.user_id === user.id) ?? null;

  const rank = me?.rank ?? 0;
  const totalMembers = assembly?.standings.length ?? 0;
  const streak = me?.current_streak ?? 0;
  const score = me?.score ?? 0;
  const approvedWeeks = me?.approved_weeks ?? 0;
  const missedWeeks = me?.missed_weeks ?? 0;
  const status = me?.weekly_status ?? "no_goal";

  const verdict = getVerdict({ rank, totalMembers, streak, status, missedWeeks, approvedWeeks });

  return (
    <div className={`page-shell ${styles.page}`}>

      {/* Hero — большие цифры */}
      <section className={styles.hero}>
        <div className={styles.heroLeft}>
          <div className={styles.rankWrap}>
            <span className={styles.rankNum}>{rank > 0 ? rank : "—"}</span>
            {totalMembers > 0 && (
              <span className={styles.rankOf}>ИЗ {totalMembers}</span>
            )}
          </div>
          <div className={styles.rankLabel}>ТВОЙ РАНГ</div>
          {activeCircle && (
            <div className={styles.circleName}>КРУГ «{activeCircle.circle.name.toUpperCase()}»</div>
          )}
        </div>

        <div className={styles.heroRight}>
          <div className={styles.statRow}>
            <div className={styles.bigStat} data-tip="Дней подряд без пропуска">
              <span className={styles.bigNum}>{streak > 0 ? `🔥 ${streak}` : "0"}</span>
              <span className={styles.bigLabel}>СЕРИЯ</span>
            </div>
            <div className={styles.bigStat} data-tip="Очки за одобренные пруфы">
              <span className={styles.bigNum}>{score}</span>
              <span className={styles.bigLabel}>ОЧКИ</span>
            </div>
            <div className={styles.bigStat} data-tip="Недели с одобренным пруфом">
              <span className={styles.bigNum}>{approvedWeeks}</span>
              <span className={styles.bigLabel}>СДАНО</span>
            </div>
            <div className={styles.bigStat} data-tip="Недели без пруфа">
              <span className={`${styles.bigNum} ${missedWeeks > 0 ? styles.missed : ""}`}>
                {missedWeeks}
              </span>
              <span className={styles.bigLabel}>ПРОПУЩЕНО</span>
            </div>
          </div>
        </div>
      </section>

      {/* Вердикт */}
      <section className={`${styles.verdictSection} ${styles[`verdict_${verdict.tone}`]}`}>
        <div className={styles.verdictIcon}>{verdict.icon}</div>
        <div className={styles.verdictText}>
          <div className={styles.verdictHeadline}>{verdict.headline}</div>
          <div className={styles.verdictSub}>{verdict.sub}</div>
        </div>
      </section>

      {/* Цели */}
      <section className={styles.section}>
        <div className={styles.sectionHead}>
          <span className={styles.sectionLabel}>МОИ ЦЕЛИ</span>
          <Link href="/goals/new" className={styles.sectionCta}>+ ДОБАВИТЬ</Link>
        </div>

        {goals.length === 0 ? (
          <div className={styles.emptyBlock}>
            <span className={styles.emptyText}>ЦЕЛЕЙ НЕТ.</span>
            <Link href="/goals/new" className={styles.bigCta}>СОЗДАТЬ ПЕРВУЮ →</Link>
          </div>
        ) : (
          <div className={styles.goalGrid}>
            {goals.map((g) => (
              <div key={g.goal.id} className={`${styles.goalCard} ${styles[`goalCard_${g.goal.status}`]}`}>
                <div className={styles.goalTitle}>{g.goal.title}</div>
                <div className={styles.goalMeta}>
                  <span className={styles.goalStatusPill}>{formatGoalStatus(g.goal.status)}</span>
                  {g.goal.category && (
                    <span className={styles.goalCategory}>{g.goal.category.toUpperCase()}</span>
                  )}
                </div>
                {g.goal.status === "active" && (
                  <Link href={`/goals/${g.goal.id}/check-in`} className={styles.goalAction}>
                    СДАТЬ ПРУФ →
                  </Link>
                )}
              </div>
            ))}
          </div>
        )}
      </section>

      {/* Сравнение с кругом */}
      <section className={styles.section}>
        <div className={styles.sectionHead}>
          <span className={styles.sectionLabel}>ТЫ В КРУГЕ</span>
        </div>

        {/* Circle tabs — only when in multiple circles */}
        {circles.length > 1 && (
          <div className={styles.circleTabs}>
            {circles.map((c) => (
              <button
                key={c.circle.id}
                type="button"
                className={`${styles.circleTab} ${activeCircleId === c.circle.id ? styles.circleTabActive : ""}`}
                onClick={() => setActiveCircleId(c.circle.id)}
              >
                {c.circle.name.toUpperCase()}
                <span className={styles.circleTabMembers}>
                  {c.members.length}{" "}
                  {pluralizeRu(c.members.length, ["участник", "участника", "участников"])}
                </span>
              </button>
            ))}
          </div>
        )}

        {circles.length === 0 ? (
          <div className={styles.emptyBlock}>
            <span className={styles.emptyText}>ТЫ ПОКА НЕ В КРУГЕ.</span>
            <Link href="/goals/new" className={styles.bigCta}>СОЗДАТЬ ЦЕЛЬ →</Link>
          </div>
        ) : assembly && me ? (
          <div className={styles.standingsList}>
            {assembly.standings
              .sort((a, b) => a.rank - b.rank)
              .map((entry) => {
                const isMe = entry.user_id === user.id;
                const ahead = !isMe && entry.rank < me.rank;
                return (
                  <div key={entry.user_id} className={`${styles.standingRow} ${isMe ? styles.standingMe : ""}`}>
                    <span className={`${styles.standingRank} ${getRankClass(entry.rank)}`}>{entry.rank}</span>
                    <span className={styles.standingName}>
                      {entry.display_name.toUpperCase()}
                      {isMe && " ← ТЫ"}
                    </span>
                    <span className={styles.standingStreak}>
                      {entry.current_streak > 0 ? `🔥${entry.current_streak}` : "—"}
                    </span>
                    <span className={styles.standingScore}>{entry.score}</span>
                    {ahead && !isMe && (
                      <span className={styles.aheadBadge}>ВПЕРЕДИ</span>
                    )}
                  </div>
                );
              })}
          </div>
        ) : assembly && !me ? (
          <div className={styles.emptyBlock}>
            <span className={styles.emptyText}>У ТЕБЯ НЕТ АКТИВНЫХ ЦЕЛЕЙ В ЭТОМ КРУГЕ.</span>
            <Link href="/goals/new" className={styles.bigCta}>ДОБАВИТЬ ЦЕЛЬ →</Link>
          </div>
        ) : assemblyError ? (
          <div className={styles.emptyBlock}>
            <span className={styles.emptyText}>НЕ УДАЛОСЬ ЗАГРУЗИТЬ СТАТИСТИКУ КРУГА.</span>
          </div>
        ) : (
          <div className={styles.emptyBlock}>
            <span className={styles.emptyText}>ЗАГРУЖАЕМ ДАННЫЕ КРУГА...</span>
          </div>
        )}
      </section>

      {/* Быстрые ссылки */}
      <section className={styles.quickLinks}>
        <Link href="/dashboard" className={styles.quickLink}>← ДАШБОРД</Link>
        <Link href="/feed" className={styles.quickLink}>ЛЕНТА →</Link>
        <Link href="/settings/sharing" className={styles.quickLink}>НАСТРОЙКИ</Link>
      </section>

    </div>
  );
}

// ─── Helpers ───────────────────────────────────────────────────────────────

function formatGoalStatus(status: string): string {
  switch (status) {
    case "active": return "АКТИВНА";
    case "pending_buddy_acceptance": return "ЖДЁМ ПАРТНЁРА";
    case "completed": return "ЗАВЕРШЕНА";
    default: return status.toUpperCase();
  }
}

function getRankClass(rank: number): string {
  if (rank === 1) return styles.rankGold;
  if (rank <= 3) return styles.rankSilver;
  return styles.rankGray;
}

type Verdict = { tone: "good" | "warn" | "bad" | "neutral"; icon: string; headline: string; sub: string };

function getVerdict({
  rank, totalMembers, streak, status, missedWeeks, approvedWeeks,
}: {
  rank: number; totalMembers: number; streak: number;
  status: string; missedWeeks: number; approvedWeeks: number;
}): Verdict {
  if (status === "dropped") {
    return {
      tone: "bad",
      icon: "🥶",
      headline: "ТЫ ЗАМОРОЖЕН.",
      sub: "Один одобренный пруф снимает заморозку. Сделай это сейчас.",
    };
  }
  if (status === "at_risk") {
    return {
      tone: "warn",
      icon: "⚠️",
      headline: "ТЫ В ШАГЕ ОТ ЗАМОРОЗКИ.",
      sub: "Сдай пруф до конца дня — или завтра тебя вычеркнут.",
    };
  }
  if (rank === 1 && totalMembers > 1) {
    return {
      tone: "good",
      icon: "🏆",
      headline: `ПЕРВЫЙ В КРУГЕ.`,
      sub: `Серия ${streak}. Держи темп — остальные идут следом.`,
    };
  }
  if (streak >= 7) {
    return {
      tone: "good",
      icon: "🔥",
      headline: `СЕРИЯ ${streak}. ГОРИШЬ.`,
      sub: "Ни один пропуск за последние недели. Так держать.",
    };
  }
  if (missedWeeks > approvedWeeks && approvedWeeks + missedWeeks > 2) {
    return {
      tone: "bad",
      icon: "📉",
      headline: `ПРОПУСКОВ БОЛЬШЕ, ЧЕМ СДАЧ.`,
      sub: `${missedWeeks} против ${approvedWeeks}. Круг это видит.`,
    };
  }
  if (rank > 0 && totalMembers > 0 && rank > Math.ceil(totalMembers / 2)) {
    return {
      tone: "warn",
      icon: "👀",
      headline: `РАНГ ${rank} ИЗ ${totalMembers}.`,
      sub: "Нижняя половина круга. Есть куда расти.",
    };
  }
  if (status === "approved" || status === "comeback") {
    return {
      tone: "good",
      icon: "✅",
      headline: "ПРУФ ПРИНЯТ. ИДЁШЬ В РИТМЕ.",
      sub: "Продолжай завтра — серия не прощает пропусков.",
    };
  }
  return {
    tone: "neutral",
    icon: "🎯",
    headline: "ПОСТАВЬ ЦЕЛЬ. НАЧНИ СЕГОДНЯ.",
    sub: "Без цели нет давления. Без давления нет роста.",
  };
}
