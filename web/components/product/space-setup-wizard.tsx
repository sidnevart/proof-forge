"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import styles from "./space-setup-wizard.module.css";

// ── Types ────────────────────────────────────────────────────────────────────

type TemplateId =
  | "ipr_team"
  | "work_initiative"
  | "learning"
  | "community_challenge"
  | "sport_challenge"
  | "science_circle"
  | "interview_prep"
  | "ai_stream";

type SpaceType = "teamspace" | "community_space" | "circle";
type RhythmOption = "daily" | "3x_week" | "weekly" | "biweekly" | "custom" | "7d" | "14d" | "28d" | "42d";
type VisibilityOption = "buddy_only" | "circle" | "leader_only" | "leader_members" | "public_showcase" | "aggregated";
type MetricsMode = "learning" | "work_ipr" | "competition" | "mentoring" | "paid_community";

interface TemplateDefaults {
  rhythm: RhythmOption;
  visibility: VisibilityOption;
  spaceType: SpaceType | null;
}

const TEMPLATE_DEFAULTS: Record<TemplateId, TemplateDefaults> = {
  ipr_team:            { rhythm: "weekly",    visibility: "leader_only",     spaceType: "teamspace" },
  work_initiative:     { rhythm: "custom",    visibility: "leader_members",  spaceType: "teamspace" },
  learning:            { rhythm: "weekly",    visibility: "circle",          spaceType: null },
  community_challenge: { rhythm: "28d",       visibility: "public_showcase", spaceType: "community_space" },
  sport_challenge:     { rhythm: "daily",     visibility: "public_showcase", spaceType: "community_space" },
  science_circle:      { rhythm: "biweekly",  visibility: "circle",          spaceType: "community_space" },
  interview_prep:      { rhythm: "weekly",    visibility: "buddy_only",      spaceType: "circle" },
  ai_stream:           { rhythm: "weekly",    visibility: "leader_members",  spaceType: "teamspace" },
};

interface Option {
  value: string;
  title: string;
  desc?: string;
  recommended?: boolean;
}

interface WizardState {
  template: TemplateId | null;
  spaceType: SpaceType | null;
  rhythm: RhythmOption | null;
  visibility: VisibilityOption | null;
  metricsMode: MetricsMode | null;
  name: string;
}

interface Props {
  workspaceId?: number;
  workspaceType?: "organization" | "community";
}

function makeSlug(name: string): string {
  const base = name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9а-яё]+/gi, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 48);
  return `${base || "space"}-${Date.now().toString(36)}`;
}

// ── Option data ───────────────────────────────────────────────────────────────

const TEMPLATES: Option[] = [
  { value: "ipr_team",            title: "ИПР команды",                 desc: "Цели и прогресс сотрудников" },
  { value: "work_initiative",     title: "Рабочая инициатива",          desc: "Исследования и вклад в проекты" },
  { value: "learning",            title: "Обучение навыку",             desc: "Курс, тема или технология" },
  { value: "community_challenge", title: "Community Challenge",         desc: "Групповой вызов на срок" },
  { value: "sport_challenge",     title: "Спортивный вызов",            desc: "" },
  { value: "science_circle",      title: "Научный кружок",              desc: "" },
  { value: "interview_prep",      title: "Подготовка к собеседованию",  desc: "" },
  { value: "ai_stream",           title: "Внутренний AI-стрим",         desc: "" },
];

const SPACE_TYPES_ORG: Option[] = [
  { value: "teamspace",       title: "Teamspace",       desc: "Для команды в компании" },
  { value: "circle",          title: "Круг",            desc: "Маленькая группа 3–8 человек" },
];

const SPACE_TYPES_COMMUNITY: Option[] = [
  { value: "community_space", title: "Community Space", desc: "Для сообщества, клуба или группы" },
  { value: "circle",          title: "Круг",            desc: "Маленькая группа 3–8 человек" },
];

const SPACE_TYPES_ALL: Option[] = [
  { value: "teamspace",       title: "Teamspace",       desc: "Для команды в компании" },
  { value: "community_space", title: "Community Space", desc: "Для сообщества, клуба или группы" },
  { value: "circle",          title: "Круг",            desc: "Маленькая группа 3–8 человек" },
];

