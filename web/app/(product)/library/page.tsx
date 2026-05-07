"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { getPublicTemplates, reportContent } from "@/lib/api";
import type { PublicGoal } from "@/lib/types";
import styles from "./page.module.css";

const CATEGORIES = ["", "фитнес", "учёба", "работа", "творчество", "развитие"];

export default function LibraryPage() {
  const router = useRouter();
  const [templates, setTemplates] = useState<PublicGoal[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [q, setQ] = useState("");
  const [category, setCategory] = useState("");
  const [cursor, setCursor] = useState<number | undefined>(undefined);
  const [hasMore, setHasMore] = useState(true);
  const [expanded, setExpanded] = useState<number | null>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);

  async function load(reset: boolean) {
    if (reset) setLoading(true);
    else setLoadingMore(true);
    try {
      const data = await getPublicTemplates({
        q: q || undefined,
        category: category || undefined,
        cursor: reset ? undefined : cursor,
        limit: 24,
      });
      const next = data.templates ?? [];
      setTemplates((prev) => (reset ? next : [...prev, ...next]));
      setHasMore(next.length === 24);
      if (next.length > 0) setCursor(next[next.length - 1].id);
    } catch {
      // silent
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  }

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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hasMore, loadingMore]);

  function handleUseTemplate(t: PublicGoal) {
    const params = new URLSearchParams({
      title: t.title,
      proof_examples: t.proof_examples,
      category: t.category,
    });
    router.push(`/goals/new?${params.toString()}`);
  }

  return (
    <main className={styles.page}>
      <header className={styles.header}>
        <div>
          <span className="eyebrow">Шаблоны кругов</span>
          <h1 className={styles.heading}>БИБЛИОТЕКА. ГОТОВЫЕ ПОДХОДЫ.</h1>
        </div>
        <a href="/inspiration" className={styles.feedLink}>ЛЕНТА ПРУФОВ →</a>
      </header>

      <div className={styles.filters}>
        <input
          className={styles.search}
          placeholder="Найти круг..."
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <div className={styles.categoryTabs}>
          {CATEGORIES.map((c) => (
            <button
              key={c || "all"}
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
      ) : templates.length === 0 ? (
        <div className={styles.empty}>ПОКА ПУСТО. ИДИ ПЕРВЫМ.</div>
      ) : (
        <ol className={styles.list}>
          {templates.map((t) => (
            <TemplateRow
              key={t.id}
              template={t}
              expanded={expanded === t.id}
              onToggle={() => setExpanded(expanded === t.id ? null : t.id)}
              onUse={() => handleUseTemplate(t)}
            />
          ))}
        </ol>
      )}

      {loadingMore && <div className={styles.loadingMore}>ЗАГРУЖАЕМ ЕЩЁ…</div>}
      <div ref={sentinelRef} style={{ height: 1 }} />
    </main>
  );
}

function TemplateRow({
  template,
  expanded,
  onToggle,
  onUse,
}: {
  template: PublicGoal;
  expanded: boolean;
  onToggle: () => void;
  onUse: () => void;
}) {
  const [reported, setReported] = useState(false);

  async function handleReport() {
    if (reported) return;
    try {
      await reportContent("goal", template.id, "inappropriate");
      setReported(true);
    } catch {
      // silent
    }
  }

  return (
    <li className={styles.row}>
      <div className={styles.rowMain}>
        <div className={styles.rowLeft} onClick={onToggle}>
          {template.category && (
            <span className={styles.cat}>{template.category.toUpperCase()}</span>
          )}
          <span className={styles.title}>{template.title.toUpperCase()}</span>
          <span className={styles.alias}>{template.author_alias}</span>
        </div>
        <div className={styles.rowActions}>
          <button className={styles.useBtn} onClick={onUse}>
            ВЗЯТЬ КАК ШАБЛОН
          </button>
          <button className={styles.expandBtn} onClick={onToggle}>
            {expanded ? "▲" : "▼"}
          </button>
        </div>
      </div>

      {expanded && (
        <div className={styles.expanded}>
          {template.proof_examples && (
            <div className={styles.proofHint}>
              <span className={styles.proofLabel}>ПРУФЫ:</span>
              <p>{template.proof_examples}</p>
            </div>
          )}
          <div className={styles.expandedFooter}>
            <time className={styles.time}>
              {new Date(template.created_at).toLocaleDateString("ru-RU")}
            </time>
            <button
              className={styles.reportBtn}
              onClick={handleReport}
              disabled={reported}
            >
              {reported ? "ОТПРАВЛЕНО" : "СООБЩИТЬ"}
            </button>
          </div>
        </div>
      )}
    </li>
  );
}
