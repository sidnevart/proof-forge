"use client";

import { useEffect, useState } from "react";
import { getAIProofDrafts, acceptAIProofDraft, rejectAIProofDraft } from "@/lib/api";
import type { AIProofDraft } from "@/lib/types";
import styles from "./ai-draft-card.module.css";

export function AIDraftCard({ goalID }: { goalID: number }) {
  const [draft, setDraft] = useState<AIProofDraft | null>(null);
  const [loading, setLoading] = useState(true);
  const [processing, setProcessing] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    getAIProofDrafts()
      .then((res) => {
        if (cancelled) return;
        const d = (res.drafts ?? []).find((x) => x.goal_id === goalID) ?? null;
        setDraft(d);
      })
      .catch(() => {
        if (cancelled) return;
        setDraft(null);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [goalID]);

  async function handleAccept() {
    if (!draft) return;
    setProcessing(true);
    try {
      await acceptAIProofDraft(draft.id);
      setDraft(null);
    } catch {
      // ignore
    } finally {
      setProcessing(false);
    }
  }

  async function handleReject() {
    if (!draft) return;
    setProcessing(true);
    try {
      await rejectAIProofDraft(draft.id);
      setDraft(null);
    } catch {
      // ignore
    } finally {
      setProcessing(false);
    }
  }

  if (loading) {
    return (
      <div className={styles.skeleton}>
        <div className={styles.skeletonPulse} />
      </div>
    );
  }

  if (!draft) return null;

  return (
    <article className={styles.card} aria-label="AI черновик пруфа">
      <div className={styles.header}>
        <span className={styles.eyebrow}>AI ЧЕРНОВИК</span>
        <button
          className={styles.dismissBtn}
          onClick={handleReject}
          disabled={processing}
          aria-label="Отклонить"
          title="Отклонить"
        >
          ×
        </button>
      </div>
      <div className={styles.rationale}>{draft.rationale}</div>
      <div className={styles.meta}>
        Confidence: {draft.confidence} · {draft.note_ids?.length ?? 0} заметок
      </div>
      <div className={styles.actions}>
        <button className={styles.acceptBtn} onClick={handleAccept} disabled={processing}>
          Использовать
        </button>
        <button className={styles.rejectBtn} onClick={handleReject} disabled={processing}>
          Отменить
        </button>
      </div>
    </article>
  );
}