const RHYTHM_OPTIONS: Option[] = [
  { value: "daily",    title: "Раз в день",     desc: "" },
  { value: "weekly",   title: "Раз в неделю",   desc: "", recommended: true },
  { value: "3x_week",  title: "3 раза в неделю", desc: "" },
  { value: "custom",   title: "Свой ритм",      desc: "" },
  { value: "7d",       title: "7 дней",          desc: "Марафон" },
  { value: "14d",      title: "14 дней",         desc: "" },
  { value: "28d",      title: "28 дней",         desc: "" },
  { value: "42d",      title: "42 дня",          desc: "" },
];

const VISIBILITY_OPTIONS: Option[] = [
  { value: "buddy_only",      title: "Только я и бадди",       desc: "" },
  { value: "circle",          title: "Только круг",            desc: "" },
  { value: "leader_only",     title: "Только лидер",           desc: "" },
  { value: "leader_members",  title: "Лидер + участники",      desc: "" },
  { value: "public_showcase", title: "Витрина лучших",         desc: "Публично" },
  { value: "aggregated",      title: "Только статистика",      desc: "Агрегированно" },
];

const METRICS_OPTIONS: Option[] = [
  { value: "learning",       title: "Обучение",         desc: "Навыки и прогресс" },
  { value: "work_ipr",       title: "Работа / ИПР",     desc: "Вклад в инициативы" },
  { value: "competition",    title: "Соревнование",     desc: "Лидерборды и витрина" },
  { value: "mentoring",      title: "Наставничество",   desc: "Помощь другим" },
  { value: "paid_community", title: "Платное сообщество", desc: "Retention и конверсия" },
];

// ── Step definitions ──────────────────────────────────────────────────────────

interface StepDef {
  id: string;
  title: string;
  options: Option[];
  field: keyof Omit<WizardState, "name">;
  skippable: boolean;
}

function buildSteps(workspaceType?: "organization" | "community", templateId?: TemplateId | null): StepDef[] {
  const steps: StepDef[] = [
    {
      id: "template",
      title: "Выбери шаблон",
      options: TEMPLATES,
      field: "template",
      skippable: false,
    },
  ];

  // Step 2 (type) — skip if workspace narrows choices
  if (!templateId || TEMPLATE_DEFAULTS[templateId].spaceType === null) {
    const typeOpts = workspaceType === "organization"
      ? SPACE_TYPES_ORG
      : workspaceType === "community"
        ? SPACE_TYPES_COMMUNITY
        : SPACE_TYPES_ALL;
    steps.push({
      id: "spaceType",
      title: "Что создаём?",
      options: typeOpts,
      field: "spaceType",
      skippable: true,
    });
  }

  steps.push(
    {
      id: "rhythm",
      title: "Как часто сдавать пруфы?",
      options: RHYTHM_OPTIONS,
      field: "rhythm",
      skippable: true,
    },
    {
      id: "visibility",
      title: "Кто видит пруфы?",
      options: VISIBILITY_OPTIONS,
      field: "visibility",
      skippable: true,
    },
    {
      id: "metricsMode",
      title: "Что важнее измерять?",
      options: METRICS_OPTIONS,
      field: "metricsMode",
      skippable: true,
    },
  );

  return steps;
}

// ── Option card ───────────────────────────────────────────────────────────────

function OptionCard({
  option,
  selected,
  onClick,
}: {
  option: Option;
  selected: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      className={`${styles.optionCard} ${selected ? styles.selected : ""} ${option.recommended ? styles.recommended : ""}`}
      onClick={onClick}
    >
      {option.recommended && (
        <span className={styles.recommendedBadge}>рекомендовано</span>
      )}
      <span className={styles.optionTitle}>{option.title}</span>
      {option.desc && <span className={styles.optionDesc}>{option.desc}</span>}
    </button>
  );
}

// ── Main component ────────────────────────────────────────────────────────────

