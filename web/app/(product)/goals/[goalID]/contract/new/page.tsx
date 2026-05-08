"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";

import { getGoal } from "@/lib/api";
import { ProofContractScreen } from "@/components/product/proof-contract-screen";
import type { GoalView } from "@/lib/types";

import styles from "./page.module.css";

export default function NewContractPage() {
  const params = useParams();
  const goalId = Number(params.goalID);
  const [goalView, setGoalView] = useState<GoalView | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getGoal(goalId)
      .then((res) => setGoalView(res.goal))
      .finally(() => setLoading(false));
  }, [goalId]);

  return (
    <div className={`page-shell ${styles.page}`}>
      <header className={styles.header}>
        <Link href={`/goals/${goalId}`} className={styles.back}>
          ← {goalView?.goal.title || "Цель"}
        </Link>
      </header>

      {loading && <p className={styles.hint}>Загрузка…</p>}

      {!loading && goalView && (
        <ProofContractScreen goalId={goalId} goal={goalView} />
      )}
    </div>
  );
}
