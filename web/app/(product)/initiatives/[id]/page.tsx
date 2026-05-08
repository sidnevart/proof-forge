"use client";

import { useParams } from "next/navigation";

import { InitiativeDetailView } from "@/components/product/initiative-detail";

import styles from "./page.module.css";

export default function InitiativeDetailPage() {
  const params = useParams<{ id: string }>();
  const initiativeId = Number(params?.id);

  if (!initiativeId || isNaN(initiativeId)) {
    return <div className={styles.error}>Инициатива не найдена</div>;
  }

  return (
    <div className={styles.root}>
      <InitiativeDetailView initiativeId={initiativeId} />
    </div>
  );
}
