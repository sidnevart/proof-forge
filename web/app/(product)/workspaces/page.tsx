"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { listWorkspaces } from "@/lib/api";
import type { Workspace } from "@/lib/types";

import styles from "./page.module.css";

export default function WorkspacesPage() {
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    listWorkspaces()
      .then(setWorkspaces)
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className={`page-shell ${styles.page}`}>
      <header className={styles.header}>
        <p className={styles.eyebrow}>Мои пространства</p>
        <Link href="/workspaces/new" className={styles.newBtn}>
          + Новое
        </Link>
      </header>

      {loading && <p className={styles.hint}>Загрузка…</p>}

      {!loading && workspaces.length === 0 && (
        <div className={styles.empty}>
          <p className={styles.emptyTitle}>Нет пространств</p>
          <p className={styles.emptyDesc}>Создай первое пространство для команды или сообщества</p>
          <Link href="/workspaces/new" className={styles.createBtn}>
            Создать пространство
          </Link>
        </div>
      )}

      {!loading && workspaces.length > 0 && (
        <ul className={styles.list}>
          {workspaces.map((ws) => (
            <li key={ws.id} className={styles.card}>
              <Link href={`/workspaces/${ws.id}`} className={styles.cardLink}>
                <span className={styles.cardIcon}>
                  {ws.type === "organization" ? "🏢" : "👥"}
                </span>
                <span className={styles.cardBody}>
                  <span className={styles.cardName}>{ws.name}</span>
                  <span className={styles.cardMeta}>
                    proofforge.io/{ws.slug}
                  </span>
                </span>
                {!ws.is_active && (
                  <span className={styles.frozenBadge}>заморожено</span>
                )}
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
