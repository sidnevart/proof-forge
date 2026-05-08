"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";

import { ApiError, checkWorkspaceSlug, createWorkspace } from "@/lib/api";
import type { WorkspaceType } from "@/lib/types";

import styles from "./workspace-setup-screen.module.css";

type Step = "type" | "details";

export function WorkspaceSetupScreen() {
  const router = useRouter();
  const [step, setStep] = useState<Step>("type");
  const [type, setType] = useState<WorkspaceType | null>(null);
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [slugEdited, setSlugEdited] = useState(false);
  const [slugAvailable, setSlugAvailable] = useState<boolean | null>(null);
  const [slugChecking, setSlugChecking] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const nameInputRef = useRef<HTMLInputElement>(null);

  function toSlug(s: string): string {
    return s
      .toLowerCase()
      .replace(/[^a-z0-9\s-]/g, "")
      .replace(/\s+/g, "-")
      .replace(/-+/g, "-")
      .replace(/^-|-$/g, "")
      .slice(0, 50);
  }

  useEffect(() => {
    if (!slugEdited) {
      setSlug(toSlug(name));
      setSlugAvailable(null);
    }
  }, [name, slugEdited]);

  useEffect(() => {
    if (slug.length < 2) {
      setSlugAvailable(null);
      return;
    }
    if (!/^[a-z0-9][a-z0-9-]{1,49}$/.test(slug)) {
      setSlugAvailable(false);
      return;
    }
    setSlugChecking(true);
    const timer = setTimeout(async () => {
      try {
        const available = await checkWorkspaceSlug(slug);
        setSlugAvailable(available);
      } catch {
        setSlugAvailable(null);
      } finally {
        setSlugChecking(false);
      }
    }, 400);
    return () => clearTimeout(timer);
  }, [slug]);

  function selectType(t: WorkspaceType) {
    setType(t);
    setStep("details");
    setTimeout(() => nameInputRef.current?.focus(), 50);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (submitting || !type || !name.trim() || slugAvailable !== true) return;
    setError(null);
    setSubmitting(true);
    try {
      const ws = await createWorkspace({ name: name.trim(), slug, type });
      router.push(`/workspaces/${ws.id}`);
    } catch (err) {
      const msg =
        err instanceof ApiError
          ? err.status === 409
            ? "Этот адрес уже занят. Попробуй другой."
            : err.message
          : "Не удалось создать пространство.";
      setError(msg);
      setSubmitting(false);
    }
  }

  const canSubmit =
    name.trim().length > 0 && slugAvailable === true && !submitting;

  return (
    <div className={styles.screen}>
      <p className={styles.eyebrow}>Создать пространство</p>

      {step === "type" && (
        <>
          <h1 className={styles.title}>Для кого это пространство?</h1>
          <button
            type="button"
            className={styles.typeCard}
            onClick={() => selectType("organization")}
          >
            <span className={styles.typeCardIcon}>🏢</span>
            <span className={styles.typeCardText}>
              <span className={styles.typeCardTitle}>Для команды или компании</span>
              <span className={styles.typeCardDesc}>
                Отдел, направление или рабочая группа
              </span>
            </span>
          </button>
          <button
            type="button"
            className={styles.typeCard}
            onClick={() => selectType("community")}
          >
            <span className={styles.typeCardIcon}>👥</span>
            <span className={styles.typeCardText}>
              <span className={styles.typeCardTitle}>Для сообщества или клуба</span>
              <span className={styles.typeCardDesc}>
                Закрытая группа, клуб по интересам
              </span>
            </span>
          </button>
        </>
      )}

      {step === "details" && (
        <form onSubmit={handleSubmit}>
          <h1 className={styles.title}>
            {type === "organization" ? "Команда или компания" : "Сообщество или клуб"}
          </h1>

          <div className={styles.field}>
            <label className={styles.fieldLabel} htmlFor="ws-name">
              Название
            </label>
            <input
              id="ws-name"
              ref={nameInputRef}
              type="text"
              className={styles.input}
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={80}
              placeholder="Например, T-Bank AI Stream"
              required
              autoComplete="off"
            />
          </div>

          <div className={styles.field}>
            <label className={styles.fieldLabel} htmlFor="ws-slug">
              Адрес
            </label>
            <div className={styles.slugRow}>
              <span className={styles.slugPrefix}>proofforge.io/</span>
              <input
                id="ws-slug"
                type="text"
                className={styles.input}
                value={slug}
                onChange={(e) => {
                  setSlugEdited(true);
                  setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""));
                }}
                maxLength={50}
                placeholder="my-space"
                autoComplete="off"
              />
            </div>
            {slug.length >= 2 && (
              <p
                className={`${styles.slugStatus} ${
                  slugChecking
                    ? ""
                    : slugAvailable === true
                    ? styles.available
                    : styles.taken
                }`}
              >
                {slugChecking
                  ? "Проверяем…"
                  : slugAvailable === true
                  ? "✓ Адрес свободен"
                  : "✗ Этот адрес уже занят"}
              </p>
            )}
          </div>

          {error && <p className={styles.formError}>{error}</p>}

          <button
            type="submit"
            className={styles.submitBtn}
            disabled={!canSubmit}
          >
            {submitting ? "Создаём…" : "Создать пространство"}
          </button>

          <button
            type="button"
            className={styles.backBtn}
            onClick={() => {
              setStep("type");
              setName("");
              setSlug("");
              setSlugEdited(false);
              setSlugAvailable(null);
              setError(null);
            }}
          >
            ← Назад
          </button>
        </form>
      )}
    </div>
  );
}
