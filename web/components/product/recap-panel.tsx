"use client";

import { useCallback, useEffect, useState } from "react";

import { SectionShell } from "@/components/core/section-shell";
import { StatePanel } from "@/components/core/state-panel";
import { StatusPill } from "@/components/core/status-pill";
import { ApiError, getRecaps } from "@/lib/api";
import type { WeeklyRecap } from "@/lib/types";

import styles from "./recap-panel.module.css";

type ScreenState =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "ready"; recaps: WeeklyRecap[] };

export function RecapPanel({ goalID }: { goalID: number }) {
  const [state, setState] = useState<ScreenState>({ kind: "loading" });

  const load = useCallback(async () => {
    try {
      const data = await getRecaps(goalID);
      setState({ kind: "ready", recaps: data.recaps ?? [] });
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setState({ kind: "error", message: "Сессия не найдена." });
        return;
      }
      setState({
        kind: "error",
        message: error instanceof Error ? error.message : "Не удалось загрузить недельные сводки.",
      });
    }
  }, [goalID]);

  useEffect(() => {
    void load();
  }, [load]);

  if (state.kind === "loading") {
    return (
      <StatePanel
        tone="loading"
        title="Загружаем недельные сводки"
        description="Получаем последние итоги по цели."
      />
    );
  }

  if (state.kind === "error") {
    return <StatePanel tone="error" title="Ошибка" description={state.message} />;
  }

  const { recaps } = state;

  if (recaps.length === 0) {
    return (
      <StatePanel
        tone="empty"
        title="Пока нет недельных сводок"
        description="Недельные сводки появятся после того, как партнёр подтвердит первое отправленное подтверждение."
      />
    );
  }

  return (
    <SectionShell eyebrow="Итоги недели" title={`Недельные сводки (${recaps.length})`}>
      <ol className={styles.list}>
        {recaps.map((recap) => (
          <RecapCard key={recap.id} recap={recap} />
        ))}
      </ol>
    </SectionShell>
  );
}

function RecapCard({ recap }: { recap: WeeklyRecap }) {
  const period = `${formatDate(recap.period_start)} – ${formatDate(recap.period_end)}`;
  const statusTone = recap.status === "done" ? "approved" : recap.status === "failed" ? "rejected" : "pending";
  const statusLabel =
    recap.status === "done"
      ? "Готово"
      : recap.status === "failed"
        ? "Ошибка"
        : recap.status === "generating"
          ? "Готовится"
          : "В очереди";

  return (
    <li className={styles.card}>
      <div className={styles.cardHeader}>
        <span className={styles.period}>{period}</span>
        <StatusPill status={statusTone} label={statusLabel} />
      </div>

      {recap.status === "done" && recap.summary_text && (
        <p className={styles.summary}>{recap.summary_text}</p>
      )}

      {recap.status === "generating" && (
        <p className={styles.meta}>Генерируем сводку…</p>
      )}

      {recap.status === "pending" && (
        <p className={styles.meta}>В очереди на генерацию.</p>
      )}

      {recap.status === "failed" && (
        <p className={styles.metaError}>Не удалось сгенерировать сводку.</p>
      )}

      {recap.model_name && recap.model_name !== "noop" && (
        <p className={styles.model}>Модель: {recap.model_name}</p>
      )}
    </li>
  );
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("ru-RU", { day: "numeric", month: "short" });
}
