"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams, useRouter } from "next/navigation";

import {
  approveCheckIn,
  getCirclesFeed,
  getPublicProofs,
  rejectCheckIn,
  reportContent,
} from "@/lib/api";
import type { CircleFeedItem, PublicProof } from "@/lib/types";
import styles from "./page.module.css";

// ─────────────────────────────────────────
// Main page — tab router
// ─────────────────────────────────────────

type Tab = "circle" | "similar";

export default function FeedPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const tabParam = searchParams.get("tab");
  const activeTab: Tab = tabParam === "similar" ? "similar" : "circle";

  function setTab(t: Tab) {
    router.replace(`/feed?tab=${t}`, { scroll: false });
  }

  return (
    <main className={styles.page}>
      <h1 className={styles.title}>ЛЕНТА</h1>

      <div className={styles.tabs} role="tablist">
        <button
          role="tab"
          aria-selected={activeTab === "circle"}
          className={`${styles.tab} ${activeTab === "circle" ? styles.tabActive : ""}`}
          onClick={() => setTab("circle")}
        >
          МОЙ КРУГ
        </button>
        <button
          role="tab"
          aria-selected={activeTab === "similar"}
          className={`${styles.tab} ${activeTab === "similar" ? styles.tabActive : ""}`}
          onClick={() => setTab("similar")}
        >
          ПОХОЖИЕ ЦЕЛИ
        </button>
      </div>

      <div role="tabpanel">
        {activeTab === "circle" ? <CircleFeed /> : <SimilarFeed />}
      </div>
    </main>
  );
}

// ─────────────────────────────────────────
// CircleFeed — МОЙ КРУГ tab
// ─────────────────────────────────────────

function CircleFeed() {
  const [items, setItems] = useState<CircleFeedItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [cursor, setCursor] = useState<number | undefined>(undefined);
  const [hasMore, setHasMore] = useState(true);
  const sentinelRef = useRef<HTMLDivElement>(null);

  const load = useCallback(
    async (reset: boolean) => {
      if (reset) {
        setLoading(true);
      } else {
        setLoadingMore(true);
      }
      try {
        const data = await getCirclesFeed({
          cursor: reset ? undefined : cursor,
          limit: 24,
        });
        const next = data.items ?? [];
        setItems((prev) => (reset ? next : [...prev, ...next]));
        setHasMore(next.length === 24);
        if (next.length > 0) setCursor(next[next.length - 1].id);
      } catch {
        // silent — user may not be in a circle
      } finally {
        setLoading(false);
        setLoadingMore(false);
      }
    },
    [cursor] // eslint-disable-line react-hooks/exhaustive-deps
  );

  useEffect(() => {
    void load(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const el = sentinelRef.current;
    if (!el) return;
    const obs = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loadingMore) {
          void load(false);
        }
      },
      { threshold: 0.1 }
    );
    obs.observe(el);
    return () => obs.disconnect();
  }, [hasMore, loadingMore, load]);

  function handleApprove(id: number) {
    setItems((prev) =>
      prev.map((item) =>
        item.id === id ? { ...item, status: "approved", can_approve: false } : item
      )
    );
    void approveCheckIn(id).catch(() => {
      // revert on error
      setItems((prev) =>
        prev.map((item) =>
          item.id === id ? { ...item, status: "submitted", can_approve: true } : item
        )
      );
    });
  }

  function handleReject(id: number) {
    setItems((prev) =>
      prev.map((item) =>
        item.id === id ? { ...item, status: "approved", can_approve: false } : item
      )
    );
    void rejectCheckIn(id).catch(() => {
      setItems((prev) =>
        prev.map((item) =>
          item.id === id ? { ...item, status: "submitted", can_approve: true } : item
        )
      );
    });
  }

  if (loading) {
    return <div className={styles.empty}>ЗАГРУЖАЕМ…</div>;
  }

  if (items.length === 0) {
    return (
      <div className={styles.emptyCard}>
        <p className={styles.emptyHeading}>ПОКА ТИХО.</p>
        <p className={styles.emptySub}>
          Ты ещё не в круге или участники кругов не сдавали пруфы.
          Создай цель и пригласи людей — лента оживёт.
        </p>
        <Link href="/onboard/goal" className={styles.ctaSolid}>
          СОЗДАТЬ ЦЕЛЬ →
        </Link>
      </div>
    );
  }

  return (
    <>
      <div className={styles.grid}>
        {items.map((item) => (
          <CircleProofCard
            key={item.id}
            item={item}
            onApprove={handleApprove}
            onReject={handleReject}
          />
        ))}
      </div>
      {loadingMore && <div className={styles.loadingMore}>ЗАГРУЖАЕМ ЕЩЁ…</div>}
      <div ref={sentinelRef} style={{ height: 1 }} />
    </>
  );
}

type CircleProofCardProps = {
  item: CircleFeedItem;
  onApprove: (id: number) => void;
  onReject: (id: number) => void;
};

