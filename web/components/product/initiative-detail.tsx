"use client";

import { useCallback, useEffect, useState, useTransition } from "react";

import { ApiError, getInitiative, joinInitiative } from "@/lib/api";
import type { InitiativeDetail } from "@/lib/types";
import { InitiativePeerReview } from "./initiative-peer-review";

import styles from "./initiative-detail.module.css";

type Props = {
  initiativeId: number;
};

type State =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "ready"; detail: InitiativeDetail; joined: boolean };

export function InitiativeDetailView({ initiativeId }: Props) {
  const [state, setState] = useState<State>({ kind: "loading" });
  const [isPending, startTransition] = useTransition();

  const load = useCallback(async () => {
    try {
      const detail = await getInitiative(initiativeId);
      setState({ kind: "ready", detail, joined: false });
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setState({ kind: "error", message: "Инициатива не найдена" });
      } else {
        setState({ kind: "error", message: "Не удалось загрузить инициативу" });
      }
    }
  }, [initiativeId]);

  useEffect(() => { load(); }, [load]);

  const handleJoin = () => {
    startTransition(async () => {
      try {
        await joinInitiative(initiativeId);
        setState((prev) =>
          prev.kind === "ready" ? { ...prev, joined: true } : prev,
        );
      } catch {
        // idempotent — treat error as already joined
        setState((prev) =>
          prev.kind === "ready" ? { ...prev, joined: true } : prev,
        );
      }
    });
  };

  if (state.kind === "loading") {
    return <div className={styles.loading}>Загрузка...</div>;
  }
  if (state.kind === "error") {
    return <div className={styles.error}>{state.message}</div>;
  }

  const { detail, joined } = state;

  return (
    <div className={styles.root}>
      <div className={styles.hero}>
        <h1 className={styles.title}>{detail.title}</h1>
        {detail.description && <p className={styles.desc}>{detail.description}</p>}
        <div className={styles.criteria}>
          <span className={styles.criteriaLabel}>Критерий пруфа</span>
          <p>{detail.proof_criteria}</p>
        </div>
        <div className={styles.stats}>
          <div className={styles.stat}>
            <span className={styles.statNum}>{detail.participant_count}</span>
            <span className={styles.statLabel}>участников</span>
          </div>
          <div className={styles.stat}>
            <span className={styles.statNum}>{detail.active_today}</span>
            <span className={styles.statLabel}>активны сегодня</span>
          </div>
          {detail.pending_proof_count > 0 && (
            <div className={styles.stat}>
              <span className={styles.statNum}>{detail.pending_proof_count}</span>
              <span className={styles.statLabel}>ждут ревью</span>
            </div>
          )}
        </div>
        {!joined && (
          <button className={styles.joinBtn} onClick={handleJoin} disabled={isPending}>
            {isPending ? "Вступаем..." : "Вступить в инициативу"}
          </button>
        )}
        {joined && <p className={styles.joinedNote}>Вы вступили в инициативу</p>}
      </div>

      {detail.participants && detail.participants.length > 0 && (
        <div className={styles.section}>
          <h2 className={styles.sectionTitle}>Участники</h2>
          <ul className={styles.participants}>
            {detail.participants.map((p) => (
              <li key={p.user_id} className={styles.participant}>
                <span className={styles.participantName}>{p.display_name}</span>
                <span className={styles.participantCount}>{p.proof_count} пруфов</span>
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className={styles.section}>
        <InitiativePeerReview initiativeId={initiativeId} />
      </div>
    </div>
  );
}
