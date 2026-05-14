"use client";

import { useState, useTransition } from "react";
import { updateSharingPrefs } from "@/lib/api";
import styles from "./page.module.css";

export default function SharingSettingsPage() {
  const [shareDefault, setShareDefault] = useState(false);
  const [isAnonymous, setIsAnonymous] = useState(false);
  const [alias, setAlias] = useState("");
  const [saved, setSaved] = useState(false);
  const [isPending, startTransition] = useTransition();

  function handleSave() {
    setSaved(false);
    startTransition(async () => {
      try {
        await updateSharingPrefs({ share_default: shareDefault, is_anonymous: isAnonymous, alias });
        setSaved(true);
      } catch {
        // silent
      }
    });
  }

  return (
    <div>
      <header className={styles.header}>
        <span className="eyebrow">Настройки</span>
        <h1>ПУБЛИЧНОСТЬ</h1>
      </header>

      <div className={styles.card}>
        <p className={styles.desc}>
          Всё приватно по умолчанию. Включай только то, что хочешь показывать в ленте и библиотеке.
        </p>

        <div className={styles.options}>
          <label className={styles.option}>
            <input
              type="checkbox"
              checked={shareDefault}
              onChange={(e) => setShareDefault(e.target.checked)}
            />
            <span>
              <strong>ПРУФЫ ВИДНЫ В ЛЕНТЕ ПО УМОЛЧАНИЮ</strong>
              <span>Новые одобренные пруфы автоматически публикуются в /inspiration.</span>
            </span>
          </label>

          <label className={styles.option}>
            <input
              type="checkbox"
              checked={isAnonymous}
              onChange={(e) => setIsAnonymous(e.target.checked)}
            />
            <span>
              <strong>СКРЫТЬ ИМЯ. ПОКАЗЫВАТЬ КАК «АНОНИМ»</strong>
              <span>Имя скрыто везде в публичной ленте и библиотеке.</span>
            </span>
          </label>
        </div>

        {!isAnonymous && (
          <div className={styles.aliasField}>
            <label>
              <span className={styles.aliasLabel}>ПУБЛИЧНЫЙ ПСЕВДОНИМ</span>
              <input
                className={styles.aliasInput}
                placeholder="Как тебя называть в ленте"
                value={alias}
                onChange={(e) => setAlias(e.target.value.toUpperCase())}
                maxLength={30}
              />
            </label>
          </div>
        )}

        {saved && (
          <p className={styles.savedMsg}>СОХРАНЕНО.</p>
        )}

        <button className={styles.saveBtn} onClick={handleSave} disabled={isPending}>
          {isPending ? "СОХРАНЯЕМ…" : "СОХРАНИТЬ"}
        </button>
      </div>

      <div className={styles.links}>
        <a href="/feed" className={styles.link}>ЛЕНТА →</a>
        <a href="/library" className={styles.link}>ШАБЛОНЫ →</a>
      </div>
    </div>
  );
}
