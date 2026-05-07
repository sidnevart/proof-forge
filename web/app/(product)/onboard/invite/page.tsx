"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";

import { listCircles } from "@/lib/api";

import styles from "./page.module.css";

function OnboardInviteContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const circleId = searchParams.get("circle");

  const [memberCount, setMemberCount] = useState(1);
  const [inviteCode, setInviteCode] = useState<string | null>(null);
  const [circleName, setCircleName] = useState<string>("");
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    listCircles().then((res) => {
      const match = (res.circles ?? []).find((c) => String(c.circle.id) === circleId);
      if (match) {
        setInviteCode(match.circle.invite_code);
        setCircleName(match.circle.name);
        setMemberCount(match.members.length);
      }
    }).catch(() => {});
  }, [circleId]);

  useEffect(() => {
    if (!circleId) return;
    const id = setInterval(() => {
      listCircles().then((res) => {
        const match = (res.circles ?? []).find((c) => String(c.circle.id) === circleId);
        if (match) setMemberCount(match.members.length);
      }).catch(() => {});
    }, 10_000);
    return () => clearInterval(id);
  }, [circleId]);

  const inviteUrl = inviteCode
    ? `${typeof window !== "undefined" ? window.location.origin : ""}/dashboard?join=${inviteCode}`
    : "";

  const subtitle =
    memberCount <= 1
      ? "ОТПРАВЬ ССЫЛКУ. ОДИН ПРОТИВ ТИШИНЫ."
      : memberCount === 2
      ? "НУЖЕН ХОТЯ БЫ ЕЩЁ ОДИН."
      : "КРУГ ГОТОВ. ВРЕМЯ ПЕРВОГО ПРУФА.";

  const headline =
    memberCount <= 1
      ? "ЖДУ ДРУЗЕЙ."
      : memberCount === 2
      ? "ОДИН ПРИШЁЛ."
      : "КРУГ ГОТОВ.";

  async function handleCopy() {
    await navigator.clipboard.writeText(inviteUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  function handleShareTelegram() {
    const text = `Присоединяйся к кругу «${circleName}» в ProofForge → ${inviteUrl}`;
    window.open(`https://t.me/share/url?url=${encodeURIComponent(inviteUrl)}&text=${encodeURIComponent(text)}`);
  }

  async function handleNativeShare() {
    if (navigator.share) {
      await navigator.share({ title: `Круг «${circleName}»`, url: inviteUrl });
    }
  }

  function handleNext() {
    router.push(circleId ? `/goals/new?circle=${circleId}&onboard=1` : "/goals/new?onboard=1");
  }

  return (
    <main className={styles.page}>
      <div className={styles.content}>
        <div className={styles.eyebrow}>ОНБОРДИНГ · ШАГ 2 ИЗ 3</div>
        <h1 className={styles.title}>{headline}</h1>
        <p className={styles.sub}>{subtitle}</p>

        {inviteCode && (
          <div className={styles.codeBlock}>
            <div className={styles.codeLabel}>КОД ПРИГЛАШЕНИЯ</div>
            <div className={styles.code}>{inviteCode.toUpperCase()}</div>
          </div>
        )}

        <div className={styles.shareButtons}>
          <button className={styles.primaryBtn} onClick={handleCopy} type="button">
            {copied ? "СКОПИРОВАНО ✓" : "СКОПИРОВАТЬ ССЫЛКУ"}
          </button>
          <button className={styles.secondaryBtn} onClick={handleShareTelegram} type="button">
            ОТПРАВИТЬ В TELEGRAM
          </button>
          {typeof navigator !== "undefined" && "share" in navigator && (
            <button className={styles.secondaryBtn} onClick={handleNativeShare} type="button">
              ПОДЕЛИТЬСЯ
            </button>
          )}
        </div>

        <div className={styles.waitCounter}>
          <span className={styles.waitLabel}>ЖДУ ДРУЗЕЙ:</span>
          <span className={styles.waitCount}>{memberCount - 1} / 2</span>
        </div>

        <div className={styles.footer}>
          <button
            className={`${styles.nextBtn} ${memberCount < 3 ? styles.nextDisabled : ""}`}
            onClick={handleNext}
            disabled={memberCount < 3}
            type="button"
          >
            ДАЛЬШЕ →
          </button>
          {memberCount < 3 && (
            <p className={styles.waitNote}>Нужно хотя бы 2 участника кроме тебя.</p>
          )}
        </div>
      </div>
    </main>
  );
}

export default function OnboardInvitePage() {
  return (
    <Suspense>
      <OnboardInviteContent />
    </Suspense>
  );
}
