"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";

import { ApiError, joinTeam } from "@/lib/api";

import styles from "./page.module.css";

export default function TeamJoinPage() {
  const search = useSearchParams();
  const router = useRouter();
  const initialCode = (search.get("code") ?? "").trim();

  const [code, setCode] = useState(initialCode);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const autoTried = useRef(false);

  // If a `?code=...` is present, attempt to join automatically once.
  useEffect(() => {
    if (autoTried.current) return;
    if (!initialCode) return;
    autoTried.current = true;
    void submit(initialCode);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialCode]);

  async function submit(rawCode: string) {
    if (busy) return;
    const trimmed = rawCode.trim();
    if (!trimmed) {
      setError("Введи код приглашения.");
      return;
    }
    setError(null);
    setBusy(true);
    try {
      const detail = await joinTeam(trimmed);
      router.push(`/teams/${detail.team.id}`);
    } catch (err) {
      const msg =
        err instanceof ApiError && err.code === "invite.invalid"
          ? "Код не найден. Проверь и попробуй ещё раз."
          : err instanceof ApiError && err.code === "team.archived"
            ? "Команда в архиве — присоединиться нельзя."
            : err instanceof ApiError && err.code === "team.full"
              ? "В команде нет свободных мест."
              : err instanceof ApiError
                ? err.message
                : "Не удалось вступить.";
      setError(msg);
      setBusy(false);
    }
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    void submit(code);
  }

  return (
    <div className={`page-shell ${styles.page}`}>
      <header className={styles.header}>
        <Link href="/teams" className={styles.back}>
          ← КОМАНДЫ
        </Link>
        <h1 className={styles.title}>ВСТУПИТЬ В КОМАНДУ</h1>
        <p className={styles.lead}>
          Тимлид прислал тебе код. Введи его — и ты в команде.
        </p>
      </header>

      <form className={styles.form} onSubmit={handleSubmit}>
        <label className={styles.field}>
          <span className={styles.label}>КОД ПРИГЛАШЕНИЯ</span>
          <input
            type="text"
            className={styles.input}
            value={code}
            onChange={(e) => setCode(e.target.value.toUpperCase())}
            placeholder="ABCDEF123456"
            autoFocus
            spellCheck={false}
            autoCapitalize="characters"
            autoCorrect="off"
          />
        </label>
        {error && <div className={styles.error}>{error}</div>}
        <button type="submit" className={styles.submit} disabled={busy}>
          {busy ? "ВСТУПАЕМ…" : "ВСТУПИТЬ →"}
        </button>
      </form>
    </div>
  );
}
