"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { SectionShell } from "@/components/core/section-shell";
import { StatePanel } from "@/components/core/state-panel";
import { StatusPill } from "@/components/core/status-pill";
import { MilestonePanel } from "@/components/product/milestone-panel";
import { RecapPanel } from "@/components/product/recap-panel";
import { StakePanel } from "@/components/product/stake-panel";
import { ApiError, getGoal, listCheckIns } from "@/lib/api";
import type { CheckIn, GoalView } from "@/lib/types";

import styles from "./goal-detail-screen.module.css";

type ScreenState =
  | { kind: "loading" }
  | { kind: "unauthenticated" }
  | { kind: "error"; message: string }
  | { kind: "ready"; view: GoalView; checkIns: CheckIn[] };

export function GoalDetailScreen({ goalID }: { goalID: number }) {
  const [state, setState] = useState<ScreenState>({ kind: "loading" });

  const load = useCallback(async () => {
    try {
      const [goalData, checkInsData] = await Promise.all([
        getGoal(goalID),
        listCheckIns(goalID).catch(() => ({ check_ins: [] as CheckIn[] | null })),
      ]);
      setState({
        kind: "ready",
        view: goalData.goal,
        checkIns: checkInsData.check_ins ?? [],
      });
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setState({ kind: "unauthenticated" });
        return;
      }
      if (err instanceof ApiError && err.status === 404) {
        setState({ kind: "error", message: "Цель не найдена или у вас нет к ней доступа." });
        return;
      }
      setState({
        kind: "error",
        message: err instanceof Error ? err.message : "Не удалось загрузить цель.",
      });
    }
  }, [goalID]);

  useEffect(() => {
    void load();
  }, [load]);

  if (state.kind === "loading") {
    return (
      <main className={styles.page}>
        <StatePanel tone="loading" title="Загружаем цель" description="Собираем все данные." />
      </main>
    );
  }

  if (state.kind === "unauthenticated") {
    return (
      <main className={styles.page}>
        <StatePanel
          tone="error"
          title="Нужна сессия"
          description="Войдите, чтобы открыть страницу цели."
        />
      </main>
    );
  }

  if (state.kind === "error") {
    return (
      <main className={styles.page}>
        <StatePanel tone="error" title="Ошибка" description={state.message} />
      </main>
    );
  }

  const { view, checkIns } = state;
  const counterpart = view.role === "owner" ? view.buddy : view.owner;
  const goalActive = view.goal.status === "active";

  const submittedCount = checkIns.filter((ci) => ci.status === "submitted").length;
  const recentCheckIns = checkIns.slice(0, 5);

  return (
    <main className={styles.page}>
      <Link href="/dashboard" className={styles.backLink}>
        ← К дашборду
      </Link>

      <header className={styles.header}>
        <div className={styles.eyebrow}>
          <span>{view.role === "owner" ? "Ваша цель" : "Вы — партнёр"}</span>
          <StatusPill
            status={goalActive ? "active" : "pending"}
            label={goalActive ? "В работе" : "Ждёт принятия"}
          />
        </div>
        <h1 className={styles.title}>{view.goal.title}</h1>
        {view.goal.description ? (
          <p className={styles.description}>{view.goal.description}</p>
        ) : null}

        <div className={styles.metaRow}>
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>
              {view.role === "owner" ? "Партнёр" : "Владелец"}
            </span>
            <span className={styles.metaValue}>{counterpart.display_name}</span>
          </div>
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Стрик</span>
            <span className={styles.metaValue}>{view.goal.current_streak_count}</span>
          </div>
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Создана</span>
            <span className={styles.metaValue}>{formatDate(view.goal.created_at)}</span>
          </div>
        </div>

        {goalActive ? (
          <div className={styles.actionsRow}>
            {view.role === "owner" ? (
              <Link className={styles.primaryLink} href={`/goals/${goalID}/check-in`}>
                Создать чекин
              </Link>
            ) : null}
            {view.role === "buddy" && submittedCount > 0 ? (
              <span className={styles.secondaryLink}>
                {submittedCount} {submittedCount === 1 ? "чекин на ревью" : "чекинов на ревью"}
              </span>
            ) : null}
          </div>
        ) : null}
      </header>

      {goalActive ? (
        <>
          <MilestonePanel goalID={goalID} role={view.role} />
          <StakePanel goalID={goalID} role={view.role} />
        </>
      ) : (
        <StatePanel
          tone="pending"
          title="Цель ещё не активна"
          description={
            view.role === "owner"
              ? "Партнёр пока не принял приглашение. Контрольные точки и ставки появятся после активации."
              : "Примите приглашение чтобы цель стала активной."
          }
        />
      )}

      {goalActive && checkIns.length > 0 ? (
        <SectionShell eyebrow="История" title={`Чекины (${checkIns.length})`}>
          <ol className={styles.checkInList}>
            {recentCheckIns.map((ci) => (
              <CheckInRow key={ci.id} checkIn={ci} role={view.role} />
            ))}
          </ol>
        </SectionShell>
      ) : null}

      {goalActive ? <RecapPanel goalID={goalID} /> : null}
    </main>
  );
}

function CheckInRow({ checkIn, role }: { checkIn: CheckIn; role: "owner" | "buddy" }) {
  const date = checkIn.submitted_at ?? checkIn.created_at;
  const statusLabel =
    checkIn.status === "approved"
      ? "Подтверждён"
      : checkIn.status === "rejected"
        ? "Отклонён"
        : checkIn.status === "submitted"
          ? "На ревью"
          : checkIn.status === "changes_requested"
            ? "Нужны правки"
            : "Черновик";
  const statusTone =
    checkIn.status === "approved"
      ? "approved"
      : checkIn.status === "rejected"
        ? "rejected"
        : checkIn.status === "submitted"
          ? "pending"
          : checkIn.status === "changes_requested"
            ? "changes_requested"
            : "active";

  return (
    <li className={styles.checkInItem}>
      <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
        <span className={styles.checkInDate}>{formatDate(date)}</span>
        <StatusPill status={statusTone} label={statusLabel} />
      </div>
      {role === "buddy" && checkIn.status === "submitted" ? (
        <Link href={`/check-ins/${checkIn.id}/review`}>Проверить →</Link>
      ) : null}
    </li>
  );
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("ru-RU", { day: "numeric", month: "short", year: "numeric" });
}
