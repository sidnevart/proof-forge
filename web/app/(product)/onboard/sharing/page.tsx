"use client";

import { useState, useTransition } from "react";
import { useRouter } from "next/navigation";
import { updateSharingPrefs } from "@/lib/api";
import styles from "./page.module.css";

export default function OnboardSharingPage() {
  const router = useRouter();
  const [shareDefault, setShareDefault] = useState(false);
  const [isAnonymous, setIsAnonymous] = useState(false);
  const [alias, setAlias] = useState("");
  const [isPending, startTransition] = useTransition();

  function handleSkip() {
    router.push("/dashboard");
  }

  function handleSave() {
    startTransition(async () => {
      try {
        await updateSharingPrefs({ share_default: shareDefault, is_anonymous: isAnonymous, alias });
      } catch {
        // best-effort
      }
      router.push("/dashboard");
    });
  }

  return (
    <main className={styles.page}>
      <div className={styles.card}>
        <span className={styles.eyebrow}>Шаг 4 из 4</span>
        <h1 className={styles.heading}>ДЕЛИТЬСЯ ИЛИ НЕТ?</h1>
        <p className={styles.sub}>
          Всё приватно по умолчанию. Если хочешь вдохновлять других — включи.
        </p>

        <div className={styles.options}>
          <label className={styles.option}>
            <input
              type="checkbox"
              checked={shareDefault}
              onChange={(e) => setShareDefault(e.target.checked)}
            />
            <span>
              <strong>МОИ ПРУФЫ ВИДНЫ В ЛЕНТЕ</strong>
              <span>Одобренные пруфы попадают в /inspiration. Анонимно по умолчанию.</span>
            </span>
          </label>

          <label className={styles.option}>
            <input
              type="checkbox"
              checked={isAnonymous}
              onChange={(e) => setIsAnonymous(e.target.checked)}
            />
            <span>
              <strong>СКРЫТЬ ИМЯ</strong>
              <span>Показывать как «АНОНИМ» вместо твоего имени.</span>
            </span>
          </label>
        </div>

        {!isAnonymous && (
          <div className={styles.aliasField}>
            <label>
              <span className={styles.aliasLabel}>КАК ТЕБЯ НАЗЫВАТЬ</span>
              <input
                className={styles.aliasInput}
                placeholder="Твой публичный псевдоним (ALL CAPS)"
                value={alias}
                onChange={(e) => setAlias(e.target.value.toUpperCase())}
                maxLength={30}
              />
            </label>
          </div>
        )}

        <div className={styles.actions}>
          <button className={styles.saveBtn} onClick={handleSave} disabled={isPending}>
            {isPending ? "СОХРАНЯЕМ…" : "СОХРАНИТЬ"}
          </button>
          <button className={styles.skipBtn} onClick={handleSkip}>
            ПРОПУСТИТЬ
          </button>
        </div>
      </div>
    </main>
  );
}