function CircleProofCard({ item, onApprove, onReject }: CircleProofCardProps) {
  const [busy, setBusy] = useState(false);

  async function doApprove() {
    setBusy(true);
    onApprove(item.id);
  }

  async function doReject() {
    setBusy(true);
    onReject(item.id);
  }

  return (
    <article className={styles.card}>
      <div className={styles.cardMeta}>
        <span className={styles.alias}>{item.author_alias}</span>
        {item.streak > 0 && <span className={styles.streak}>🔥{item.streak}</span>}
        {item.category && (
          <span className={styles.cat}>{item.category.toUpperCase()}</span>
        )}
        <span className={styles.circleBadge}>{item.circle_name.toUpperCase()}</span>
      </div>

      <p className={styles.goalTitle}>{item.goal_title.toUpperCase()}</p>

      {item.text_content && (
        <p className={styles.proofText}>{item.text_content}</p>
      )}
      {item.external_url && (
        <a
          className={styles.proofLink}
          href={item.external_url}
          target="_blank"
          rel="noopener noreferrer"
        >
          {item.external_url}
        </a>
      )}

      {item.status === "approved" && (
        <p className={styles.approvedBadge}>✓ ОДОБРЕНО</p>
      )}

      {item.can_approve && item.status === "submitted" && (
        <div className={styles.approveRow}>
          <button
            type="button"
            className={styles.approveBtn}
            disabled={busy}
            onClick={doApprove}
          >
            ✓ ОДОБРИТЬ
          </button>
          <button
            type="button"
            className={styles.rejectBtn}
            disabled={busy}
            onClick={doReject}
          >
            ✗ ОТКЛОНИТЬ
          </button>
        </div>
      )}

      <div className={styles.cardFooter}>
        <time className={styles.time}>
          {new Date(item.submitted_at).toLocaleDateString("ru-RU")}
        </time>
      </div>
    </article>
  );
}

// ─────────────────────────────────────────
// SimilarFeed — ПОХОЖИЕ ЦЕЛИ tab (migrated from /inspiration)
// ─────────────────────────────────────────

const CATEGORIES = ["", "фитнес", "учёба", "работа", "творчество", "развитие"];

function SimilarFeed() {
  const [proofs, setProofs] = useState<PublicProof[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [q, setQ] = useState("");
  const [category, setCategory] = useState("");
  const [cursor, setCursor] = useState<number | undefined>(undefined);
  const [hasMore, setHasMore] = useState(true);
  const sentinelRef = useRef<HTMLDivElement>(null);

  const load = useCallback(
    async (reset: boolean) => {
      if (reset) {
        setLoading(true);
      } else {
        setLoadingMore(true);
      }
      try {
        const data = await getPublicProofs({
          q: q || undefined,
          category: category || undefined,
          cursor: reset ? undefined : cursor,
          limit: 24,
        });
        const next = data.proofs ?? [];
        setProofs((prev) => (reset ? next : [...prev, ...next]));
        setHasMore(next.length === 24);
        if (next.length > 0) setCursor(next[next.length - 1].id);
      } catch {
        // silent
      } finally {
        setLoading(false);
        setLoadingMore(false);
      }
    },
    [q, category, cursor] // eslint-disable-line react-hooks/exhaustive-deps
  );

  useEffect(() => {
    void load(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q, category]);

  useEffect(() => {
    const el = sentinelRef.current;
    if (!el) return;
    const obs = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loadingMore) {
          void load(false);
        }
      },
      { threshold: 0.1 }
    );
    obs.observe(el);
    return () => obs.disconnect();
  }, [hasMore, loadingMore, load]);

  return (
    <>
      <div className={styles.filters}>
        <input
          className={styles.search}
          placeholder="Поиск по теме..."
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <div className={styles.categoryTabs}>
          {CATEGORIES.map((c) => (
            <button
              key={c || "all"}
              type="button"
              className={`${styles.catBtn} ${category === c ? styles.catActive : ""}`}
              onClick={() => setCategory(c)}
            >
              {c || "ВСЕ"}
            </button>
          ))}
        </div>
      </div>

      {loading ? (
        <div className={styles.empty}>ЗАГРУЖАЕМ…</div>
      ) : proofs.length === 0 ? (
        <div className={styles.empty}>ПОКА ТИХО. ИДИ ПЕРВЫМ.</div>
      ) : (
        <div className={styles.grid}>
          {proofs.map((p) => (
            <SimilarProofCard key={p.id} proof={p} />
          ))}
        </div>
      )}

      {loadingMore && <div className={styles.loadingMore}>ЗАГРУЖАЕМ ЕЩЁ…</div>}
      <div ref={sentinelRef} style={{ height: 1 }} />
    </>
  );
}

function SimilarProofCard({ proof }: { proof: PublicProof }) {
  const [reported, setReported] = useState(false);

  async function handleReport() {
    if (reported) return;
    try {
      await reportContent("checkin", proof.id, "inappropriate");
      setReported(true);
    } catch {
      // silent
    }
  }

  return (
    <article className={styles.card}>
      <div className={styles.cardMeta}>
        <span className={styles.alias}>{proof.author_alias}</span>
        {proof.streak > 0 && (
          <span className={styles.streak}>🔥{proof.streak}</span>
        )}
        {proof.category && (
          <span className={styles.cat}>{proof.category.toUpperCase()}</span>
        )}
      </div>
      <p className={styles.goalTitle}>{proof.goal_title.toUpperCase()}</p>
      {proof.text_content && (
        <p className={styles.proofText}>{proof.text_content}</p>
      )}
      {proof.external_url && (
        <a
          className={styles.proofLink}
          href={proof.external_url}
          target="_blank"
          rel="noopener noreferrer"
        >
          {proof.external_url}
        </a>
      )}
      <div className={styles.cardFooter}>
        <time className={styles.time}>
          {new Date(proof.created_at).toLocaleDateString("ru-RU")}
        </time>
        <button
          type="button"
          className={styles.reportBtn}
          onClick={handleReport}
          disabled={reported}
        >
          {reported ? "ОТПРАВЛЕНО" : "СООБЩИТЬ"}
        </button>
      </div>
    </article>
  );
}
