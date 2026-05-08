"use client";

import Link from "next/link";
import { type FormEvent, useCallback, useEffect, useState, useTransition } from "react";
import { useRouter } from "next/navigation";

import { GoalCircleCard } from "@/components/product/goal-circle-card";
import { NowCard } from "@/components/product/now-card";
import { PersonalProgressBar } from "@/components/product/personal-progress-bar";
import { ApiError, getDashboard, getNowCard, getPersonalLeaderboard, loginUser, registerUser } from "@/lib/api";
import { pluralizeRu } from "@/lib/plural";
import type { DashboardResponse, GoalView, NowCardData, PersonalLeaderboard } from "@/lib/types";

import styles from "./dashboard-screen.module.css";

type ScreenState =
  | { kind: "loading" }
  | { kind: "unauthenticated" }
  | { kind: "error"; message: string }
  | { kind: "no_goals"; dashboard: DashboardResponse }
  | { kind: "has_goals"; dashboard: DashboardResponse; goals: GoalView[] };

const MEMBER_FORMS: [string, string, string] = ["участник", "участника", "участников"];

export function DashboardScreen() {
  const router = useRouter();
  const [screenState, setScreenState] = useState<ScreenState>({ kind: "loading" });
  const [registerError, setRegisterError] = useState<string | null>(null);
  const [loginError, setLoginError] = useState<string | null>(null);
  const [nowCard, setNowCard] = useState<NowCardData | null>(null);
  const [personalStats, setPersonalStats] = useState<PersonalLeaderboard | null>(null);
  const [isRegistering, startRegisterTransition] = useTransition();
  const [isLoggingIn, startLoginTransition] = useTransition();

  const loadDashboard = useCallback(async () => {
    try {
      const dashboard = await getDashboard();
      const goals = dashboard.goals ?? [];
      if (goals.length === 0) {
        setScreenState({ kind: "no_goals", dashboard });
      } else {
        setScreenState({ kind: "has_goals", dashboard, goals });
      }
      // Load smart card and stats in background, errors are non-fatal.
      void getNowCard().then(setNowCard).catch(() => null);
      void getPersonalLeaderboard().then(setPersonalStats).catch(() => null);
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setScreenState({ kind: "unauthenticated" });
        return;
      }
      setScreenState({
        kind: "error",
        message: error instanceof Error ? error.message : "Не удалось загрузить дашборд.",
      });
    }
  }, []);

  useEffect(() => {
    void loadDashboard();
  }, [loadDashboard]);

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
        <div className={styles.loadingState}>
          <span className={styles.loadingText}>ЗАГРУЗКА...</span>
        </div>
      </main>
    );
  }

  if (screenState.kind === "error") {
    return (
      <main className={styles.page}>
        <div className={styles.errorState}>
          <span className={styles.errorText}>ОШИБКА.</span>
          <p className={styles.errorDesc}>{screenState.message}</p>
          <button className={styles.retryBtn} onClick={() => void loadDashboard()}>
            ПОВТОРИТЬ
          </button>
        </div>
      </main>
    );
  }

  if (screenState.kind === "unauthenticated") {
    return (
      <main className={styles.page}>
        <header className={styles.authHeader}>
          <h1 className={styles.authHeaderTitle}>ВОЙДИТЕ, ЧТОБЫ ДЕРЖАТЬ КРУГ ПОД КОНТРОЛЕМ</h1>
        </header>

        <div className={styles.authGrid}>
          <div className={styles.authCard}>
            <div className={styles.cardLabel}>ВХОД</div>
            <h2 className={styles.cardTitle}>Уже есть аккаунт</h2>
            <form className={styles.form} onSubmit={handleLoginSubmit}>
              <label className={styles.field}>
                <span className={styles.fieldLabel}>ЭЛЕКТРОННАЯ ПОЧТА</span>
                <input
                  className={styles.input}
                  name="email"
                  type="email"
                  placeholder="your@email.com"
                  required
                />
              </label>
              {loginError ? (
                <p className={styles.formError} role="alert">{loginError}</p>
              ) : null}
              <button className={styles.submitBtn} type="submit" disabled={isLoggingIn}>
                {isLoggingIn ? "ВХОДИМ..." : "ВОЙТИ"}
              </button>
            </form>
          </div>

          <div className={styles.authCard}>
            <div className={styles.cardLabel}>РЕГИСТРАЦИЯ</div>
            <h2 className={styles.cardTitle}>Создать аккаунт</h2>
            <form className={styles.form} onSubmit={handleRegisterSubmit}>
              <label className={styles.field}>
                <span className={styles.fieldLabel}>ВАШЕ ИМЯ</span>
                <input
                  className={styles.input}
                  name="display_name"
                  placeholder="Например, Артём"
                  required
                />
              </label>
              <label className={styles.field}>
                <span className={styles.fieldLabel}>ЭЛЕКТРОННАЯ ПОЧТА</span>
                <input
                  className={styles.input}
                  name="email"
                  type="email"
                  placeholder="you@example.com"
                  required
                />
              </label>
              {registerError ? (
                <p className={styles.formError} role="alert">{registerError}</p>
              ) : null}
              <button className={styles.submitBtn} type="submit" disabled={isRegistering}>
                {isRegistering ? "СОЗДАЁМ АККАУНТ..." : "СОЗДАТЬ АККАУНТ"}
              </button>
            </form>
          </div>

          <div className={styles.authCard}>
            <div className={styles.cardLabel}>КАК ЭТО РАБОТАЕТ</div>
            <ul className={styles.ruleList}>
              <li>Создай круг — пригласи партнёра.</li>
              <li>Пригласи партнёра по ссылке. Он принимает приглашение.</li>
              <li>Каждый день сдаёшь пруф. Партнёр одобряет или нет.</li>
            </ul>
          </div>
        </div>
      </main>
    );
  }

  if (screenState.kind === "no_goals") {
    const firstCircle = screenState.dashboard.circles?.[0] ?? null;
    return (
      <main className={styles.page}>
        <div className={styles.emptyWrap}>
          <article className={styles.emptyCard} aria-labelledby="empty-state-heading">
            <div className={styles.eyebrow}>
              {firstCircle
                ? `КРУГ «${firstCircle.name.toUpperCase()}» · ${firstCircle.member_count} ${pluralizeRu(firstCircle.member_count, MEMBER_FORMS)}`
                : "НОВЫЙ КРУГ"}
            </div>
            <h1 id="empty-state-heading" className={styles.emptyHeading}>
              ЧТО БУДЕШЬ ДОКАЗЫВАТЬ?
            </h1>
            <p className={styles.emptySub}>
              Создай круг — партнёр получит приглашение.
            </p>
            <Link href="/onboard/goal" className={styles.ctaSolid}>
              СОЗДАТЬ КРУГ
            </Link>
          </article>
        </div>
      </main>
    );
  }

  // has_goals
  const { goals } = screenState;

  return (
    <main className={styles.page}>
      {nowCard && <NowCard data={nowCard} />}
      {personalStats && <PersonalProgressBar data={personalStats} />}
      <div className={styles.goalStack}>
        {goals.map((g) => (
          <GoalCircleCard
            key={g.goal.id}
            circleId={g.goal.circle_id}
            goalId={g.goal.id}
            goalTitle={g.goal.title}
            goalStatus={g.goal.status}
            buddyStatus={g.pact.status}
            seasonDay={computeSeasonDay(g.goal.created_at)}
            seasonDaysLeft={computeSeasonDaysLeft(g.goal.created_at)}
            seasonStatus={computeSeasonStatus(g.goal.created_at)}
            membersCount={1}
            lastCheckInAt={null}
            viewerRole={g.viewer_role}
            onCheckIn={() => router.push(`/goals/${g.goal.id}/check-in`)}
          />
        ))}
      </div>

      <div className={styles.addMoreWrap}>
        <Link href="/onboard/goal" className={styles.ctaGhost}>
          + СОЗДАТЬ ЕЩЁ ОДИН КРУГ
        </Link>
      </div>
    </main>
  );
}

// ── Season helpers ────────────────────────────────────────────────────────────

function computeSeasonDay(createdAt: string): number {
  const created = new Date(createdAt).getTime();
  const now = Date.now();
  const daysSince = Math.floor((now - created) / (1000 * 60 * 60 * 24));
  return Math.min(Math.max(daysSince + 1, 1), 7);
}

function computeSeasonDaysLeft(createdAt: string): number {
  const created = new Date(createdAt).getTime();
  const now = Date.now();
  const daysSince = Math.floor((now - created) / (1000 * 60 * 60 * 24));
  return Math.max(7 - daysSince, 0);
}

function computeSeasonStatus(createdAt: string): "active" | "completed" {
  const created = new Date(createdAt).getTime();
  const now = Date.now();
  const daysSince = Math.floor((now - created) / (1000 * 60 * 60 * 24));
  return daysSince >= 7 ? "completed" : "active";
}
