"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { ApiError, listMyTeams } from "@/lib/api";
import type { TeamDetail } from "@/lib/types";
import { TeamsList } from "@/components/product/teams-list";

import styles from "./page.module.css";

type State =
  | { kind: "loading" }
  | { kind: "ready"; teams: TeamDetail[] }
  | { kind: "error"; message: string };

export default function TeamsIndexPage() {
  const [state, setState] = useState<State>({ kind: "loading" });

  useEffect(() => {
    listMyTeams()
      .then((teams) => setState({ kind: "ready", teams }))
      .catch((err: unknown) => {
        const message =
          err instanceof ApiError && err.message
            ? err.message
            : "Не удалось загрузить команды";
        setState({ kind: "error", message });
      });
  }, []);

  return (
    <div className={`page-shell ${styles.page}`}>
      <header className={styles.header}>
        <h1 className={styles.title}>КОМАНДЫ</h1>
        <div className={styles.actions}>
          <Link href="/teams/join" className={styles.secondary}>
            ВСТУПИТЬ ПО КОДУ
          </Link>
          <Link href="/teams/new" className={styles.primary}>
            + СОЗДАТЬ КОМАНДУ
          </Link>
        </div>
      </header>

      {state.kind === "loading" && <div className={styles.loading}>ЗАГРУЖАЕМ…</div>}
      {state.kind === "error" && <div className={styles.error}>{state.message}</div>}
      {state.kind === "ready" && <TeamsList teams={state.teams} />}
    </div>
  );
}
