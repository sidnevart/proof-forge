"use client";

import Link from "next/link";
import { type FormEvent, useEffect, useState, useTransition } from "react";

import { Button } from "@/components/core/button";
import { SectionShell } from "@/components/core/section-shell";
import { StatePanel } from "@/components/core/state-panel";
import { StatusPill } from "@/components/core/status-pill";
import { proofEvents, weeklyPosterStats } from "@/lib/demo-data";
import { ApiError, getDashboard, loginUser, registerUser } from "@/lib/api";
import { formatDateLabel } from "@/lib/ui-copy";
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
  const [isRegistering, startRegisterTransition] = useTransition();
  const [isLoggingIn, startLoginTransition] = useTransition();

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

  const { dashboard } = screenState;
  const goals = dashboard.goals ?? [];
  const primaryGoal = pickPrimaryGoal(goals);
  const otherGoals = primaryGoal
    ? goals.filter((goal) => goal.goal.id !== primaryGoal.goal.id)
    : [];

  return (
    <main className={styles.page}>
      <header className={styles.header}>
        <div>
          <span className="eyebrow">Операционный центр</span>
          <h1>{dashboard.user.display_name}, держите цель под контролем</h1>
        </div>
        <StatusPill
          status={primaryGoal?.goal.status === "pending_buddy_acceptance" ? "pending" : "active"}
          label={
            primaryGoal?.goal.status === "pending_buddy_acceptance"
              ? "Ждём ответа партнёра"
              : "Цель в работе"
          }
        />
      </header>

      {primaryGoal ? (
        <>
          <section className={styles.heroGrid}>
            <SectionShell eyebrow="Следующий шаг" title={getNextStep(primaryGoal).title}>
              <div className={styles.nextStepBlock}>
                <strong className={styles.focusGoal}>{primaryGoal.goal.title}</strong>
                <p>{getNextStep(primaryGoal).description}</p>
                <div className={styles.actionRow}>
                  <Link className={styles.primaryLink} href={getNextStep(primaryGoal).href}>
                    {getNextStep(primaryGoal).action}
                  </Link>
                  <Link className={styles.secondaryLink} href="/goals/new">
                    Добавить ещё одну цель
                  </Link>
                </div>
              </div>
            </SectionShell>

            <SectionShell eyebrow="Состояние контура" title="Что происходит сейчас">
              <dl className={styles.snapshotList}>
                <div>
                  <dt>Главная цель</dt>
                  <dd>{primaryGoal.goal.title}</dd>
                </div>
                <div>
                  <dt>Партнёр</dt>
                  <dd>{primaryGoal.buddy.display_name}</dd>
                </div>
                <div>
                  <dt>Статус цели</dt>
                  <dd>{primaryGoal.goal.status === "active" ? "Активна" : "Ждёт принятия"}</dd>
                </div>
                <div>
                  <dt>Следующая точка</dt>
                  <dd>{getNextStep(primaryGoal).shortLabel}</dd>
                </div>
              </dl>
            </SectionShell>
          </section>

          <section className={styles.surfaceGrid}>
            <SectionShell eyebrow="Главная цель" title="Карточка цели">
              <GoalCard goal={primaryGoal} />
            </SectionShell>

            <SectionShell eyebrow="Состояние прогресса" title="Движение по цели">
              <div className={styles.progressPanel}>
                <div className={styles.healthValue}>{formatHealthScore(dashboard.summary)}</div>
                <p>{getProgressText(primaryGoal, dashboard.summary)}</p>
              </div>
            </SectionShell>

            <SectionShell eyebrow="Статус партнёра" title="Кто подтверждает движение">
              <StatePanel
                tone={primaryGoal.goal.status === "pending_buddy_acceptance" ? "pending" : "success"}
                title={
                  primaryGoal.goal.status === "pending_buddy_acceptance"
                    ? "Партнёр ещё не принял приглашение"
                    : "Партнёр готов к проверке"
                }
                description={
                  primaryGoal.goal.status === "pending_buddy_acceptance"
                    ? `Приглашение активно до ${formatShortDate(primaryGoal.invite.expires_at)}. Пока партнёр не подтвердит участие, прогресс по цели не может перейти в рабочий цикл.`
                    : "Подтверждения можно отправлять на проверку. Решение по прогрессу принимает партнёр."
                }
              />
            </SectionShell>

            <SectionShell eyebrow="История подтверждений" title="Последние сигналы">
              <ol className={styles.eventList}>
                {proofEvents.map((event) => (
                  <li className={styles.eventItem} key={`${event.title}-${event.time}`}>
                    <div className={styles.eventHeader}>
                      <strong>{event.title}</strong>
                      <StatusPill status={event.status} />
                    </div>
                    <p>{event.detail}</p>
                    <span>{event.time}</span>
                  </li>
                ))}
              </ol>
            </SectionShell>

            <SectionShell eyebrow="Еженедельная сводка" title="Недельная сводка">
              <div className={styles.poster}>
                <div className={styles.posterStats}>
                  {weeklyPosterStats.map((item) => (
                    <div key={item.label}>
                      <span>{item.label}</span>
                      <strong>{item.value}</strong>
                    </div>
                  ))}
                </div>
                <p>
                  {primaryGoal.goal.status === "pending_buddy_acceptance"
                    ? "Пока неделе не хватает принятого приглашения: как только партнёр войдёт в цикл, сводка начнёт отражать подтверждённое движение."
                    : "Неделя читается через подтверждения и решения по ним. Здесь собирается не мотивация, а фактический ритм выполнения цели."}
                </p>
              </div>
            </SectionShell>

            {otherGoals.length > 0 ? (
              <SectionShell eyebrow="Остальные цели" title="Что ещё идёт параллельно">
                <div className={styles.otherGoals}>
                  {otherGoals.map((goal) => (
                    <article className={styles.goalListCard} key={goal.goal.id}>
                      <strong>{goal.goal.title}</strong>
                      <p>{goal.goal.description || "Описание пока не заполнено."}</p>
                      <StatusPill
                        status={goal.goal.status === "active" ? "active" : "pending"}
                        label={goal.goal.status === "active" ? "Активна" : "Ждёт принятия"}
                      />
                    </article>
                  ))}
                </div>
              </SectionShell>
            ) : null}
          </section>
        </>
      ) : (
        <>
          <section className={styles.heroGrid}>
            <SectionShell eyebrow="Следующий шаг" title="Создайте первую цель">
              <div className={styles.nextStepBlock}>
                <strong className={styles.focusGoal}>Рабочий контур ещё не запущен</strong>
                <p>
                  Пока у вас нет ни одной цели, поэтому главный экран не может вести вас
                  по циклу. Начните с формулировки цели и приглашения партнёра.
                </p>
                <div className={styles.actionRow}>
                  <Link className={styles.primaryLink} href="/goals/new">
                    Создать первую цель
                  </Link>
                </div>
              </div>
            </SectionShell>

            <SectionShell eyebrow="Что появится дальше" title="Будущие поверхности">
              <div className={styles.snapshotList}>
                <div>
                  <dt>Главная цель</dt>
                  <dd>Станет центром экрана</dd>
                </div>
                <div>
                  <dt>Партнёр</dt>
                  <dd>Получит приглашение к участию</dd>
                </div>
                <div>
                  <dt>Подтверждения</dt>
                  <dd>Появятся в истории сигналов</dd>
                </div>
                <div>
                  <dt>Сводка</dt>
                  <dd>Соберёт картину недели</dd>
                </div>
              </div>
            </SectionShell>
          </section>

          <section className={styles.surfaceGrid}>
            <SectionShell eyebrow="Главная цель" title="Цель появится здесь">
              <StatePanel
                tone="empty"
                title="Пока ничего не зафиксировано"
                description="После создания цели здесь появится карточка с описанием, статусом и данными партнёра."
              />
            </SectionShell>

            <SectionShell eyebrow="История подтверждений" title="Журнал пока пуст">
              <StatePanel
                tone="empty"
                title="Нет подтверждений"
                description="Как только по цели появятся первые материалы, они будут складываться в этот журнал."
              />
            </SectionShell>
          </section>
        </>
      )}
    </main>
  );
}