export function SpaceSetupWizard({ workspaceId, workspaceType }: Props) {
  const router = useRouter();
  const [stepIdx, setStepIdx] = useState(0);
  const [state, setState] = useState<WizardState>({
    template: null,
    spaceType: null,
    rhythm: null,
    visibility: null,
    metricsMode: null,
    name: "",
  });
  const [submitting, setSubmitting] = useState(false);
  const [copyDone, setCopyDone] = useState(false);

  const steps = buildSteps(workspaceType, state.template);
  const isLastStep = stepIdx === steps.length;
  const progress = steps.length > 0 ? (stepIdx / steps.length) * 100 : 0;

  function applyTemplateDefaults(templateId: TemplateId) {
    const d = TEMPLATE_DEFAULTS[templateId];
    setState((prev) => ({
      ...prev,
      template: templateId,
      rhythm: d.rhythm,
      visibility: d.visibility,
      spaceType: d.spaceType ?? prev.spaceType,
    }));
  }

  function handleOptionSelect(field: keyof Omit<WizardState, "name">, value: string) {
    if (field === "template") {
      applyTemplateDefaults(value as TemplateId);
    } else {
      setState((prev) => ({ ...prev, [field]: value }));
    }
    // Auto-advance to next step
    const nextIdx = stepIdx + 1;
    if (nextIdx <= steps.length) {
      setStepIdx(nextIdx);
    }
  }

  function skipStep() {
    setStepIdx((i) => Math.min(i + 1, steps.length));
  }

  async function handleCreate(later = false) {
    if (!state.name.trim() || submitting) return;
    setSubmitting(true);
    try {
      const name = state.name.trim();
      const spaceType = state.spaceType ?? "circle";
      let res: Response;

      if (spaceType === "teamspace") {
        res = await fetch("/v1/teams", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({ name, ai_mode: "metadata-only" }),
        });
        if (res.ok && !later) {
          const data = await res.json();
          router.push(`/spaces/teamspace/${data.data?.team?.id ?? ""}`);
          return;
        }
      } else if (spaceType === "community_space") {
        res = await fetch("/v1/community-spaces", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({
            name,
            slug: makeSlug(name),
            description: "",
            is_public: state.visibility === "public_showcase",
            workspace_id: workspaceId,
          }),
        });
        if (res.ok && !later) {
          const data = await res.json();
          router.push(`/spaces/community/${data.id ?? ""}`);
          return;
        }
      } else {
        res = await fetch("/v1/circles", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({ name }),
        });
        if (res.ok && !later) {
          router.push("/dashboard");
          return;
        }
      }

      if (!res.ok || later) {
        router.push("/dashboard");
      }
    } catch {
      router.push("/dashboard");
    }
  }

  async function handleCopyLink() {
    const link = `${window.location.origin}/spaces/new?workspace_id=${workspaceId ?? ""}`;
    await navigator.clipboard.writeText(link);
    setCopyDone(true);
    setTimeout(() => setCopyDone(false), 2000);
  }

  const currentStep = steps[stepIdx];

  return (
    <div className={styles.wizard}>
      {/* Progress bar */}
      <div className={styles.progressBar}>
        <div className={styles.progressFill} style={{ width: `${progress}%` }} />
      </div>

      {!isLastStep && currentStep && (
        <>
          <div className={styles.stepHeader}>
            <p className={styles.stepCounter}>
              Шаг {stepIdx + 1} из {steps.length + 1}
            </p>
            <h2 className={styles.stepTitle}>{currentStep.title}</h2>
          </div>

          <div className={styles.optionGrid}>
            {currentStep.options.map((opt) => (
              <OptionCard
                key={opt.value}
                option={{
                  ...opt,
                  recommended:
                    opt.recommended ||
                    (state[currentStep.field] as string) === opt.value,
                }}
                selected={(state[currentStep.field] as string) === opt.value}
                onClick={() => handleOptionSelect(currentStep.field, opt.value)}
              />
            ))}
          </div>

          {currentStep.skippable && (
            <button type="button" className={styles.skipBtn} onClick={skipStep}>
              Пропустить
            </button>
          )}
        </>
      )}

      {isLastStep && (
        <>
          <div className={styles.stepHeader}>
            <p className={styles.stepCounter}>
              Шаг {steps.length + 1} из {steps.length + 1}
            </p>
            <h2 className={styles.stepTitle}>Придумай название</h2>
          </div>

          <input
            type="text"
            className={styles.nameInput}
            value={state.name}
            onChange={(e) => setState((prev) => ({ ...prev, name: e.target.value }))}
            placeholder="Название пространства"
            maxLength={80}
            autoFocus
          />

          <div className={styles.launchActions}>
            <button
              type="button"
              className={styles.primaryLaunchBtn}
              disabled={state.name.trim().length < 2 || submitting}
              onClick={() => handleCreate(false)}
            >
              {submitting ? "Создаём…" : "Создать пространство"}
            </button>
            <button
              type="button"
              className={styles.secondaryLaunchBtn}
              onClick={handleCopyLink}
            >
              {copyDone ? "✓ Ссылка скопирована" : "Скопировать ссылку приглашения"}
            </button>
            <button
              type="button"
              className={styles.skipBtn}
              onClick={() => handleCreate(true)}
            >
              Запустить позже
            </button>
          </div>
        </>
      )}
    </div>
  );
}
