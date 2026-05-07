"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useState } from "react";

import styles from "./page.module.css";

export default function InviteBuddyPage() {
  const searchParams = useSearchParams();
  const token = searchParams.get("token") ?? "";
  const title = searchParams.get("title") ?? "";
  const [copied, setCopied] = useState(false);

  const origin = typeof window !== "undefined" ? window.location.origin : "";
  const inviteLink = `${origin}/invites/${token}`;

  function handleCopy() {
    navigator.clipboard.writeText(inviteLink).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <main className={styles.page}>
      <article className={styles.card}>
        <p className={styles.eyebrow}>ШАГ 2 ИЗ 2 · ПРИГЛАСИ ПАРТНЁРА</p>
        <h1 className={styles.heading}>
          {title ? `«${title}» СОЗДАН!` : "КРУГ СОЗДАН!"} ТЕПЕРЬ ПРИГЛАСИ ПАРТНЁРА
        </h1>

        <div className={styles.urlBox}>
          <span className={styles.urlText}>{inviteLink}</span>
          <button
            type="button"
            className={styles.copyBtn}
            onClick={handleCopy}
            aria-label="Скопировать ссылку"
          >
            {copied ? "СКОПИРОВАНО ✓" : "СКОПИРОВАТЬ ССЫЛКУ"}
          </button>
        </div>

        <p className={styles.subtext}>
          Отправь эту ссылку своему партнёру — он должен принять приглашение, чтобы круг стал активным.
        </p>

        <Link href="/dashboard" className={styles.ctaSolid}>
          ПЕРЕЙТИ В ДАШБОРД →
        </Link>
      </article>
    </main>
  );
}
