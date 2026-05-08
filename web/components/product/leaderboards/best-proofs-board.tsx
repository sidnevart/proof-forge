"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import styles from "./best-proofs-board.module.css";

interface ProofArtifact {
  id: number;
  goal_title: string;
  summary: string;
  artifact_url: string | null;
  reaction_count: number;
  author_display_name: string;
  submitted_at: string;
  week_label: string;
}

function apiFetch<T>(url: string): Promise<T> {
  return fetch(url, { credentials: "include" }).then((r) => {
    if (!r.ok) throw new Error(r.statusText);
    return r.json() as Promise<T>;
  });
}

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const days = Math.floor(diff / 86_400_000);
  if (days === 0) return "сегодня";
  if (days === 1) return "вчера";
  if (days < 7) return `${days} дн. назад`;
  return new Date(iso).toLocaleDateString("ru-RU", { day: "numeric", month: "short" });
}

interface Props {
  spaceId: number;
  spaceKind: "teamspace" | "community_space";
}

export function BestProofsBoard({ spaceId, spaceKind }: Props) {
  const [proofs, setProofs] = useState<ProofArtifact[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const url =
      spaceKind === "community_space"
        ? `/v1/community-spaces/${spaceId}/best-proofs`
        : null;

    if (!url) {
      setProofs([]);
      setLoading(false);
      return;
    }

    apiFetch<{ proofs: ProofArtifact[] }>(url)
      .then((d) => setProofs(d.proofs ?? []))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [spaceId, spaceKind]);

  return (
    <div className={styles.root}>
      <h3 className={styles.heading}>Лучшие пруфы недели</h3>
      <p className={styles.hint}>Соревнуются артефакты — не люди</p>

      {loading ? (
        <p className={styles.empty}>Загрузка…</p>
      ) : proofs.length === 0 ? (
        <p className={styles.empty}>Пруфов пока нет</p>
      ) : (
        <div className={styles.list}>
          {proofs.map((p, i) => (
            <div key={p.id} className={styles.card}>
              <div className={styles.rank}>{i + 1}</div>
              <div className={styles.body}>
                <div className={styles.goalTitle}>{p.goal_title}</div>
                <p className={styles.summary}>{p.summary}</p>
                <div className={styles.footer}>
                  <span className={styles.author}>{p.author_display_name}</span>
                  <span className={styles.dot}>·</span>
                  <span className={styles.date}>{timeAgo(p.submitted_at)}</span>
                  {p.reaction_count > 0 && (
                    <>
                      <span className={styles.dot}>·</span>
                      <span className={styles.reactions}>{p.reaction_count} 👍</span>
                    </>
                  )}
                </div>
              </div>
              {p.artifact_url && (
                <Link href={p.artifact_url} className={styles.artifactLink} target="_blank" rel="noopener">
                  ↗
                </Link>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
