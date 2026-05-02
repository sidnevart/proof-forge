"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { SectionShell } from "@/components/core/section-shell";
import { StatePanel } from "@/components/core/state-panel";
import { StatusPill } from "@/components/core/status-pill";
import { MilestonePanel } from "@/components/product/milestone-panel";
import { RecapPanel } from "@/components/product/recap-panel";
import { StakePanel } from "@/components/product/stake-panel";
import { getCheckIn, listCheckIns } from "@/lib/api";
import type { CheckIn, GoalView, ReviewRecord } from "@/lib/types";

import styles from "./goal-detail-screen.module.css";

type CheckInWithReview = {
  checkIn: CheckIn;
  latestReview?: ReviewRecord;
};

export function OwnerGoalScreen({ view }: { view: GoalView }) {
  const goalID = view.goal.id;
  const goalActive = view.goal.status === "active";

  const [rows, setRows] = useState<CheckInWithReview[]>([]);

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
            <span className={styles.metaLabel}>Создана</span>
            <span className={styles.metaValue}>{formatDate(view.goal.created_at)}</span>
          </div>
        </div>

        {goalActive ? (
          <div className={styles.actionsRow}>
            <Link className={styles.primaryLink} href={`/goals/${goalID}/check-in`}>
              {needsAttention ? "Доработать чекин" : "Создать чекин"}
            </Link>
          </div>
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
                  <OwnerHistoryRow key={row.checkIn.id} row={row} goalID={goalID} />
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

function OwnerHistoryRow({ row, goalID }: { row: CheckInWithReview; goalID: number }) {
  const { checkIn, latestReview } = row;
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
        {cta ? (
          <Link href={cta.href} className={styles.checkInOpenLink}>
            {cta.label} →
          </Link>
        ) : null}
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