function GoalCard({ goal }: { goal: GoalView }) {
  return (
    <article className={styles.goalCard}>
      <div className={styles.goalCardHeader}>
        <div>
          <strong>{goal.goal.title}</strong>
          <p>{goal.goal.description || "Описание пока не заполнено."}</p>
        </div>
        <StatusPill
          status={goal.goal.status === "active" ? "active" : "pending"}
          label={goal.goal.status === "active" ? "Активна" : "Ждёт принятия"}
        />
      </div>

      <div className={styles.goalMeta}>
        <div>
          <span>Партнёр</span>
          <strong>{goal.buddy.display_name}</strong>
        </div>
        <div>
          <span>Почта</span>
          <strong>{goal.buddy.email}</strong>
        </div>
        <div>
          <span>Приглашение действует до</span>
          <strong>{formatShortDate(goal.invite.expires_at)}</strong>
        </div>
        <div>
          <span>Последнее изменение</span>
          <strong>{formatShortDate(goal.goal.updated_at)}</strong>
        </div>
      </div>
    </article>
  );
}

function pickPrimaryGoal(goals: GoalView[]) {
  return goals.find((goal) => goal.goal.status === "pending_buddy_acceptance") ?? goals[0];
}

function getNextStep(goal: GoalView) {
  if (goal.goal.status === "pending_buddy_acceptance") {
    return {
      title: "Дождитесь ответа партнёра",
      description:
        "Приглашение уже отправлено. Как только партнёр примет участие, цель перейдёт в рабочий цикл, и вы сможете отправлять подтверждения прогресса.",
      action: "Создать ещё одну цель",
      href: "/goals/new",
      shortLabel: "Ждём принятия приглашения",
    };
  }

  return {
    title: "Подготовьте новое подтверждение",
    description:
      "Следующая задача — собрать материалы по цели и отправить их партнёру на проверку.",
    action: "Перейти к подтверждению",
    href: `/goals/${goal.goal.id}/check-in`,
    shortLabel: "Собрать подтверждение",
  };
}

function getProgressText(goal: GoalView, summary: DashboardResponse["summary"]) {
  if (goal.goal.status === "pending_buddy_acceptance") {
    return `Сейчас в системе ${summary.pending_buddy_acceptance} ${
      summary.pending_buddy_acceptance === 1 ? "цель ждёт" : "цели ждут"
    } ответа партнёра. Подтверждённый прогресс начнётся после принятия приглашения.`;
  }

  return "Цель находится в рабочем цикле. Следующий шаг — собрать подтверждение и отправить его партнёру на проверку.";
}

function formatHealthScore(summary: DashboardResponse["summary"]) {
  if (summary.total_goals === 0) {
    return "0%";
  }

  const activeWeight = summary.active_goals * 100;
  const pendingWeight = summary.pending_buddy_acceptance * 45;
  return `${Math.round((activeWeight + pendingWeight) / summary.total_goals)}%`;
}

function formatShortDate(value: string) {
  return formatDateLabel(value, { month: "short" });
}
