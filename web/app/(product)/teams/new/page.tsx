"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { ApiError, createTeam } from "@/lib/api";
import type { TeamAIMode } from "@/lib/types";

import styles from "./page.module.css";

const AI_MODE_OPTIONS: { value: TeamAIMode; title: string; description: string }[] = [
  {
    value: "off",
    title: "БЕЗ AI",
    description:
      "Без вызовов AI вообще. Только шаблонные напоминания и сводки.",
  },
  {
    value: "metadata-only",
    title: "ТОЛЬКО МЕТАДАННЫЕ",
    description:
      "AI видит алиасы, заголовки целей, статусы. Содержимое заметок и пруфов не передаётся. По умолчанию.",
  },
  {
    value: "full",
    title: "ПОЛНЫЙ РЕЖИМ",
    description:
      "AI видит и помогает с содержимым. Каждый участник отдельно даёт согласие.",
  },
];

export default function NewTeamPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [aiMode, setAIMode] = useState<TeamAIMode>("metadata-only");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (submitting) return;
    setError(null);
    setSubmitting(true);
    try {
      const team = await createTeam({ name: name.trim(), ai_mode: aiMode });
      router.push(`/teams/${team.team.id}`);
    } catch (err) {
      const msg =
        err instanceof ApiError
          ? err.code === "validation.field_invalid"
            ? "Проверь название — от 1 до 80 символов."
            : err.message
          : "Не удалось создать команду.";
      setError(msg);
      setSubmitting(false);
    }
  }

  return (
    <div className={`page-shell ${styles.page}`}>
      <header className={styles.header}>
        <Link href="/teams" className={styles.back}>
          ← КОМАНДЫ
        </Link>
        <h1 className={styles.title}>НОВАЯ КОМАНДА</h1>
      </header>

      <form className={styles.form} onSubmit={handleSubmit}>
        <label className={styles.field}>
          <span className={styles.label}>НАЗВАНИЕ</span>
          <input
            type="text"
            className={styles.input}
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={80}
            required
            placeholder="Например, Бэкенд-команда"
          />
        </label>

        <fieldset className={styles.fieldset}>
          <legend className={styles.label}>РЕЖИМ AI</legend>
          {AI_MODE_OPTIONS.map((opt) => (
            <label
              key={opt.value}
              className={`${styles.option} ${aiMode === opt.value ? styles.optionActive : ""}`}
            >
              <input
                type="radio"
                name="ai_mode"
                value={opt.value}
                checked={aiMode === opt.value}
                onChange={() => setAIMode(opt.value)}
                className={styles.radio}
              />
              <span className={styles.optionTitle}>{opt.title}</span>
              <span className={styles.optionDesc}>{opt.description}</span>
            </label>
          ))}
        </fieldset>

        {error && <div className={styles.error}>{error}</div>}

        <button type="submit" className={styles.submit} disabled={submitting}>
          {submitting ? "СОЗДАЁМ…" : "СОЗДАТЬ КОМАНДУ →"}
        </button>
      </form>
    </div>
  );
}
