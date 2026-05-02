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

import styles from "./dashboard-screen.module.css";

type ScreenState =
  | { kind: "loading" }
  | { kind: "unauthenticated" }
  | { kind: "error"; message: string }
  | { kind: "ready"; dashboard: DashboardResponse; primaryCheckIns: CheckIn[] };

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
      const ownerGoals = (dashboard.goals ?? []).filter((g) => g.role === "owner");
      const primary = pickPrimaryGoal(ownerGoals);
      let primaryCheckIns: CheckIn[] = [];
      if (primary && primary.goal.status === "active") {
        try {
          const data = await listCheckIns(primary.goal.id);
          primaryCheckIns = data.check_ins ?? [];
        } catch {
          primaryCheckIns = [];
        }
      }
      setScreenState({ kind: "ready", dashboard, primaryCheckIns });
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

  const { dashboard, primaryCheckIns } = screenState;
  const allGoals = dashboard.goals ?? [];
  const ownerGoals = allGoals.filter((g) => g.role === "owner");
  const buddyGoals = allGoals.filter((g) => g.role === "buddy");
  const primaryGoal = pickPrimaryGoal(ownerGoals);
  const otherGoals = primaryGoal
    ? ownerGoals.filter((goal) => goal.goal.id !== primaryGoal.goal.id)
    : [];

  const checkInStats = computeCheckInStats(primaryCheckIns);
  const recentCheckIns = primaryCheckIns.slice(0, 5);

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
                  <Link className={styles.secondaryLink} href={`/goals/${primaryGoal.goal.id}`}>
                    Открыть цель
                  </Link>
                  <Link className={styles.secondaryLink} href="/goals/new">
                    Новая цель
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

            {primaryGoal.goal.status === "active" ? (
              <StakePanel goalID={primaryGoal.goal.id} role="owner" />
            ) : null}

            <SectionShell eyebrow="История подтверждений" title="Последние сигналы">
              {recentCheckIns.length === 0 ? (
                <StatePanel
                  tone="empty"
                  title={
                    primaryGoal.goal.status === "pending_buddy_acceptance"
                      ? "Подтверждения появятся после принятия приглашения"
                      : "Подтверждений пока нет"
                  }
                  description={
                    primaryGoal.goal.status === "pending_buddy_acceptance"
                      ? "Цикл запустится, когда партнёр примет приглашение."
                      : "Соберите первое подтверждение и отправьте партнёру на проверку."
                  }
                  meta={
                    primaryGoal.goal.status === "active" ? (
                      <Link className={styles.secondaryLink} href={`/goals/${primaryGoal.goal.id}/check-in`}>
                        Подготовить подтверждение
                      </Link>
                    ) : null
                  }
                />
              ) : (
                <ol className={styles.eventList}>
                  {recentCheckIns.map((ci) => (
                    <CheckInEventItem key={ci.id} checkIn={ci} goalID={primaryGoal.goal.id} />
                  ))}
                </ol>
              )}
            </SectionShell>

            <SectionShell eyebrow="Сводка цели" title="Картина движения">
              <div className={styles.poster}>
                <div className={styles.posterStats}>
                  <div>
                    <span>Подтверждено</span>
                    <strong>{formatStat(checkInStats.approved)}</strong>
                  </div>
                  <div>
                    <span>Ждут проверки</span>
                    <strong>{formatStat(checkInStats.submitted)}</strong>
                  </div>
                  <div>
                    <span>Стрик</span>
                    <strong>{formatStat(primaryGoal.goal.current_streak_count)}</strong>
                  </div>
                  <div>
                    <span>На доработке</span>
                    <strong>{formatStat(checkInStats.changesRequested)}</strong>
                  </div>
                </div>
                <p>
                  {primaryGoal.goal.status === "pending_buddy_acceptance"
                    ? "Сводка начнёт отражать движение, как только партнёр примет приглашение."
                    : checkInStats.total === 0
                      ? "Цикл запущен, но подтверждения ещё не приходили — соберите первое."
                      : "Сводка считается по реальным подтверждениям и решениям партнёра."}
                </p>
              </div>
            </SectionShell>

            {otherGoals.length > 0 ? (
              <SectionShell eyebrow="Остальные цели" title="Что ещё идёт параллельно">
                <div className={styles.otherGoals}>
                  {otherGoals.map((goal) => (
                    <Link key={goal.goal.id} href={`/goals/${goal.goal.id}`} className={styles.goalListCard}>
                      <strong>{goal.goal.title}</strong>
                      <p>{goal.goal.description || "Описание пока не заполнено."}</p>
                      <StatusPill
                        status={goal.goal.status === "active" ? "active" : "pending"}
                        label={goal.goal.status === "active" ? "Активна" : "Ждёт принятия"}
                      />
                    </Link>
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

      {buddyGoals.length > 0 ? (
        <section>
          <SectionShell
            eyebrow="Где вы партнёр"
            title={`Цели партнёров (${buddyGoals.length})`}
          >
            <div className={styles.otherGoals}>
              {buddyGoals.map((g) => (
                <Link
                  key={g.goal.id}
                  href={`/goals/${g.goal.id}`}
                  className={styles.goalListCard}
                >
                  <strong>{g.goal.title}</strong>
                  <p>Владелец: {g.owner.display_name}</p>
                  <StatusPill
                    status={g.goal.status === "active" ? "active" : "pending"}
                    label={g.goal.status === "active" ? "В работе" : "Ждёт принятия"}
                  />
                </Link>
              ))}
            </div>
          </SectionShell>
        </section>
      ) : null}
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

function computeCheckInStats(list: CheckIn[]) {
  return list.reduce(
    (acc, ci) => {
      acc.total += 1;
      if (ci.status === "approved") acc.approved += 1;
      else if (ci.status === "submitted") acc.submitted += 1;
      else if (ci.status === "changes_requested") acc.changesRequested += 1;
      else if (ci.status === "rejected") acc.rejected += 1;
      return acc;
    },
    { total: 0, approved: 0, submitted: 0, changesRequested: 0, rejected: 0 },
  );
}

function formatStat(n: number): string {
  return n < 10 ? `0${n}` : String(n);
}

function CheckInEventItem({ checkIn, goalID }: { checkIn: CheckIn; goalID: number }) {
  const date = checkIn.submitted_at ?? checkIn.created_at;
  const status =
    checkIn.status === "approved"
      ? { label: "Подтверждён", tone: "approved" as const, detail: "Партнёр подтвердил движение по цели." }
      : checkIn.status === "rejected"
        ? { label: "Отклонён", tone: "rejected" as const, detail: "Партнёр отклонил это подтверждение." }
        : checkIn.status === "submitted"
          ? { label: "На ревью", tone: "pending" as const, detail: "Партнёр получил материалы и принимает решение." }
          : checkIn.status === "changes_requested"
            ? { label: "Нужны правки", tone: "changes_requested" as const, detail: "Партнёр вернул на доработку — откройте цель и посмотрите комментарий." }
            : { label: "Черновик", tone: "active" as const, detail: "Идёт сборка материалов." };

  const href =
    checkIn.status === "draft" || checkIn.status === "changes_requested"
      ? `/goals/${goalID}/check-in`
      : `/goals/${goalID}`;

  return (
    <li className={styles.eventItem}>
      <div className={styles.eventHeader}>
        <strong>
          <Link href={href}>Чекин #{checkIn.id}</Link>
        </strong>
        <StatusPill status={status.tone} label={status.label} />
      </div>
      <p>{status.detail}</p>
      <span>{formatDateLabel(date, { month: "short", day: "numeric" })}</span>
    </li>
  );
}
