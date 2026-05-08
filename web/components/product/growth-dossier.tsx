"use client";

import { useState } from "react";
import styles from "./growth-dossier.module.css";

interface DossierEntry {
  goal_title: string;
  proofs_count: number;
  highlights: string[];
  skills: string[];
  conclusion: string;
}

interface DossierResult {
  period_label: string;
  total_proofs: number;
  active_weeks: number;
  top_skills: string[];
  goals: DossierEntry[];
  overall_summary: string;
  next_focus: string;
}

function addDays(date: Date, n: number): string {
  const d = new Date(date);
  d.setDate(d.getDate() + n);
  return d.toISOString().slice(0, 10);
}

const today = new Date();
const PRESET_RANGES = [
  { label: "Последние 4 недели", start: addDays(today, -28), end: addDays(today, 0) },
  { label: "Последние 3 месяца",  start: addDays(today, -90), end: addDays(today, 0) },
  { label: "Этот год",            start: `${today.getFullYear()}-01-01`, end: addDays(today, 0) },
];

export function GrowthDossierWidget() {
  const [startDate, setStartDate] = useState(addDays(today, -28));
  const [endDate, setEndDate] = useState(addDays(today, 0));
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<DossierResult | null>(null);
  const [error, setError] = useState("");

  async function generate() {
    setLoading(true);
    setError("");
    setResult(null);
    try {
      const r = await fetch("/v1/ai/growth-dossier", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ start_date: startDate, end_date: endDate }),
      });
      const data = await r.json();
      if (!r.ok) {
        setError(data.message ?? "Не удалось сгенерировать досье");
        return;
      }
      setResult(data);
    } catch {
      setError("Не удалось сгенерировать досье. Попробуй ещё раз.");
    } finally {
      setLoading(false);
    }
  }

  async function exportText() {
    if (!result) return;
    const lines = [
      `# Досье роста — ${result.period_label}`,
      "",
      `**Всего пруфов:** ${result.total_proofs}   **Активных недель:** ${result.active_weeks}`,
      "",
      `**Ключевые навыки:** ${result.top_skills.join(", ")}`,
      "",
      "---",
      "",
      ...result.goals.flatMap((g) => [
        `## ${g.goal_title}`,
        `Пруфов: ${g.proofs_count}`,
        "",
        `Достижения: ${g.highlights.join("; ")}`,
        `Навыки: ${g.skills.join(", ")}`,
        `Вывод: ${g.conclusion}`,
        "",
      ]),
      "---",
      "",
      `**Итог периода:** ${result.overall_summary}`,
      "",
      `**Следующий фокус:** ${result.next_focus}`,
    ];
    const blob = new Blob([lines.join("\n")], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `dosier-${startDate}-${endDate}.txt`;
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div className={styles.root}>
      <div className={styles.setupBlock}>
        <h2 className={styles.heading}>Досье роста</h2>
        <p className={styles.desc}>
          AI собирает итог периода из твоих пруфов: навыки, достижения, следующий фокус.
          Готово для встречи 1:1 или ИПР.
        </p>

        <div className={styles.presets}>
          {PRESET_RANGES.map((p) => (
            <button
              key={p.label}
              type="button"
              className={
                startDate === p.start && endDate === p.end
                  ? `${styles.preset} ${styles.presetActive}`
                  : styles.preset
              }
              onClick={() => { setStartDate(p.start); setEndDate(p.end); }}
            >
              {p.label}
            </button>
          ))}
        </div>

        <div className={styles.dateRow}>
          <label className={styles.dateLabel}>
            <span>С</span>
            <input
              type="date"
              className={styles.dateInput}
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
            />
          </label>
          <label className={styles.dateLabel}>
            <span>По</span>
            <input
              type="date"
              className={styles.dateInput}
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
            />
          </label>
        </div>

        {error && <p className={styles.error}>{error}</p>}

        <button
          type="button"
          className={styles.generateBtn}
          disabled={loading || !startDate || !endDate}
          onClick={generate}
        >
          {loading ? "AI анализирует пруфы…" : "Сгенерировать досье"}
        </button>
      </div>

      {result && (
        <div className={styles.dossier}>
          <div className={styles.dossierHeader}>
            <h3 className={styles.periodLabel}>{result.period_label}</h3>
            <button type="button" className={styles.exportBtn} onClick={exportText}>
              Скачать txt
            </button>
          </div>

          <div className={styles.topStats}>
            <div className={styles.stat}>
              <span className={styles.statNum}>{result.total_proofs}</span>
              <span className={styles.statLabel}>пруфов</span>
            </div>
            <div className={styles.stat}>
              <span className={styles.statNum}>{result.active_weeks}</span>
              <span className={styles.statLabel}>активных недель</span>
            </div>
          </div>

          {result.top_skills.length > 0 && (
            <div className={styles.skillsBlock}>
              <span className={styles.blockLabel}>Ключевые навыки</span>
              <div className={styles.skillTags}>
                {result.top_skills.map((s) => (
                  <span key={s} className={styles.skillTag}>{s}</span>
                ))}
              </div>
            </div>
          )}

          <div className={styles.goalsList}>
            {result.goals.map((g, i) => (
              <div key={i} className={styles.goalBlock}>
                <div className={styles.goalHeader}>
                  <span className={styles.goalTitle}>{g.goal_title}</span>
                  <span className={styles.goalProofs}>{g.proofs_count} пруф.</span>
                </div>
                {g.highlights.length > 0 && (
                  <ul className={styles.highlights}>
                    {g.highlights.map((h, j) => <li key={j}>{h}</li>)}
                  </ul>
                )}
                {g.skills.length > 0 && (
                  <div className={styles.goalSkills}>
                    {g.skills.map((s) => <span key={s} className={styles.goalSkillTag}>{s}</span>)}
                  </div>
                )}
                <p className={styles.goalConclusion}>{g.conclusion}</p>
              </div>
            ))}
          </div>

          <div className={styles.summaryBlock}>
            <span className={styles.blockLabel}>Итог периода</span>
            <p className={styles.summaryText}>{result.overall_summary}</p>
          </div>

          <div className={styles.nextBlock}>
            <span className={styles.blockLabel}>Следующий фокус</span>
            <p className={styles.nextText}>{result.next_focus}</p>
          </div>
        </div>
      )}
    </div>
  );
}
