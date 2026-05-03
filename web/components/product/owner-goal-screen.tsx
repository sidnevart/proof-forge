"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState, useTransition } from "react";

import { Button } from "@/components/core/button";
import { SectionShell } from "@/components/core/section-shell";
import { StatePanel } from "@/components/core/state-panel";
import { StatusPill } from "@/components/core/status-pill";
import { MilestonePanel } from "@/components/product/milestone-panel";
import { RecapPanel } from "@/components/product/recap-panel";
import { StakePanel } from "@/components/product/stake-panel";
import {
  deleteCheckIn,
  deleteGoal,
  getCheckIn,
  listCheckIns,
  setGoalDeadline,
} from "@/lib/api";
import type { CheckIn, GoalView, ReviewRecord } from "@/lib/types";

import styles from "./goal-detail-screen.module.css";

type CheckInWithReview = {
  checkIn: CheckIn;
  latestReview?: ReviewRecord;
};

export function OwnerGoalScreen({ view }: { view: GoalView }) {
  const router = useRouter();
  const goalID = view.goal.id;
  const goalActive = view.goal.status === "active";

  const [rows, setRows] = useState<CheckInWithReview[]>([]);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [isDeletingGoal, startDeleteGoal] = useTransition();
  const [deletingCheckInID, setDeletingCheckInID] = useState<number | null>(null);
  const [deadline, setDeadline] = useState<string | null>(view.goal.deadline_at ?? null);
  const [editingDeadline, setEditingDeadline] = useState(false);
  const [deadlineDraft, setDeadlineDraft] = useState<string>(view.goal.deadline_at ?? "");
  const [deadlineError, setDeadlineError] = useState<string | null>(null);
  const [isSavingDeadline, startSaveDeadline] = useTransition();

  const loadCheckIns = useCallback(async () => {
    if (!goalActive) return;
    try {
      const data = await listCheckIns(goalID);
      const list = data.check_ins ?? [];
      const recent = list.slice(0, 5);
      const detailed = await Promise.all(
        recent.map(async (ci) => {
          if (ci.status === "draft" || ci.status === "submitted") {
            return { checkIn: ci } as CheckInWithReview;
          }
          try {
            const detail = await getCheckIn(ci.id);
            const reviews = detail.reviews ?? [];
            return {
              checkIn: detail.check_in,
              latestReview: reviews.length > 0 ? reviews[reviews.length - 1] : undefined,
            };
          } catch {
            return { checkIn: ci } as CheckInWithReview;
          }
        }),
      );
      setRows([...detailed, ...list.slice(5).map((ci) => ({ checkIn: ci }))]);
    } catch {
      setRows([]);
    }
  }, [goalActive, goalID]);

  useEffect(() => {
    void loadCheckIns();
  }, [loadCheckIns]);

  function handleDeleteGoal() {
    if (typeof window !== "undefined") {
      const ok = window.confirm(
        `Удалить цель «${view.goal.title}»? Это действие необратимо: вместе с целью пропадут чекины, контрольные точки, ставки и сводки.`,
      );
      if (!ok) return;
    }
    setDeleteError(null);
    startDeleteGoal(async () => {
      try {
        await deleteGoal(goalID);
        router.push("/dashboard");
      } catch (err) {
        setDeleteError(err instanceof Error ? err.message : "Не удалось удалить цель.");
      }
    });
  }

  function saveDeadline(value: string | null) {
    setDeadlineError(null);
    startSaveDeadline(async () => {
      try {
        await setGoalDeadline(goalID, value);
        setDeadline(value);
        setDeadlineDraft(value ?? "");
        setEditingDeadline(false);
      } catch (err) {
        setDeadlineError(err instanceof Error ? err.message : "Не удалось сохранить дедлайн.");
      }
    });
  }

  async function handleDeleteCheckIn(checkInID: number) {
    if (typeof window !== "undefined") {
      const ok = window.confirm("Удалить этот чекин? Все материалы пропадут.");
      if (!ok) return;
    }
    setDeleteError(null);
    setDeletingCheckInID(checkInID);
    try {
      await deleteCheckIn(checkInID);
      await loadCheckIns();
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : "Не удалось удалить чекин.");
    } finally {
      setDeletingCheckInID(null);
    }
  }

  const needsAttention = rows.find(
    (r) => r.checkIn.status === "changes_requested" || r.checkIn.status === "draft",
  );

  return (
    <main className={styles.page}>
      <Link href="/dashboard" className={styles.backLink}>
        ← К дашборду
      </Link>

      <header className={styles.header}>
        <div className={styles.eyebrow}>
          <span>Ваша цель</span>
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
            <span className={styles.metaLabel}>Партнёр</span>
            <span className={styles.metaValue}>{view.buddy.display_name}</span>
          </div>
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Email партнёра</span>
            <span className={styles.metaValue}>{view.buddy.email}</span>
          </div>
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Стрик</span>
            <span className={styles.metaValue}>{view.goal.current_streak_count}</span>
          </div>
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Дедлайн</span>
            {editingDeadline ? (
              <span style={{ display: "flex", gap: 6, alignItems: "center" }}>
                <input
                  type="date"
                  value={deadlineDraft}
                  onChange={(e) => setDeadlineDraft(e.target.value)}
                  className={styles.deadlineInput}
                  disabled={isSavingDeadline}
                />
                <button
                  type="button"
                  className={styles.deadlineAction}
                  onClick={() => saveDeadline(deadlineDraft || null)}
                  disabled={isSavingDeadline}
                >
                  Сохранить
                </button>
                <button
                  type="button"
                  className={styles.deadlineAction}
                  onClick={() => {
                    setEditingDeadline(false);
                    setDeadlineDraft(deadline ?? "");
                    setDeadlineError(null);
                  }}
                  disabled={isSavingDeadline}
                >
                  Отмена
                </button>
              </span>
            ) : (
              <span style={{ display: "flex", gap: 8, alignItems: "center" }}>
                <span className={styles.metaValue}>{deadline ? formatDateOnly(deadline) : "—"}</span>
                <button
                  type="button"
                  className={styles.deadlineAction}
                  onClick={() => setEditingDeadline(true)}
                >
                  {deadline ? "изменить" : "поставить"}
                </button>
                {deadline ? (
                  <button
                    type="button"
                    className={styles.deadlineAction}
                    onClick={() => saveDeadline(null)}
                    disabled={isSavingDeadline}
                  >
                    убрать
                  </button>
                ) : null}
              </span>
            )}
          </div>
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Создана</span>
            <span className={styles.metaValue}>{formatDate(view.goal.created_at)}</span>
          </div>
        </div>
        {deadlineError ? (
          <p className={styles.deleteError} role="alert">
            {deadlineError}
          </p>
        ) : null}

        <div className={styles.actionsRow}>
          {goalActive ? (
            <Link className={styles.primaryLink} href={`/goals/${goalID}/check-in`}>
              {needsAttention ? "Доработать чекин" : "Создать чекин"}
            </Link>
          ) : null}
          <Button variant="ghost" onClick={handleDeleteGoal} disabled={isDeletingGoal}>
            {isDeletingGoal ? "Удаляем..." : "Удалить цель"}
          </Button>
        </div>
        {deleteError ? (
          <p className={styles.deleteError} role="alert">
            {deleteError}
          </p>
        ) : null}
      </header>

      {goalActive ? (
        <>
          <MilestonePanel goalID={goalID} role="owner" />
          <StakePanel goalID={goalID} role="owner" />

          {rows.length > 0 ? (
            <SectionShell eyebrow="История" title={`Ваши чекины (${rows.length})`}>
              <ol className={styles.checkInList}>
                {rows.slice(0, 5).map((row) => (
                  <OwnerHistoryRow
                    key={row.checkIn.id}
                    row={row}
                    goalID={goalID}
                    onDelete={handleDeleteCheckIn}
                    isDeleting={deletingCheckInID === row.checkIn.id}
                  />
                ))}
              </ol>
            </SectionShell>
          ) : (
            <SectionShell eyebrow="История" title="Чекинов пока нет">
              <StatePanel
                tone="empty"
                title="Соберите первый чекин"
                description="Подготовьте материалы и отправьте партнёру на проверку."
                meta={
                  <Link className={styles.primaryLink} href={`/goals/${goalID}/check-in`}>
                    Подготовить чекин
                  </Link>
                }
              />
            </SectionShell>
          )}

          <RecapPanel goalID={goalID} />
        </>
      ) : (
        <StatePanel
          tone="pending"
          title="Ждём ответа партнёра"
          description={`Приглашение отправлено на ${view.buddy.email}. Контрольные точки и ставки появятся после того, как партнёр примет участие.`}
        />
      )}
    </main>
  );
}

