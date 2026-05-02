"use client";

import { type FormEvent, useCallback, useEffect, useState, useTransition } from "react";

import { Button } from "@/components/core/button";
import { SectionShell } from "@/components/core/section-shell";
import { StatePanel } from "@/components/core/state-panel";
import {
  ApiError,
  completeMilestone,
  createMilestone,
  deleteMilestone,
  listMilestones,
  reopenMilestone,
} from "@/lib/api";
import { cn } from "@/lib/cn";
import type { GoalRole, Milestone } from "@/lib/types";

import styles from "./milestone-panel.module.css";

type ScreenState =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "ready"; milestones: Milestone[] };

type Props = {
  goalID: number;
  role: GoalRole;
};

export function MilestonePanel({ goalID, role }: Props) {
  const [state, setState] = useState<ScreenState>({ kind: "loading" });
  const [title, setTitle] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isCreating, startCreate] = useTransition();

  const isOwner = role === "owner";
  const isBuddy = role === "buddy";

  const load = useCallback(async () => {
    try {
      const data = await listMilestones(goalID);
      setState({ kind: "ready", milestones: data.milestones ?? [] });
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setState({ kind: "error", message: "Сессия не найдена." });
        return;
      }
      setState({
        kind: "error",
        message: err instanceof Error ? err.message : "Не удалось загрузить контрольные точки.",
      });
    }
  }, [goalID]);

  useEffect(() => {
    void load();
  }, [load]);

  function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    const t = title.trim();
    if (!t) return;

    startCreate(async () => {
      try {
        await createMilestone(goalID, t, "");
        setTitle("");
        await load();
      } catch (err) {
        setError(err instanceof Error ? err.message : "Не удалось создать.");
      }
    });
  }

  if (state.kind === "loading") {
    return <StatePanel tone="loading" title="Загружаем контрольные точки" description="" />;
  }

  if (state.kind === "error") {
    return <StatePanel tone="error" title="Ошибка" description={state.message} />;
  }

  const { milestones } = state;
  const completed = milestones.filter((m) => m.status === "completed").length;
  const total = milestones.length;

  return (
    <SectionShell eyebrow="Контрольные точки" title={total > 0 ? `Прогресс: ${completed} / ${total}` : "Контрольные точки"}>
      {total > 0 ? (
        <div className={styles.progress}>
          <div className={styles.progressBar}>
            <div
              className={styles.progressFill}
              style={{ width: `${total === 0 ? 0 : (completed / total) * 100}%` }}
            />
          </div>
          <span>{Math.round(total === 0 ? 0 : (completed / total) * 100)}%</span>
        </div>
      ) : (
        <p className={styles.empty}>
          {isOwner
            ? "Разбейте цель на 3–5 проверяемых контрольных точек."
            : "Партнёр пока не разбил цель на контрольные точки."}
        </p>
      )}

      {milestones.length > 0 ? (
        <ol className={styles.list}>
          {milestones.map((m) => (
            <MilestoneCard
              key={m.id}
              milestone={m}
              isOwner={isOwner}
              isBuddy={isBuddy}
              onUpdate={load}
            />
          ))}
        </ol>
      ) : null}

      {isOwner ? (
        <form className={styles.addForm} onSubmit={handleCreate}>
          <div className={styles.addRow}>
            <input
              className={styles.addInput}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Например: «PR с корутинами в публичный репо»"
            />
            <Button type="submit" disabled={isCreating || !title.trim()}>
              {isCreating ? "..." : "Добавить"}
            </Button>
          </div>
          {error ? <p className={styles.error} role="alert">{error}</p> : null}
        </form>
      ) : null}
    </SectionShell>
  );
}

function MilestoneCard({
  milestone,
  isOwner,
  isBuddy,
  onUpdate,
}: {
  milestone: Milestone;
  isOwner: boolean;
  isBuddy: boolean;
  onUpdate: () => Promise<void>;
}) {
  const [error, setError] = useState<string | null>(null);
  const [isActing, startAct] = useTransition();

  const isCompleted = milestone.status === "completed";

  function handleComplete() {
    setError(null);
    startAct(async () => {
      try {
        await completeMilestone(milestone.id);
        await onUpdate();
      } catch (err) {
        setError(err instanceof Error ? err.message : "Не удалось закрыть.");
      }
    });
  }

  function handleReopen() {
    setError(null);
    startAct(async () => {
      try {
        await reopenMilestone(milestone.id);
        await onUpdate();
      } catch (err) {
        setError(err instanceof Error ? err.message : "Не удалось переоткрыть.");
      }
    });
  }

  function handleDelete() {
    setError(null);
    startAct(async () => {
      try {
        await deleteMilestone(milestone.id);
        await onUpdate();
      } catch (err) {
        setError(err instanceof Error ? err.message : "Не удалось удалить.");
      }
    });
  }

  return (
    <li className={cn(styles.card, isCompleted && styles.completed)}>
      <div className={styles.cardHeader}>
        <p className={cn(styles.title, isCompleted && styles.titleCompleted)}>
          {milestone.title}
        </p>
      </div>

      {milestone.description ? (
        <p className={styles.description}>{milestone.description}</p>
      ) : null}

      {isCompleted && milestone.completed_at ? (
        <p className={styles.completedMeta}>Закрыта {formatDate(milestone.completed_at)}</p>
      ) : null}

      <div className={styles.actions}>
        {isBuddy && !isCompleted ? (
          <Button variant="secondary" onClick={handleComplete} disabled={isActing}>
            {isActing ? "..." : "Закрыть"}
          </Button>
        ) : null}
        {isBuddy && isCompleted ? (
          <Button variant="ghost" onClick={handleReopen} disabled={isActing}>
            {isActing ? "..." : "Переоткрыть"}
          </Button>
        ) : null}
        {isOwner && !isCompleted ? (
          <Button variant="ghost" onClick={handleDelete} disabled={isActing}>
            {isActing ? "..." : "Удалить"}
          </Button>
        ) : null}
      </div>

      {error ? <p className={styles.error} role="alert">{error}</p> : null}
    </li>
  );
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("ru-RU", { day: "numeric", month: "short", year: "numeric" });
}
