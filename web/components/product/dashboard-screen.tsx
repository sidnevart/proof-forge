"use client";

import { type FormEvent, useEffect, useState, useTransition } from "react";

import { Button } from "@/components/core/button";
import { SectionShell } from "@/components/core/section-shell";
import { StatePanel } from "@/components/core/state-panel";
import { StatusPill } from "@/components/core/status-pill";
import { ApiError, createGoal, getDashboard, loginUser, registerUser } from "@/lib/api";
import type { DashboardResponse, GoalView } from "@/lib/types";

import styles from "./dashboard-screen.module.css";

type ScreenState =
  | { kind: "loading" }
  | { kind: "unauthenticated" }
  | { kind: "error"; message: string }
  | { kind: "ready"; dashboard: DashboardResponse };

export function DashboardScreen() {
  const [screenState, setScreenState] = useState<ScreenState>({ kind: "loading" });
  const [registerError, setRegisterError] = useState<string | null>(null);
  const [loginError, setLoginError] = useState<string | null>(null);
  const [goalError, setGoalError] = useState<string | null>(null);
  const [isRegistering, startRegisterTransition] = useTransition();
  const [isLoggingIn, startLoginTransition] = useTransition();
  const [isCreatingGoal, startCreateGoalTransition] = useTransition();

  useEffect(() => {
    void loadDashboard();
  }, []);

  async function loadDashboard() {
    try {
      const dashboard = await getDashboard();
      setScreenState({ kind: "ready", dashboard });
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setScreenState({ kind: "unauthenticated" });
        return;
      }

      setScreenState({
        kind: "error",
        message: error instanceof Error ? error.message : "Не удалось загрузить dashboard.",
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
          setLoginError("Аккаунт с таким email не найден. Зарегистрируйтесь.");
        } else {
          setLoginError(error instanceof Error ? error.message : "Не удалось войти.");
        }
      }
    });
  }

  function handleGoalSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setGoalError(null);

    const form = event.currentTarget;
    const formData = new FormData(form);
    startCreateGoalTransition(async () => {
      try {
        await createGoal({
          title: String(formData.get("title") ?? ""),
          description: String(formData.get("description") ?? ""),
          buddy_name: String(formData.get("buddy_name") ?? ""),
          buddy_email: String(formData.get("buddy_email") ?? ""),
        });
        form.reset();
        await loadDashboard();
      } catch (error) {
        setGoalError(error instanceof Error ? error.message : "Не удалось создать goal.");
      }
    });
  }

  if (screenState.kind === "loading") {
    return (
      <main className={styles.page}>
        <header className={styles.header}>
          <div>
            <span className="eyebrow">Dashboard bootstrap</span>
            <h1>Checkpoint control room</h1>
          </div>
          <StatusPill status="pending" label="Loading surface" />
        </header>
        <StatePanel
          tone="loading"
          title="Поднимаем внешний контур"
          description="Система проверяет сессию и загружает реальный goals dashboard."
        />
      </main>
    );
  }

  if (screenState.kind === "error") {
    return (
      <main className={styles.page}>
        <header className={styles.header}>
          <div>
            <span className="eyebrow">Dashboard fault</span>
            <h1>Checkpoint control room</h1>
          </div>
          <StatusPill status="rejected" label="Surface degraded" />
        </header>
        <StatePanel
          tone="error"
          title="Dashboard недоступен"
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
            <span className="eyebrow">Onboarding gate</span>
            <h1>Войдите в accountability loop</h1>
          </div>
          <StatusPill status="pending" label="Session required" />
        </header>

        <section className={styles.authGrid}>
          <SectionShell eyebrow="Войти" title="Уже есть аккаунт">
            <form className={styles.form} onSubmit={handleLoginSubmit}>
              <label className={styles.field}>
                <span>Email</span>
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

          <SectionShell eyebrow="Registration" title="Создайте рабочий контур">
            <form className={styles.form} onSubmit={handleRegisterSubmit}>
              <label className={styles.field}>
                <span>Ваше имя</span>
                <input name="display_name" placeholder="Например, Артём" required />
              </label>
              <label className={styles.field}>
                <span>Email</span>
                <input name="email" type="email" placeholder="you@example.com" required />
              </label>
              {registerError ? (
                <p className={styles.formError} role="alert">
                  {registerError}
                </p>
              ) : null}
              <Button type="submit" disabled={isRegistering}>
                {isRegistering ? "Регистрируем..." : "Создать аккаунт"}
              </Button>
            </form>
          </SectionShell>

          <SectionShell eyebrow="Why this exists" title="Не трекер привычек">
            <div className={styles.copyBlock}>
              <p>
                ProofForge нужен не для мягкого self-tracking, а для фиксации серьёзной
                цели с обязательным buddy и наблюдаемым статусом движения.
              </p>
              <ul className={styles.ruleList}>
                <li>Цель создаётся вместе с accountability partner.</li>
                <li>Buddy approval остаётся будущим источником правды.</li>
                <li>Dashboard должен показывать систему, а не motivational feed.</li>
              </ul>
            </div>
          </SectionShell>
        </section>
      </main>
    );
  }

  const { dashboard } = screenState;
  const pendingCount = dashboard.summary.pending_buddy_acceptance;

  return (
    <main className={styles.page}>
      <header className={styles.header}>
        <div>
          <span className="eyebrow">Goals dashboard</span>
          <h1>{dashboard.user.display_name}, держите контур под наблюдением</h1>
        </div>
        <StatusPill
          status={pendingCount > 0 ? "pending" : "active"}
          label={pendingCount > 0 ? "Buddy confirmation pending" : "Loop online"}
        />
      </header>

      <section className={styles.topGrid}>
        <SectionShell eyebrow="Progress Health" title="Сигнал движения">
          <div className={styles.progressPanel}>
            <div className={styles.healthValue}>{formatHealthScore(dashboard.summary)}</div>
            <p>
              {dashboard.summary.total_goals === 0
                ? "Контур ещё не запущен. Первый шаг — создать goal и сразу назначить buddy."
                : pendingCount > 0
                  ? `Сейчас ${pendingCount} goal ждёт принятия buddy. Подтверждённый progress начнётся после acceptance.`
                  : "Все текущие goals в активном контуре. Дальше продукт должен перевести это в proof и approval loop."}
            </p>
          </div>
        </SectionShell>

        <SectionShell eyebrow="Buddy Status" title="Статус peer contour">
          <div className={styles.buddyGrid}>
            <StatePanel
              tone={pendingCount > 0 ? "pending" : "success"}
              title={pendingCount > 0 ? "Buddy acceptance pending" : "Buddy loop armed"}
              description={
                pendingCount > 0
                  ? "Инвайт уже создан, но goal ещё не перешёл в активный pact."
                  : "Все текущие buddies приняты и контур готов к check-in flow."
              }
            />
          </div>
        </SectionShell>

        <SectionShell eyebrow="Goal Creation" title="Запустить новый pact">
          <form className={styles.form} onSubmit={handleGoalSubmit}>
            <label className={styles.field}>
              <span>Goal title</span>
              <input name="title" placeholder="Ship first customer-ready slice" required />
            </label>
            <label className={styles.field}>
              <span>Описание</span>
              <textarea
                name="description"
                placeholder="Что именно должно быть доведено до результата"
                rows={4}
              />
            </label>
            <div className={styles.inlineFields}>
              <label className={styles.field}>
                <span>Buddy name</span>
                <input name="buddy_name" placeholder="Serious peer" required />
              </label>
              <label className={styles.field}>
                <span>Buddy email</span>
                <input name="buddy_email" type="email" placeholder="peer@example.com" required />
              </label>
            </div>
            {goalError ? (
              <p className={styles.formError} role="alert">
                {goalError}
              </p>
            ) : null}
            <Button type="submit" disabled={isCreatingGoal}>
              {isCreatingGoal ? "Фиксируем pact..." : "Создать goal"}
            </Button>
          </form>
        </SectionShell>
      </section>

      <section className={styles.middleGrid}>
        <SectionShell eyebrow="Pact Cards" title="Текущие accountability goals">
          {(dashboard.goals ?? []).length === 0 ? (
            <StatePanel
              tone="empty"
              title="Пока нет ни одного goal"
              description="Система уже готова к работе, но первый pact ещё не зафиксирован."
            />
          ) : (
            <div className={styles.goalList}>
              {(dashboard.goals ?? []).map((item) => (
                <GoalCard key={item.goal.id} goal={item} />
              ))}
            </div>
          )}
        </SectionShell>

        <SectionShell eyebrow="Proof Wall" title="Архив proof появится здесь">
          <StatePanel
            tone="empty"
            title="Proof wall пока пуст"
            description="Этот slice довёл систему только до регистрации и goal setup. Следующий слой добавит check-ins и реальные proof artifacts."
          />
        </SectionShell>

        <SectionShell eyebrow="Weekly Poster" title="Недельная сводка">
          <div className={styles.poster}>
            <div className={styles.posterStats}>
              <div>
                <span>Всего goals</span>
                <strong>{padStat(dashboard.summary.total_goals)}</strong>
              </div>
              <div>
                <span>Ждут buddy</span>
                <strong>{padStat(dashboard.summary.pending_buddy_acceptance)}</strong>
              </div>
              <div>
                <span>Активны</span>
                <strong>{padStat(dashboard.summary.active_goals)}</strong>
              </div>
            </div>
            <p>
              {dashboard.summary.total_goals === 0
                ? "Пока weekly poster фиксирует только готовность системы. Следующее наблюдаемое событие — создание первого goal."
                : `Система уже видит ${dashboard.summary.total_goals} зафиксированный goal. Ближайшее узкое место сейчас — перевести pending invites в принятый buddy loop.`}
            </p>
          </div>
        </SectionShell>
      </section>

      <section className={styles.statesGrid}>
        <StatePanel
          tone="loading"
          title="Loading state"
          description="Состояние загрузки остаётся обязательным даже для operational dashboard."
        />
        <StatePanel
          tone="error"
          title="Error state"
          description="Если API недоступен, пользователь должен видеть fault, а не фальшивую успешность."
        />
        <StatePanel
          tone="empty"
          title="Empty state"
          description="Пустой экран здесь недопустим: система должна объяснять, какого артефакта или шага не хватает."
        />
        <StatePanel
          tone="success"
          title="Success state"
          description="Даже успех читается как подтверждённый операционный статус, а не как motivational celebration."
        />
      </section>
    </main>
  );
}

