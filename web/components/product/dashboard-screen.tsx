"use client";

import Link from "next/link";
import { type FormEvent, useEffect, useState, useTransition } from "react";

import { Button } from "@/components/core/button";
import { SectionShell } from "@/components/core/section-shell";
import { StatePanel } from "@/components/core/state-panel";
import { StatusPill } from "@/components/core/status-pill";
import { StakePanel } from "@/components/product/stake-panel";
import { ApiError, getDashboard, listCheckIns, loginUser, registerUser } from "@/lib/api";
import { formatDateLabel } from "@/lib/ui-copy";
import type { CheckIn, DashboardResponse, GoalView } from "@/lib/types";
import { type AppMode, useAppMode } from "@/lib/use-app-mode";

import styles from "./dashboard-screen.module.css";

type BuddyReviewItem = {
  checkIn: CheckIn;
  goal: GoalView;
};

type ScreenState =
  | { kind: "loading" }
  | { kind: "unauthenticated" }
  | { kind: "error"; message: string }
  | {
      kind: "ready";
      dashboard: DashboardResponse;
      buddyReviewQueue: BuddyReviewItem[];
    };

export function DashboardScreen() {
  const [screenState, setScreenState] = useState<ScreenState>({ kind: "loading" });
  const [registerError, setRegisterError] = useState<string | null>(null);
  const [loginError, setLoginError] = useState<string | null>(null);
  const [isRegistering, startRegisterTransition] = useTransition();
  const [isLoggingIn, startLoginTransition] = useTransition();
  const [mode, setMode] = useAppMode("all");

  useEffect(() => {
    void loadDashboard();
  }, []);

  async function loadDashboard() {
    try {
      const dashboard = await getDashboard();
      const goals = dashboard.goals ?? [];
      const buddyActiveGoals = goals.filter(
        (g) => g.role === "buddy" && g.goal.status === "active",
      );

      const buddyReviewQueue: BuddyReviewItem[] = [];
      const buddyResults = await Promise.all(
        buddyActiveGoals.map(async (g) => {
          try {
            const data = await listCheckIns(g.goal.id);
            return { goal: g, checkIns: data.check_ins ?? [] };
          } catch {
            return { goal: g, checkIns: [] as CheckIn[] };
          }
        }),
      );
      for (const { goal, checkIns } of buddyResults) {
        for (const ci of checkIns) {
          if (ci.status === "submitted") {
            buddyReviewQueue.push({ checkIn: ci, goal });
          }
        }
      }
      buddyReviewQueue.sort((a, b) => {
        const aDate = a.checkIn.submitted_at ?? a.checkIn.created_at;
        const bDate = b.checkIn.submitted_at ?? b.checkIn.created_at;
        return new Date(aDate).getTime() - new Date(bDate).getTime();
      });

      setScreenState({ kind: "ready", dashboard, buddyReviewQueue });
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setScreenState({ kind: "unauthenticated" });
        return;
      }

      setScreenState({
        kind: "error",
        message: error instanceof Error ? error.message : "Не удалось загрузить экран целей.",
      });
    }
  }

  function handleRegisterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setRegisterError(null);

    const formData = new FormData(event.currentTarget);
    startRegisterTransition(async () => {
      try {
        await registerUser({
          email: String(formData.get("email") ?? ""),
          display_name: String(formData.get("display_name") ?? ""),
        });
        await loadDashboard();
      } catch (error) {
        setRegisterError(
          error instanceof Error ? error.message : "Не удалось зарегистрировать пользователя.",
        );
      }
    });
  }

  function handleLoginSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoginError(null);

    const formData = new FormData(event.currentTarget);
    startLoginTransition(async () => {
      try {
        await loginUser(String(formData.get("email") ?? ""));
        await loadDashboard();
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) {
          setLoginError("Аккаунт с таким адресом не найден. Создайте его ниже.");
        } else {
          setLoginError(error instanceof Error ? error.message : "Не удалось войти.");
        }
      }
    });
  }

  if (screenState.kind === "loading") {
    return (
      <main className={styles.page}>
        <header className={styles.header}>
          <div>
            <span className="eyebrow">Операционный центр</span>
            <h1>Собираем состояние вашей цели</h1>
          </div>
          <StatusPill status="pending" label="Загружаем данные" />
        </header>
        <StatePanel
          tone="loading"
          title="Поднимаем контекст"
          description="Проверяем сессию и собираем текущее состояние по целям."
        />
      </main>
    );
  }

  if (screenState.kind === "error") {
    return (
      <main className={styles.page}>
        <header className={styles.header}>
          <div>
            <span className="eyebrow">Операционный центр</span>
            <h1>Экран целей временно недоступен</h1>
          </div>
          <StatusPill status="rejected" label="Нужна повторная попытка" />
        </header>
        <StatePanel
          tone="error"
          title="Не удалось собрать состояние"
          description={screenState.message}
          meta={
            <Button variant="secondary" onClick={() => void loadDashboard()}>
              Повторить загрузку
            </Button>
          }
        />
      </main>
    );
  }

  if (screenState.kind === "unauthenticated") {
    return (
      <main className={styles.page}>
        <header className={styles.header}>
          <div>
            <span className="eyebrow">Вход в систему</span>
            <h1>Войдите, чтобы держать цель под контролем</h1>
          </div>
          <StatusPill status="pending" label="Нужна сессия" />
        </header>

        <section className={styles.authGrid}>
          <SectionShell eyebrow="Вход" title="Уже есть аккаунт">
            <form className={styles.form} onSubmit={handleLoginSubmit}>
              <label className={styles.field}>
                <span>Электронная почта</span>
                <input name="email" type="email" placeholder="your@email.com" required />
              </label>
              {loginError ? (
                <p className={styles.formError} role="alert">
                  {loginError}
                </p>
              ) : null}
              <Button type="submit" disabled={isLoggingIn}>
                {isLoggingIn ? "Входим..." : "Войти"}
              </Button>
            </form>
          </SectionShell>

          <SectionShell eyebrow="Регистрация" title="Создать рабочий контур">
            <form className={styles.form} onSubmit={handleRegisterSubmit}>
              <label className={styles.field}>
                <span>Ваше имя</span>
                <input name="display_name" placeholder="Например, Артём" required />
              </label>
              <label className={styles.field}>
                <span>Электронная почта</span>
                <input name="email" type="email" placeholder="you@example.com" required />
              </label>
              {registerError ? (
                <p className={styles.formError} role="alert">
                  {registerError}
                </p>
              ) : null}
              <Button type="submit" disabled={isRegistering}>
                {isRegistering ? "Создаём аккаунт..." : "Создать аккаунт"}
              </Button>
            </form>
          </SectionShell>

          <SectionShell eyebrow="Что происходит дальше" title="Как устроен рабочий цикл">
            <div className={styles.copyBlock}>
              <p>
                После входа вы создаёте цель, приглашаете партнёра и ведёте её через
                подтверждения прогресса. Главный экран всегда показывает одно следующее
                действие, а не сваливает всё в одну панель.
              </p>
              <ul className={styles.ruleList}>
                <li>Цель создаётся отдельно от обзорного экрана.</li>
                <li>Партнёр принимает приглашение и проверяет подтверждения.</li>
                <li>Еженедельная сводка собирает ритм движения по цели.</li>
              </ul>
            </div>
          </SectionShell>
        </section>
      </main>
    );
  }

  const { dashboard, buddyReviewQueue } = screenState;
  const allGoals = dashboard.goals ?? [];
  const ownerGoals = allGoals.filter((g) => g.role === "owner");
  const buddyGoals = allGoals.filter((g) => g.role === "buddy");
  const pendingBuddyInvites = buddyGoals.filter(
    (g) => g.goal.status === "pending_buddy_acceptance",
  );

  const filtered =
    mode === "owner" ? ownerGoals : mode === "buddy" ? buddyGoals : allGoals;
  const sorted = sortGoalsForList(filtered);

  return (
    <main className={styles.page}>
      <header className={styles.header}>
        <div>
          <span className="eyebrow">Цели</span>
          <h1>{dashboard.user.display_name}, всё в одном списке</h1>
        </div>
        <Link className={styles.primaryLink} href="/goals/new">
          Создать цель
        </Link>
      </header>

      <ModeSwitch
        mode={mode}
        onChange={setMode}
        allCount={allGoals.length}
        ownerCount={ownerGoals.length}
        buddyCount={buddyGoals.length}
        reviewQueueCount={buddyReviewQueue.length}
        pendingInviteCount={pendingBuddyInvites.length}
      />

      {(mode === "all" || mode === "buddy") && buddyReviewQueue.length > 0 ? (
        <SectionShell
          eyebrow="Очередь проверки"
          title={`Ждут вашего решения (${buddyReviewQueue.length})`}
        >
          <ol className={styles.eventList}>
            {buddyReviewQueue.map((item) => (
              <li key={item.checkIn.id} className={styles.eventItem}>
                <div className={styles.eventHeader}>
                  <strong>
                    <Link href={`/check-ins/${item.checkIn.id}/review`}>
                      {item.goal.goal.title} · Чекин #{item.checkIn.id}
                    </Link>
                  </strong>
                  <StatusPill status="pending" label="На ревью" />
                </div>
                <p>Владелец: {item.goal.owner.display_name}</p>
                <div className={styles.eventFooter}>
                  <span>
                    Отправлен{" "}
                    {formatDateLabel(
                      item.checkIn.submitted_at ?? item.checkIn.created_at,
                      { month: "short", day: "numeric" },
                    )}
                  </span>
                  <Link
                    className={styles.eventLink}
                    href={`/check-ins/${item.checkIn.id}/review`}
                  >
                    Проверить →
                  </Link>
                </div>
              </li>
            ))}
          </ol>
        </SectionShell>
      ) : null}

      {sorted.length === 0 ? (
        <SectionShell
          eyebrow="Список целей"
          title={
            mode === "buddy"
              ? "Партнёрств пока нет"
              : mode === "owner"
                ? "Своих целей пока нет"
                : "Ни одной цели пока нет"
          }
        >
          <StatePanel
            tone="empty"
            title="Здесь будет список"
            description={
              mode === "buddy"
                ? "Когда вас пригласят как партнёра, цель появится в этом списке."
                : "Создайте первую цель и пригласите партнёра."
            }
            meta={
              mode !== "buddy" ? (
                <Link className={styles.primaryLink} href="/goals/new">
                  Создать цель
                </Link>
              ) : null
            }
          />
        </SectionShell>
      ) : (
        <SectionShell
          eyebrow="Список целей"
          title={`${modeLabel(mode)} (${sorted.length})`}
        >
          <ol className={styles.goalList}>
            {sorted.map((goal) => (
              <GoalListRow key={goal.goal.id} view={goal} />
            ))}
          </ol>
        </SectionShell>
      )}

    </main>
  );
}

