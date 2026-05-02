"use client";

import Link from "next/link";
import { useCallback, useEffect, useState, useTransition } from "react";

import { Button } from "@/components/core/button";
import { SectionShell } from "@/components/core/section-shell";
import { StatePanel } from "@/components/core/state-panel";
import { StatusPill } from "@/components/core/status-pill";
import { MilestonePanel } from "@/components/product/milestone-panel";
import { RecapPanel } from "@/components/product/recap-panel";
import { StakePanel } from "@/components/product/stake-panel";
import { acceptGoalInvite, ApiError, getCheckIn, listCheckIns } from "@/lib/api";
import type { CheckIn, GoalView, ReviewRecord } from "@/lib/types";

import styles from "./goal-detail-screen.module.css";

type CheckInWithReview = {
  checkIn: CheckIn;
  latestReview?: ReviewRecord;
};

export function BuddyGoalScreen({
  view,
  onReload,
}: {
  view: GoalView;
  onReload: () => void;
}) {
  const goalID = view.goal.id;
  const goalActive = view.goal.status === "active";

  const [rows, setRows] = useState<CheckInWithReview[]>([]);
  const [acceptError, setAcceptError] = useState<string | null>(null);
  const [isAccepting, startAccept] = useTransition();

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

  function handleAcceptInvite() {
    setAcceptError(null);
    startAccept(async () => {
      try {
        await acceptGoalInvite(goalID);
        onReload();
      } catch (err) {
        if (err instanceof ApiError) {
          if (err.status === 410) {
            setAcceptError("Срок действия приглашения истёк. Попросите владельца цели выслать новое.");
            return;
          }
          if (err.status === 403) {
            setAcceptError("Это приглашение выписано на другой аккаунт.");
            return;
          }
          if (err.status === 409) {
            setAcceptError("Приглашение уже принято — обновите страницу.");
            return;
          }
        }
        setAcceptError(err instanceof Error ? err.message : "Не удалось принять приглашение.");
      }
    });
  }

  const queue = rows.filter((r) => r.checkIn.status === "submitted");
  const history = rows.filter((r) => r.checkIn.status !== "submitted");

  return (
    <main className={styles.page}>
      <Link href="/dashboard" className={styles.backLink}>
        ← К дашборду
      </Link>

      <header className={styles.header}>
        <div className={styles.eyebrow}>
          <span>Вы — партнёр</span>
          <StatusPill
            status={goalActive ? "active" : "pending"}
            label={goalActive ? "Цель в работе" : "Ждёт принятия"}
          />
        </div>
        <h1 className={styles.title}>{view.goal.title}</h1>
        {view.goal.description ? (
          <p className={styles.description}>{view.goal.description}</p>
        ) : null}

        <div className={styles.metaRow}>
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Владелец</span>
            <span className={styles.metaValue}>{view.owner.display_name}</span>
          </div>
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Email владельца</span>
            <span className={styles.metaValue}>{view.owner.email}</span>
          </div>
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Стрик</span>
            <span className={styles.metaValue}>{view.goal.current_streak_count}</span>
          </div>
        </div>
      </header>

      {!goalActive ? (
        <SectionShell eyebrow="Нужно действие" title="Примите приглашение">
          <div className={styles.acceptBlock}>
            <p>
              {view.owner.display_name} приглашает вас стать партнёром по этой цели.
              После принятия вы сможете проверять подтверждения, закрывать контрольные точки
              и фиксировать ставки.
            </p>
            <div className={styles.actionsRow}>
              <Button onClick={handleAcceptInvite} disabled={isAccepting}>
                {isAccepting ? "Принимаем..." : "Принять приглашение"}
              </Button>
            </div>
            {acceptError ? (
              <p className={styles.acceptError} role="alert">
                {acceptError}
              </p>
            ) : null}
          </div>
        </SectionShell>
      ) : (
        <>
          <SectionShell
            eyebrow="Очередь проверки"
            title={queue.length === 0 ? "Подтверждений на проверке нет" : `Ждут вашего решения (${queue.length})`}
          >
            {queue.length === 0 ? (
              <StatePanel
                tone="empty"
                title="Сейчас от вас ничего не ждут"
                description={`${view.owner.display_name} ещё не отправлял новых материалов на проверку.`}
              />
            ) : (
              <ol className={styles.checkInList}>
                {queue.map((row) => (
                  <li key={row.checkIn.id} className={styles.checkInItem}>
                    <div className={styles.checkInItemHeader}>
                      <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
                        <strong className={styles.checkInTitle}>Чекин #{row.checkIn.id}</strong>
                        <StatusPill status="pending" label="На ревью" />
                      </div>
                      <Link
                        href={`/check-ins/${row.checkIn.id}/review`}
                        className={styles.primaryLink}
                      >
                        Проверить →
                      </Link>
                    </div>
                    <span className={styles.checkInDate}>
                      Отправлен {formatDate(row.checkIn.submitted_at ?? row.checkIn.created_at)}
                    </span>
                  </li>
                ))}
              </ol>
            )}
          </SectionShell>

          <MilestonePanel goalID={goalID} role="buddy" />
          <StakePanel goalID={goalID} role="buddy" />

          {history.length > 0 ? (
            <SectionShell eyebrow="История" title={`Ваши решения (${history.length})`}>
              <ol className={styles.checkInList}>
                {history.map((row) => (
                  <BuddyHistoryRow key={row.checkIn.id} row={row} />
                ))}
              </ol>
            </SectionShell>
          ) : null}

          <RecapPanel goalID={goalID} />
        </>
      )}
    </main>
  );
}

function BuddyHistoryRow({ row }: { row: CheckInWithReview }) {
  const { checkIn, latestReview } = row;
  const date = checkIn.submitted_at ?? checkIn.created_at;
  const statusLabel =
    checkIn.status === "approved"
      ? "Подтверждён"
      : checkIn.status === "rejected"
        ? "Отклонён"
        : checkIn.status === "changes_requested"
          ? "Возвращён на доработку"
          : "Черновик";
  const statusTone =
    checkIn.status === "approved"
      ? "approved"
      : checkIn.status === "rejected"
        ? "rejected"
        : checkIn.status === "changes_requested"
          ? "changes_requested"
          : "active";

  return (
    <li className={styles.checkInItem}>
      <div className={styles.checkInItemHeader}>
        <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
          <strong className={styles.checkInTitle}>Чекин #{checkIn.id}</strong>
          <StatusPill status={statusTone} label={statusLabel} />
        </div>
      </div>
      <span className={styles.checkInDate}>{formatDate(date)}</span>
      {latestReview && latestReview.comment ? (
        <div className={styles.checkInComment}>
          <span className={styles.checkInCommentLabel}>Ваш комментарий</span>
          <p className={styles.checkInCommentBody}>{latestReview.comment}</p>
        </div>
      ) : null}
    </li>
  );
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("ru-RU", { day: "numeric", month: "short", year: "numeric" });
}
