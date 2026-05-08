"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, createContract } from "@/lib/api";
import type { GoalView } from "@/lib/types";
import styles from "./proof-contract-screen.module.css";

// ── Types ────────────────────────────────────────────────────────────────────

type ContractPath = "single" | "rhythm" | "initiative";

type ProofKind = "artifact" | "note" | "demo" | "experiment";

interface FormData {
  path: ContractPath | null;
  rhythm: string | null;
  whatToProve: string;
  proofKind: ProofKind | null;
  dueAt: Date | null;
  buddyChoice: "existing" | "other" | "none";
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function endOfToday(): Date {
  const d = new Date();
  d.setHours(23, 59, 59, 0);
  return d;
}

function addDays(d: Date, n: number): Date {
  const r = new Date(d);
  r.setDate(r.getDate() + n);
  return r;
}

function nextFriday(): Date {
  const d = new Date();
  const day = d.getDay(); // 0=Sun
  const diff = (5 - day + 7) % 7 || 7;
  return addDays(d, diff);
}

function formatDate(d: Date): string {
  return d.toLocaleDateString("ru-RU", { day: "numeric", month: "long" });
}

function toISO(d: Date): string {
  return d.toISOString();
}

function proofKindLabel(k: ProofKind): string {
  const m: Record<ProofKind, string> = {
    artifact: "Артефакт",
    note: "Заметка",
    demo: "Демо",
    experiment: "Эксперимент",
  };
  return m[k];
}

// ── Props ─────────────────────────────────────────────────────────────────────

interface Props {
  goalId: number;
  goal: GoalView;
}

// ── Component ─────────────────────────────────────────────────────────────────

export function ProofContractScreen({ goalId, goal }: Props) {
  const router = useRouter();
  const hasBuddy = goal.pact?.status === "active";
  const buddyName = goal.buddy?.display_name || goal.buddy?.email || "";

  const [form, setForm] = useState<FormData>({
    path: null,
    rhythm: null,
    whatToProve: "",
    proofKind: null,
    dueAt: null,
    buddyChoice: hasBuddy ? "existing" : "none",
  });
  const [step, setStep] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Steps depend on selected path
  type StepId = "path" | "rhythm" | "what" | "kind" | "when" | "who";

  function getSteps(): StepId[] {
    if (!form.path) return ["path"];
    if (form.path === "rhythm") return ["path", "rhythm", "what", "kind", "when", "who"];
    if (form.path === "initiative") return ["path", "what", "when", "who"];
    return ["path", "what", "kind", "when", "who"]; // single
  }

  const steps = getSteps();
  const currentStepId = steps[step];

  function advance() {
    setStep((s) => Math.min(s + 1, steps.length - 1));
  }

  function selectPath(p: ContractPath) {
    setForm((f) => ({ ...f, path: p }));
    setStep(1);
  }

  function selectRhythm(r: string) {
    setForm((f) => ({ ...f, rhythm: r }));
    advance();
  }

  function selectProofKind(k: ProofKind) {
    setForm((f) => ({ ...f, proofKind: k }));
    advance();
  }

  function selectDate(d: Date) {
    setForm((f) => ({ ...f, dueAt: d }));
    advance();
  }

  function selectBuddy(choice: FormData["buddyChoice"]) {
    setForm((f) => ({ ...f, buddyChoice: choice }));
  }

  const isLastStep = step === steps.length - 1;
  const canSubmit =
    form.whatToProve.trim().length > 0 &&
    form.dueAt !== null &&
    !submitting;

  async function handleSubmit() {
    if (!canSubmit) return;
    setError(null);
    setSubmitting(true);
    try {
      const howToProve = form.proofKind ? proofKindLabel(form.proofKind) : "";
      const buddyId =
        form.buddyChoice === "existing" && hasBuddy && goal.buddy?.id
          ? goal.buddy.id
          : undefined;
      await createContract(goalId, {
        what_to_prove: form.whatToProve.trim(),
        how_to_prove: howToProve,
        due_at: toISO(form.dueAt!),
        buddy_user_id: buddyId,
      });
      router.push(`/goals/${goalId}?toast=contract_created`);
    } catch (err) {
      const msg = err instanceof ApiError ? err.message : "Не удалось создать контракт.";
      setError(msg);
      setSubmitting(false);
    }
  }

  const summaryLine = (() => {
    const parts: string[] = [];
    if (form.dueAt) parts.push(`До ${formatDate(form.dueAt)}`);
    if (form.proofKind) parts.push(proofKindLabel(form.proofKind));
    if (form.buddyChoice === "existing" && buddyName) parts.push(`${buddyName} подтвердит`);
    return parts.join(" · ");
  })();

  return (
    <div className={styles.screen}>
      {/* Step: path selection */}
      {currentStepId === "path" && (
        <>
          <p className={styles.eyebrow}>Что хочешь сделать?</p>
          <button
            type="button"
            className={`${styles.pathCard} ${styles.primary}`}
            onClick={() => selectPath("single")}
          >
            <span className={styles.pathArrow}>→</span>
            <span className={styles.pathText}>
              <span className={styles.pathTitle}>Сдать разовый пруф</span>
              <span className={styles.pathDesc}>Один конкретный результат</span>
            </span>
          </button>
          <button
            type="button"
            className={styles.pathCard}
            onClick={() => selectPath("rhythm")}
          >
            <span className={styles.pathArrow}>→</span>
            <span className={styles.pathText}>
              <span className={styles.pathTitle}>Настроить регулярный ритм</span>
              <span className={styles.pathDesc}>Сдавать регулярно по расписанию</span>
            </span>
          </button>
          <button
            type="button"
            className={styles.pathCard}
            onClick={() => selectPath("initiative")}
          >
            <span className={styles.pathArrow}>→</span>
            <span className={styles.pathText}>
              <span className={styles.pathTitle}>Подключиться к инициативе</span>
              <span className={styles.pathDesc}>Рабочие проекты и исследования</span>
            </span>
          </button>
        </>
      )}

      {/* Step: rhythm */}
      {currentStepId === "rhythm" && (
        <>
          <p className={styles.eyebrow}>Как часто?</p>
          <div className={styles.optionGrid}>
            {[
              { value: "daily", label: "Раз в день" },
              { value: "weekly", label: "Раз в неделю", recommended: true },
              { value: "biweekly", label: "Раз в две недели" },
              { value: "friday", label: "По пятницам" },
            ].map((o) => (
              <button
                key={o.value}
                type="button"
                className={`${styles.optionCard} ${form.rhythm === o.value ? styles.selected : ""} ${o.recommended ? styles.recommended : ""}`}
                onClick={() => selectRhythm(o.value)}
              >
                {o.recommended && <span className={styles.recommendedBadge}>рекомендовано</span>}
                <span className={styles.optionLabel}>{o.label}</span>
              </button>
            ))}
          </div>
        </>
      )}

      {/* Step: what to prove */}
      {currentStepId === "what" && (
        <>
          <p className={styles.eyebrow}>Что ты докажешь?</p>
          <textarea
            className={styles.textarea}
            value={form.whatToProve}
            onChange={(e) => setForm((f) => ({ ...f, whatToProve: e.target.value }))}
            placeholder='Например: "Покажу мини-пример с SupervisorJob и объясню где применимо в нашем сервисе"'
            rows={5}
          />
          <button type="button" className={styles.aiBtn}>
            Предложить формулировку
          </button>
          <button
            type="button"
            className={styles.nextBtn}
            disabled={form.whatToProve.trim().length < 5}
            onClick={advance}
          >
            Далее →
          </button>
        </>
      )}

      {/* Step: proof kind */}
      {currentStepId === "kind" && (
        <>
          <p className={styles.eyebrow}>Чем докажешь?</p>
          <div className={styles.optionGrid}>
            {(
              [
                { value: "artifact", label: "Артефакт", desc: "MR, файл, документ" },
                { value: "note", label: "Заметка", desc: "Текстовый отчёт" },
                { value: "demo", label: "Демо", desc: "Видео, скрин" },
                { value: "experiment", label: "Эксперимент", desc: "Результат исследования" },
              ] as Array<{ value: ProofKind; label: string; desc: string }>
            ).map((o) => (
              <button
                key={o.value}
                type="button"
                className={`${styles.optionCard} ${form.proofKind === o.value ? styles.selected : ""}`}
                onClick={() => selectProofKind(o.value)}
              >
                <span className={styles.optionLabel}>{o.label}</span>
                <span className={styles.optionDesc}>{o.desc}</span>
              </button>
            ))}
          </div>
        </>
      )}

      {/* Step: when */}
      {currentStepId === "when" && (
        <>
          <p className={styles.eyebrow}>Когда покажешь?</p>
          <div className={styles.dateGrid}>
            {[
              { label: "Сегодня", date: endOfToday() },
              { label: "Завтра", date: addDays(endOfToday(), 1) },
              { label: "Эта пятница", date: nextFriday() },
              { label: "Через неделю", date: addDays(new Date(), 7) },
            ].map(({ label, date }) => (
              <button
                key={label}
                type="button"
                className={`${styles.dateBtn} ${form.dueAt?.toDateString() === date.toDateString() ? styles.selected : ""}`}
                onClick={() => selectDate(date)}
              >
                <span className={styles.dateBtnLabel}>{label}</span>
                <span className={styles.dateBtnDate}>{formatDate(date)}</span>
              </button>
            ))}
          </div>
          <label className={styles.datePickerLabel}>
            <span className={styles.datePickerText}>Другая дата →</span>
            <input
              type="date"
              className={styles.datePicker}
              min={new Date().toISOString().slice(0, 10)}
              onChange={(e) => {
                if (e.target.value) {
                  selectDate(new Date(e.target.value + "T23:59:59"));
                }
              }}
            />
          </label>
        </>
      )}

      {/* Step: who confirms */}
      {currentStepId === "who" && (
        <>
          <p className={styles.eyebrow}>Кто подтвердит?</p>
          {hasBuddy && (
            <button
              type="button"
              className={`${styles.pathCard} ${form.buddyChoice === "existing" ? styles.primary : ""}`}
              onClick={() => selectBuddy("existing")}
            >
              <span className={styles.pathArrow}>→</span>
              <span className={styles.pathText}>
                <span className={styles.pathTitle}>Мой бадди — {buddyName}</span>
              </span>
            </button>
          )}
          <button
            type="button"
            className={`${styles.pathCard} ${form.buddyChoice === "none" ? styles.primary : ""}`}
            onClick={() => selectBuddy("none")}
          >
            <span className={styles.pathArrow}>→</span>
            <span className={styles.pathText}>
              <span className={styles.pathTitle}>Без подтверждения</span>
            </span>
          </button>
        </>
      )}

      {/* Submit bar (always visible on last step) */}
      {isLastStep && (
        <div className={styles.submitBar}>
          {summaryLine && (
            <p className={styles.summaryLine}>{summaryLine}</p>
          )}
          {error && <p className={styles.formError}>{error}</p>}
          <button
            type="button"
            className={styles.submitBtn}
            disabled={!canSubmit}
            onClick={handleSubmit}
          >
            {submitting ? "Создаём…" : "Создать контракт"}
          </button>
        </div>
      )}
    </div>
  );
}
