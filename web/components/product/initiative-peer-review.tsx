"use client";

import { useCallback, useEffect, useState, useTransition } from "react";

import { ApiError, approveInitiativeProof, listPendingProofs } from "@/lib/api";
import type { PendingProof } from "@/lib/types";

import styles from "./initiative-peer-review.module.css";

type Props = {
  initiativeId: number;
};

type State =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "ready"; proofs: PendingProof[] };

export function InitiativePeerReview({ initiativeId }: Props) {
  const [state, setState] = useState<State>({ kind: "loading" });
  const [comments, setComments] = useState<Record<number, string>>({});
  const [approved, setApproved] = useState<Set<number>>(new Set());
  const [isPending, startTransition] = useTransition();

  const load = useCallback(async () => {
    try {
      const proofs = await listPendingProofs(initiativeId);
      setState({ kind: "ready", proofs });
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) {
        setState({ kind: "error", message: "Вы не участник этой инициативы" });
      } else {
        setState({ kind: "error", message: "Не удалось загрузить пруфы" });
      }
    }
  }, [initiativeId]);

  useEffect(() => { load(); }, [load]);

  const handleApprove = (checkinId: number) => {
    startTransition(async () => {
      try {
        await approveInitiativeProof(initiativeId, checkinId, comments[checkinId]);
        setApproved((prev) => new Set(prev).add(checkinId));
      } catch {
        // ignore — user can retry
      }
    });
  };

  if (state.kind === "loading") {
    return <div className={styles.loading}>Загрузка пруфов...</div>;
  }
  if (state.kind === "error") {
    return <div className={styles.error}>{state.message}</div>;
  }

  const pending = state.proofs.filter((p) => !approved.has(p.checkin_id));

  return (
    <div className={styles.root}>
      <h3 className={styles.title}>Пруфы на ревью</h3>

      {pending.length === 0 && (
        <p className={styles.empty}>Нет пруфов, ожидающих проверки</p>
      )}

      <ul className={styles.list}>
        {pending.map((proof) => (
          <li key={proof.checkin_id} className={styles.card}>
            <div className={styles.author}>{proof.author_name}</div>
            <p className={styles.content}>{proof.content}</p>
            <time className={styles.time}>
              {new Date(proof.submitted_at).toLocaleString("ru-RU", {
                day: "numeric",
                month: "short",
                hour: "2-digit",
                minute: "2-digit",
              })}
            </time>
            <div className={styles.actions}>
              <input
                className={styles.commentInput}
                placeholder="Комментарий (необязательно)"
                value={comments[proof.checkin_id] ?? ""}
                onChange={(e) =>
                  setComments((c) => ({ ...c, [proof.checkin_id]: e.target.value }))
                }
              />
              <button
                className={styles.approveBtn}
                onClick={() => handleApprove(proof.checkin_id)}
                disabled={isPending}
              >
                Подтвердить
              </button>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