function ModeSwitch({
  mode,
  onChange,
  allCount,
  ownerCount,
  buddyCount,
  reviewQueueCount,
  pendingInviteCount,
}: {
  mode: AppMode;
  onChange: (next: AppMode) => void;
  allCount: number;
  ownerCount: number;
  buddyCount: number;
  reviewQueueCount: number;
  pendingInviteCount: number;
}) {
  const buddyAttention = reviewQueueCount + pendingInviteCount;

  return (
    <div className={styles.modeSwitch} role="tablist" aria-label="Фильтр списка">
      <ModeTab
        active={mode === "all"}
        label="Все"
        badge={allCount > 0 ? String(allCount) : null}
        onClick={() => onChange("all")}
      />
      <ModeTab
        active={mode === "owner"}
        label="Свои"
        badge={ownerCount > 0 ? String(ownerCount) : null}
        onClick={() => onChange("owner")}
      />
      <ModeTab
        active={mode === "buddy"}
        label="Партнёрство"
        badge={
          buddyAttention > 0
            ? String(buddyAttention)
            : buddyCount > 0
              ? String(buddyCount)
              : null
        }
        attention={buddyAttention > 0}
        onClick={() => onChange("buddy")}
      />
    </div>
  );
}

function ModeTab({
  active,
  label,
  badge,
  attention,
  onClick,
}: {
  active: boolean;
  label: string;
  badge: string | null;
  attention?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      className={active ? styles.modeTabActive : styles.modeTab}
      onClick={onClick}
    >
      <span>{label}</span>
      {badge ? (
        <span className={attention ? styles.modeBadgeAttention : styles.modeBadge}>
          {badge}
        </span>
      ) : null}
    </button>
  );
}