function GoalCard({ goal }: { goal: GoalView }) {
  return (
    <article className={styles.goalCard}>
      <div className={styles.goalCardHeader}>
        <div>
          <strong>{goal.goal.title}</strong>
          <p>{goal.goal.description || "Описание ещё не расширено, но pact уже зафиксирован."}</p>
        </div>
        <StatusPill
          status={goal.goal.status === "active" ? "active" : "pending"}
          label={goal.goal.status === "active" ? "Goal active" : "Waiting for buddy"}
        />
      </div>

      <div className={styles.goalMeta}>
        <div>
          <span>Buddy</span>
          <strong>{goal.buddy.display_name}</strong>
        </div>
        <div>
          <span>Email</span>
          <strong>{goal.buddy.email}</strong>
        </div>
        <div>
          <span>Pact</span>
          <strong>{goal.pact.status}</strong>
        </div>
        <div>
          <span>Invite expires</span>
          <strong>{formatDate(goal.invite.expires_at)}</strong>
        </div>
      </div>
    </article>
  );
}

function padStat(value: number) {
  return value.toString().padStart(2, "0");
}

function formatHealthScore(summary: DashboardResponse["summary"]) {
  if (summary.total_goals === 0) {
    return "0%";
  }

  const activeWeight = summary.active_goals * 100;
  const pendingWeight = summary.pending_buddy_acceptance * 45;
  return `${Math.round((activeWeight + pendingWeight) / summary.total_goals)}%`;
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "short",
  }).format(new Date(value));
}
