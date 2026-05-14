"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import {
  getDashboard,
  getMyMemberships,
  getTelegramLinkStatus,
  getWeeklyAssembly,
  listCircles,
  logoutUser,
} from "@/lib/api";
import type { CircleDetail, DashboardResponse, MyMembership, WeeklyAssembly } from "@/lib/types";
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

type ProfileTab = "overview" | "goals" | "spaces" | "standings";

const TABS: { key: ProfileTab; label: string }[] = [
  { key: "overview", label: "ОБЗОР" },
  { key: "goals", label: "КРУГИ" },
  { key: "spaces", label: "ПРОСТРАНСТВА" },
  { key: "standings", label: "РЕЙТИНГ" },
];

export default function MePage() {
  const router = useRouter();
  const [state, setState] = useState<PageState>({ kind: "loading" });
  const [tab, setTab] = useState<ProfileTab>("overview");
  const [activeCircleId, setActiveCircleId] = useState<number | null>(null);
  const [assembly, setAssembly] = useState<WeeklyAssembly | null>(null);
  const [assemblyError, setAssemblyError] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const [tgLinked, setTgLinked] = useState<boolean | null>(null);
  const [tgUsername, setTgUsername] = useState<string>("");
  const [memberships, setMemberships] = useState<MyMembership[]>([]);

  async function handleLogout() {
    if (loggingOut) return;
    const ok = window.confirm(
      "Выйти из аккаунта?\n\nЛогин и пароль понадобятся, чтобы вернуться."
    );
    if (!ok) return;
    setLoggingOut(true);
    try {
      await logoutUser();
    } finally {
      router.push("/");
    }
  }

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

    // Load Telegram link status in background — non-fatal.
    getTelegramLinkStatus()
      .then((s) => {
        setTgLinked(s.linked);
        if (s.username) setTgUsername(s.username);
      })
      .catch(() => setTgLinked(false));

    // Load memberships in background — non-fatal.
    getMyMemberships()
      .then((res) => setMemberships(res.memberships ?? []))
      .catch(() => setMemberships([]));
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
        <div className={styles.loading}>
          ОШИБКА. <Link href="/dashboard">← НАЗАД</Link>
        </div>
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

  const verdict = getVerdict({
    rank,
    totalMembers,
    streak,
    status,
    missedWeeks,
    approvedWeeks,
  });

  // Goal count for tab badge
  const goalCount = goals.length;
  const membershipCount = memberships.length;

  return (
    <div className={`page-shell ${styles.page}`}>
      {/* Hero — компактные цифры */}
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
            <div className={styles.circleName}>
              КРУГ «{activeCircle.circle.name.toUpperCase()}»
            </div>
          )}
        </div>

        <div className={styles.heroRight}>
          <div className={styles.statRow}>
            <div className={styles.bigStat} data-tip="Дней подряд без пропуска">
              <span className={styles.bigNum}>
                {streak > 0 ? `🔥 ${streak}` : "0"}
              </span>
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
              <span
                className={`${styles.bigNum} ${
                  missedWeeks > 0 ? styles.missed : ""
                }`}
              >
                {missedWeeks}
              </span>
              <span className={styles.bigLabel}>ПРОПУЩЕНО</span>
            </div>
          </div>
        </div>
      </section>

      {/* Tab Bar */}
      <nav className={styles.tabBar} role="tablist" aria-label="Разделы профиля">
        {TABS.map(({ key, label }) => (
          <button
            key={key}
            type="button"
            role="tab"
            aria-selected={tab === key}
            className={`${styles.tab} ${tab === key ? styles.tabActive : ""}`}
            onClick={() => setTab(key)}
          >
            {label}
            {key === "goals" && goalCount > 0 && (
              <span className={styles.tabBadge}>{goalCount}</span>
            )}
            {key === "spaces" && membershipCount > 0 && (
              <span className={styles.tabBadge}>{membershipCount}</span>
            )}
          </button>
        ))}
      </nav>

      {/* ОБЗОР */}
      {tab === "overview" && (
        <section
          className={`${styles.verdictSection} ${styles[`verdict_${verdict.tone}`]}`}
        >
          <div className={styles.verdictIcon}>{verdict.icon}</div>
          <div className={styles.verdictText}>
            <div className={styles.verdictHeadline}>{verdict.headline}</div>
            <div className={styles.verdictSub}>{verdict.sub}</div>
          </div>
        </section>
      )}

      {/* КРУГИ */}
      {tab === "goals" && (
        <section className={styles.section}>
          <div className={styles.sectionHead}>
            <span className={styles.sectionLabel}>МОИ КРУГИ</span>
            <Link href="/goals/new" className={styles.sectionCta}>
              + ДОБАВИТЬ
            </Link>
          </div>

          {goals.length === 0 ? (
            <div className={styles.emptyBlock}>
              <span className={styles.emptyText}>КРУГОВ НЕТ.</span>
              <Link href="/goals/new" className={styles.bigCta}>
                СОЗДАТЬ ПЕРВЫЙ →
              </Link>
            </div>
          ) : (
            <div className={styles.goalGrid}>
              {goals.map((g) => (
                <div
                  key={g.goal.id}
                  className={`${styles.goalCard} ${
                    styles[`goalCard_${g.goal.status}`]
                  }`}
                >
                  <div className={styles.goalTitle}>{g.goal.title}</div>
                  <div className={styles.goalMeta}>
                    <span className={styles.goalStatusPill}>
                      {formatGoalStatus(g.goal.status)}
                    </span>
                    {g.goal.category && (
                      <span className={styles.goalCategory}>
                        {g.goal.category.toUpperCase()}
                      </span>
                    )}
                  </div>
                  {g.goal.status === "active" && (
                    <Link
                      href={`/goals/${g.goal.id}/check-in`}
                      className={styles.goalAction}
                    >
                      СДАТЬ ПРУФ →
                    </Link>
                  )}
                </div>
              ))}
            </div>
          )}
        </section>
      )}

      {/* ПРОСТРАНСТВА */}
      {tab === "spaces" && (
        <section className={styles.section}>
          <div className={styles.sectionHead}>
            <span className={styles.sectionLabel}>МОИ ПРОСТРАНСТВА</span>
          </div>
          {memberships.length === 0 ? (
            <div className={styles.emptyBlock}>
              <span className={styles.emptyText}>ПРОСТРАНСТВ НЕТ.</span>
              <span className={styles.emptyText} style={{ fontSize: "14px", opacity: 0.7 }}>
                Команды и сообщества появятся здесь.
              </span>
            </div>
          ) : (
            <div className={styles.membershipList}>
              {memberships.map((m) => {
                const href =
                  m.space_type === "teamspace"
                    ? `/teamspaces/${m.space_id}`
                    : `/community-spaces/${m.space_id}/analytics`;
                return (
                  <Link
                    key={`${m.space_type}-${m.space_id}`}
                    href={href}
                    className={styles.membershipRow}
                  >
                    <span className={styles.membershipType}>
                      {m.space_type === "teamspace" ? "КОМАНДА" : "СООБЩЕСТВО"}
                    </span>
                    <span className={styles.membershipName}>
                      {m.space_name.toUpperCase()}
                    </span>
                    <span className={styles.membershipRole}>
                      {m.role.toUpperCase()}
                    </span>
                  </Link>
                );
              })}
            </div>
          )}
        </section>
      )}

      {/* РЕЙТИНГ */}
      {tab === "standings" && (
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
                  className={`${styles.circleTab} ${
                    activeCircleId === c.circle.id ? styles.circleTabActive : ""
                  }`}
                  onClick={() => setActiveCircleId(c.circle.id)}
                >
                  {c.circle.name.toUpperCase()}
                  <span className={styles.circleTabMembers}>
                    {c.members.length}{" "}
                    {pluralizeRu(c.members.length, [
                      "участник",
                      "участника",
                      "участников",
                    ])}
                  </span>
                </button>
              ))}
            </div>
          )}

          {circles.length === 0 ? (
            <div className={styles.emptyBlock}>
              <span className={styles.emptyText}>ТЫ ПОКА НЕ В КРУГЕ.</span>
              <Link href="/goals/new" className={styles.bigCta}>
                СОЗДАТЬ КРУГ →
              </Link>
            </div>
          ) : assembly && me ? (
            <div className={styles.standingsList}>
              {assembly.standings
                .sort((a, b) => a.rank - b.rank)
                .map((entry) => {
                  const isMe = entry.user_id === user.id;
                  const ahead = !isMe && entry.rank < me.rank;
                  return (
                    <div
                      key={entry.user_id}
                      className={`${styles.standingRow} ${
                        isMe ? styles.standingMe : ""
                      }`}
                    >
                      <span
                        className={`${styles.standingRank} ${getRankClass(entry.rank)}`}
                      >
                        {entry.rank}
                      </span>
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
              <span className={styles.emptyText}>В ЭТОМ КРУГЕ ПОКА ТИХО.</span>
              <Link href="/goals/new" className={styles.bigCta}>
                ОТКРЫТЬ КРУГ →
              </Link>
            </div>
          ) : assemblyError ? (
            <div className={styles.emptyBlock}>
              <span className={styles.emptyText}>
                НЕ УДАЛОСЬ ЗАГРУЗИТЬ СТАТИСТИКУ КРУГА.
              </span>
            </div>
          ) : (
            <div className={styles.emptyBlock}>
              <span className={styles.emptyText}>ЗАГРУЖАЕМ ДАННЫЕ КРУГА...</span>
            </div>
          )}
        </section>
      )}

      {/* Аккаунт — внизу намеренно */}
      <section className={styles.accountSection}>
        <div className={styles.accountHead}>АККАУНТ</div>
        <div className={styles.accountRow}>
          <span className={styles.accountEmail}>{user.email}</span>
          <button
            type="button"
            className={styles.logoutBtn}
            onClick={handleLogout}
            disabled={loggingOut}
          >
            {loggingOut ? "ВЫХОД…" : "ВЫЙТИ"}
          </button>
        </div>
        {tgLinked !== null && (
          <div className={styles.telegramRow}>
            <div className={styles.telegramInfo}>
              <span
                className={`${styles.telegramDot} ${
                  tgLinked ? styles.telegramDotOn : ""
                }`}
              />
              <span className={styles.telegramLabel}>
                {tgLinked
                  ? tgUsername
                    ? `Telegram: @${tgUsername}`
                    : "Telegram подключен"
                  : "Telegram не подключен"}
              </span>
            </div>
            <Link href="/settings/telegram" className={styles.telegramAction}>
              {tgLinked ? "НАСТРОИТЬ →" : "ПОДКЛЮЧИТЬ →"}
            </Link>
          </div>
        )}
      </section>
    </div>
  );
}

// ─── Helpers ───────────────────────────────────────────────────────────────

function formatGoalStatus(status: string): string {
  switch (status) {
    case "active":
      return "АКТИВНА";
    case "pending_buddy_acceptance":
      return "ЖДЁМ ПАРТНЁРА";
    case "completed":
      return "ЗАВЕРШЕНА";
    default:
      return status.toUpperCase();
  }
}

function getRankClass(rank: number): string {
  if (rank === 1) return styles.rankGold;
  if (rank <= 3) return styles.rankSilver;
  return styles.rankGray;
}

type Verdict = {
  tone: "good" | "warn" | "bad" | "neutral";
  icon: string;
  headline: string;
  sub: string;
};

function getVerdict({
  rank,
  totalMembers,
  streak,
  status,
  missedWeeks,
  approvedWeeks,
}: {
  rank: number;
  totalMembers: number;
  streak: number;
  status: string;
  missedWeeks: number;
  approvedWeeks: number;
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
    headline: "СОБЕРИ КРУГ. НАЧНИ СЕГОДНЯ.",
    sub: "Без круга нет давления. Без давления нет роста.",
  };
}
