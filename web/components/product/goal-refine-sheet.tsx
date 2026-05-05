"use client";

import { useEffect, useState, useTransition } from "react";
import { ApiError, refineGoal } from "@/lib/api";
import type { GoalRefineVariant } from "@/lib/types";
import styles from "./goal-refine-sheet.module.css";

interface GoalRefineSheetProps {
  draftText: string;
  onAccept: (variant: GoalRefineVariant, category: string) => void;
  onClose: () => void;
}

type SheetState =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "ready"; category: string; variants: GoalRefineVariant[] };

export function GoalRefineSheet({ draftText, onAccept, onClose }: GoalRefineSheetProps) {
  const [state, setState] = useState<SheetState>({ kind: "loading" });
  const [, startTransition] = useTransition();

  useEffect(() => {
    startTransition(async () => {
      try {
        const result = await refineGoal(draftText);
        setState({ kind: "ready", category: result.category, variants: result.variants });
      } catch (err) {
        if (err instanceof ApiError && err.status === 429) {
          setState({ kind: "error", message: "Лимит запросов исчерпан. Попробуй завтра." });
        } else {
          setState({ kind: "error", message: "Не удалось уточнить цель. Попробуй ещё раз." });
        }
      }
    });
  }, [draftText, startTransition]);

  // Close on Escape
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <>
      <div className={styles.backdrop} onClick={onClose} aria-hidden />
      <div className={styles.sheet} role="dialog" aria-label="Уточнить цель">
        <div className={styles.sheetHeader}>
          <span className={styles.sheetTitle}>⚡ УТОЧНИ ЦЕЛЬ</span>
          <button className={styles.closeBtn} onClick={onClose} aria-label="Закрыть">×</button>
        </div>

        {state.kind === "loading" && (
          <div className={styles.loadingState}>
            <span className={styles.loadingText}>АНАЛИЗИРУЮ…</span>
          </div>
        )}

        {state.kind === "error" && (
          <div className={styles.errorState}>
            <span className={styles.errorText}>{state.message}</span>
          </div>
        )}

        {state.kind === "ready" && (
          <>
            <div className={styles.categoryBadge}>
              {state.category.toUpperCase()}
            </div>
            <div className={styles.variants}>
              {state.variants.map((variant, i) => (
                <VariantCard
                  key={i}
                  variant={variant}
                  index={i + 1}
                  onAccept={() => onAccept(variant, state.category)}
                />
              ))}
            </div>
          </>
        )}
      </div>
    </>
  );
}

function VariantCard({
  variant,
  index,
  onAccept,
}: {
  variant: GoalRefineVariant;
  index: number;
  onAccept: () => void;
}) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className={styles.variantCard}>
      <div className={styles.variantHeader}>
        <span className={styles.variantIndex}>0{index}</span>
        <span className={styles.variantTitle}>{variant.title.toUpperCase()}</span>
      </div>

      <p className={styles.variantSmart}>{variant.smart}</p>

      {expanded && (
        <div className={styles.proofExamples}>
          <div className={styles.proofLabel}>ЧТО СЧИТАЕТСЯ ПРУФОМ:</div>
          <ul className={styles.proofList}>
            {variant.proof_examples.map((ex, j) => (
              <li key={j} className={styles.proofItem}>{ex}</li>
            ))}
          </ul>
        </div>
      )}

      <div className={styles.variantActions}>
        <button className={styles.acceptBtn} onClick={onAccept}>
          ВЗЯТЬ
        </button>
        <button
          className={styles.expandBtn}
          onClick={() => setExpanded((v) => !v)}
        >
          {expanded ? "СКРЫТЬ ПРИМЕРЫ" : "ПРИМЕРЫ ПРУФОВ"}
        </button>
      </div>
    </div>
  );
}
