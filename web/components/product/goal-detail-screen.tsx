"use client";

import { useCallback, useEffect, useState } from "react";

import { StatePanel } from "@/components/core/state-panel";
import { BuddyGoalScreen } from "@/components/product/buddy-goal-screen";
import { OwnerGoalScreen } from "@/components/product/owner-goal-screen";
import { ApiError, getGoal } from "@/lib/api";
import type { GoalView } from "@/lib/types";

import styles from "./goal-detail-screen.module.css";

type ScreenState =
  | { kind: "loading" }
  | { kind: "unauthenticated" }
  | { kind: "error"; message: string }
  | { kind: "ready"; view: GoalView };

export function GoalDetailScreen({ goalID }: { goalID: number }) {
  const [state, setState] = useState<ScreenState>({ kind: "loading" });

  const load = useCallback(async () => {
    try {
      const data = await getGoal(goalID);
      setState({ kind: "ready", view: data.goal });
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

  if (state.view.role === "buddy") {
    return <BuddyGoalScreen view={state.view} onReload={load} />;
  }
  return <OwnerGoalScreen view={state.view} />;
}