function OwnerHistoryRow({
  row,
  goalID,
  onDelete,
  isDeleting,
}: {
  row: CheckInWithReview;
  goalID: number;
  onDelete: (checkInID: number) => void;
  isDeleting: boolean;
}) {
  const { checkIn, latestReview } = row;
  const canDelete = checkIn.status === "draft" || checkIn.status === "changes_requested";
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

  const cta =
    checkIn.status === "changes_requested"
      ? { label: "Доработать", href: `/goals/${goalID}/check-in` }
      : checkIn.status === "draft"
        ? { label: "Продолжить", href: `/goals/${goalID}/check-in` }
        : null;

  const commentLabel =
    latestReview?.decision === "approved"
      ? "Комментарий партнёра при подтверждении"
      : latestReview?.decision === "rejected"
        ? "Партнёр отклонил"
        : latestReview?.decision === "changes_requested"
          ? "Партнёр просит доработать"
          : "Комментарий партнёра";

  return (
    <li className={styles.checkInItem}>
      <div className={styles.checkInItemHeader}>
        <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
          <strong className={styles.checkInTitle}>Чекин #{checkIn.id}</strong>
          <StatusPill status={statusTone} label={statusLabel} />
        </div>
        <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
          {cta ? (
            <Link href={cta.href} className={styles.checkInOpenLink}>
              {cta.label} →
            </Link>
          ) : null}
          {canDelete ? (
            <button
              type="button"
              className={styles.checkInDelete}
              onClick={() => onDelete(checkIn.id)}
              disabled={isDeleting}
            >
              {isDeleting ? "Удаляем..." : "Удалить"}
            </button>
          ) : null}
        </div>
      </div>
      <span className={styles.checkInDate}>{formatDate(date)}</span>
      {latestReview && latestReview.comment ? (
        <div className={styles.checkInComment}>
          <span className={styles.checkInCommentLabel}>{commentLabel}</span>
          <p className={styles.checkInCommentBody}>{latestReview.comment}</p>
        </div>
      ) : null}
    </li>
  );
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("ru-RU", { day: "numeric", month: "short", year: "numeric" });
}

function formatDateOnly(iso: string): string {
  // Parse YYYY-MM-DD as local date so we don't shift by timezone.
  const [y, m, d] = iso.split("-").map((part) => parseInt(part, 10));
  if (!y || !m || !d) return iso;
  return new Date(y, m - 1, d).toLocaleDateString("ru-RU", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}
