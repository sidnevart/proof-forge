"use client";

import { useParams } from "next/navigation";
import { useState } from "react";

import { InitiativeList } from "@/components/product/initiative-list";
import type { InitiativeSpaceType } from "@/lib/types";

import styles from "./page.module.css";

type Tab = "initiatives";

const SPACE_LABELS: Record<string, string> = {
  teamspace: "Команда",
  community: "Сообщество",
};

export default function SpacePage() {
  const params = useParams<{ type: string; id: string }>();
  const spaceType = params?.type as InitiativeSpaceType;
  const spaceId = Number(params?.id);
  const [tab, setTab] = useState<Tab>("initiatives");

  if (!spaceType || !spaceId || isNaN(spaceId)) {
    return <div className={styles.error}>Неверный адрес пространства</div>;
  }

  const spaceLabel = SPACE_LABELS[spaceType] ?? "Пространство";

  return (
    <div className={styles.root}>
      <header className={styles.header}>
        <div className={styles.breadcrumb}>{spaceLabel}</div>
        <h1 className={styles.pageTitle}>Пространство #{spaceId}</h1>
      </header>

      <nav className={styles.tabs} role="tablist">
        <button
          role="tab"
          aria-selected={tab === "initiatives"}
          className={`${styles.tab} ${tab === "initiatives" ? styles.activeTab : ""}`}
          onClick={() => setTab("initiatives")}
        >
          Инициативы
        </button>
      </nav>

      <div className={styles.content}>
        {tab === "initiatives" && (
          <InitiativeList spaceType={spaceType} spaceId={spaceId} canCreate />
        )}
      </div>
    </div>
  );
}