function modeLabel(mode: AppMode): string {
  if (mode === "owner") return "Свои цели";
  if (mode === "buddy") return "Партнёрства";
  return "Все цели";
}

function sortGoalsForList(goals: GoalView[]): GoalView[] {
  return [...goals].sort((a, b) => {
    const aPending = a.goal.status === "pending_buddy_acceptance" ? 0 : 1;
    const bPending = b.goal.status === "pending_buddy_acceptance" ? 0 : 1;
    if (aPending !== bPending) return aPending - bPending;
    const aDl = a.goal.deadline_at;
    const bDl = b.goal.deadline_at;
    if (aDl && bDl) return aDl.localeCompare(bDl);
    if (aDl) return -1;
    if (bDl) return 1;
    return b.goal.created_at.localeCompare(a.goal.created_at);
  });
}

function GoalListRow({ view }: { view: GoalView }) {
  const isPending = view.goal.status === "pending_buddy_acceptance";
  const counterpart = view.role === "owner" ? view.buddy : view.owner;
  const counterpartLabel = view.role === "owner" ? "Партнёр" : "Владелец";
  const deadline = view.goal.deadline_at ? deadlineAccent(view.goal.deadline_at) : null;

  return (
    <li className={styles.goalRow}>
      <Link href={`/goals/${view.goal.id}`} className={styles.goalRowLink}>
        <div className={styles.goalRowMain}>
          <div className={styles.goalRowTitle}>
            <strong>{view.goal.title}</strong>
            <span className={view.role === "owner" ? styles.roleOwner : styles.roleBuddy}>
              {view.role === "owner" ? "Свои" : "Партнёр"}
            </span>
          </div>
          {view.goal.description ? (
            <p className={styles.goalRowDescription}>{view.goal.description}</p>
          ) : null}
          <div className={styles.goalRowMeta}>
            <span>
              {counterpartLabel}: <strong>{counterpart.display_name}</strong>
            </span>
            <span>
              Стрик: <strong>{view.goal.current_streak_count}</strong>
            </span>
          </div>
        </div>
        <div className={styles.goalRowAside}>
          <StatusPill
            status={isPending ? "pending" : "active"}
            label={isPending ? "Ждёт принятия" : "В работе"}
          />
          {deadline ? (
            <span className={`${styles.deadlineChip} ${styles[`deadline_${deadline.tone}`] ?? ""}`}>
              {deadline.label}
            </span>
          ) : (
            <span className={styles.deadlineChip}>без дедлайна</span>
          )}
        </div>
      </Link>
    </li>
  );
}

function deadlineAccent(iso: string): { label: string; tone: "overdue" | "soon" | "ok" } {
  const [y, m, d] = iso.split("-").map((part) => parseInt(part, 10));
  const target = new Date(y, m - 1, d);
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const days = Math.round((target.getTime() - today.getTime()) / (24 * 60 * 60 * 1000));
  const formatted = target.toLocaleDateString("ru-RU", {
    day: "numeric",
    month: "short",
  });
  if (days < 0) {
    return { label: `просрочено на ${Math.abs(days)} дн · ${formatted}`, tone: "overdue" };
  }
  if (days === 0) return { label: `сегодня · ${formatted}`, tone: "soon" };
  if (days <= 7) return { label: `через ${days} дн · ${formatted}`, tone: "soon" };
  return { label: `до ${formatted}`, tone: "ok" };
}

