"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { type FormEvent, useEffect, useRef, useState, useTransition } from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/core/button";
import { SectionShell } from "@/components/core/section-shell";
import { StatePanel } from "@/components/core/state-panel";
import { StatusPill } from "@/components/core/status-pill";
import { GoalRefineSheet } from "@/components/product/goal-refine-sheet";
import { ApiError, createGoal, getDashboard, loginUser, registerUser } from "@/lib/api";
import type { GoalRefineVariant } from "@/lib/types";

import styles from "./goal-setup-screen.module.css";

type AuthState =
  | { kind: "checking" }
  | { kind: "authenticated" }
  | { kind: "unauthenticated" };

export function GoalSetupScreen() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [authState, setAuthState] = useState<AuthState>({ kind: "checking" });
  const [error, setError] = useState<string | null>(null);
  const [createdTitle, setCreatedTitle] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();

  const [loginError, setLoginError] = useState<string | null>(null);
  const [registerError, setRegisterError] = useState<string | null>(null);
  const [viewerEmail, setViewerEmail] = useState<string>("");
  const [refineOpen, setRefineOpen] = useState(false);
  const [refineDraft, setRefineDraft] = useState("");
  const [acceptedVariant, setAcceptedVariant] = useState<GoalRefineVariant | null>(null);
  const [acceptedCategory, setAcceptedCategory] = useState<string>("");
  const titleRef = useRef<HTMLInputElement>(null);
  const [isLoggingIn, startLoginTransition] = useTransition();
  const [isRegistering, startRegisterTransition] = useTransition();
  const titleParam = searchParams.get("title");
  const proofExamplesParam = searchParams.get("proof_examples");
  const categoryParam = searchParams.get("category");

  // Pre-fill from library template if query params are present.
  useEffect(() => {
    if (titleParam && titleRef.current) {
      titleRef.current.value = titleParam;
    }
    if (proofExamplesParam) {
      setAcceptedVariant({
        title: titleParam ?? "",
        smart: titleParam ?? "",
        proof_examples: proofExamplesParam.split("\n").filter(Boolean),
      });
    }
    if (categoryParam) {
      setAcceptedCategory(categoryParam);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    getDashboard()
      .then((dashboard) => {
        setViewerEmail(dashboard.user.email);
        setAuthState({ kind: "authenticated" });
      })
      .catch((err) => {
        // Only a real 401 means the visitor isn't logged in. Any other failure
        // (network, 5xx, timeout) is a transient issue — show it as an error
        // instead of falsely flipping into «authenticated», which used to let
        // users submit a goal that the backend would later reject.
        if (err instanceof ApiError && err.status === 401) {
          setAuthState({ kind: "unauthenticated" });
        } else {
          setError(
            err instanceof Error
              ? err.message
              : "Не удалось проверить авторизацию. Обновите страницу.",
          );
        }
      });
  }, []);

  function handleLoginSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoginError(null);
    const formData = new FormData(event.currentTarget);
    startLoginTransition(async () => {
      try {
        const email = String(formData.get("email") ?? "");
        await loginUser(email);
        setViewerEmail(email);
        setAuthState({ kind: "authenticated" });
      } catch (err) {
        if (err instanceof ApiError && err.status === 404) {
          setLoginError("Аккаунт не найден. Создайте его справа.");
        } else {
          setLoginError(err instanceof Error ? err.message : "Не удалось войти.");
        }
      }
    });
  }

  function handleRegisterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setRegisterError(null);
    const formData = new FormData(event.currentTarget);
    startRegisterTransition(async () => {
      try {
        const email = String(formData.get("email") ?? "");
        await registerUser({
          email,
          display_name: String(formData.get("display_name") ?? ""),
        });
        setViewerEmail(email);
        setAuthState({ kind: "authenticated" });
      } catch (err) {
        setRegisterError(err instanceof Error ? err.message : "Не удалось создать аккаунт.");
      }
    });
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);

    const formData = new FormData(event.currentTarget);
    const title = String(formData.get("title") ?? "");
    const buddyName = String(formData.get("buddy_name") ?? "");
    const buddyEmail = String(formData.get("buddy_email") ?? "");

    startTransition(async () => {
      try {
        const result = await createGoal({
          title,
          description: String(formData.get("description") ?? ""),
          buddy_name: buddyName,
          buddy_email: buddyEmail,
          circle_id: 0,
          proof_examples: acceptedVariant?.proof_examples.join("\n"),
          category: acceptedCategory || undefined,
        });
        setCreatedTitle(title);
        const acceptanceToken = result.goal.invite?.acceptance_token ?? "";
        const encodedTitle = encodeURIComponent(title);
        const encodedToken = encodeURIComponent(acceptanceToken);
        setTimeout(
          () => router.push(`/onboard/invite-buddy?token=${encodedToken}&title=${encodedTitle}`),
          800,
        );
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : "Не удалось создать круг.");
      }
    });
  }

  // suppress unused warning — viewerEmail used for display if needed in future
  void viewerEmail;

  if (authState.kind === "checking") {
    return (
      <main className={styles.page}>
        <header className={styles.header}>
          <h1 className={styles.headerTitle}>СОЗДАНИЕ КРУГА</h1>
          <StatusPill status="pending" label="Проверяем сессию" />
        </header>
        <StatePanel tone="loading" title="Секунду" description="Проверяем авторизацию." />
      </main>
    );
  }

  if (authState.kind === "unauthenticated") {
    return (
      <main className={styles.page}>
        <header className={styles.header}>
          <h1 className={styles.headerTitle}>ВОЙДИТЕ, ЧТОБЫ СОЗДАТЬ КРУГ</h1>
          <StatusPill status="pending" label="Нужна сессия" />
        </header>

        <section className={styles.authGrid}>
          <SectionShell eyebrow="Вход" title="Уже есть аккаунт">
            <form className={styles.form} onSubmit={handleLoginSubmit}>
              <label className={styles.field}>
                <span>Электронная почта</span>
                <input name="email" type="email" placeholder="your@email.com" required />
              </label>
              {loginError ? (
                <p className={styles.error} role="alert">{loginError}</p>
              ) : null}
              <Button type="submit" disabled={isLoggingIn}>
                {isLoggingIn ? "ВХОДИМ..." : "ВОЙТИ"}
              </Button>
            </form>
          </SectionShell>

          <SectionShell eyebrow="Регистрация" title="Создать аккаунт">
            <form className={styles.form} onSubmit={handleRegisterSubmit}>
              <label className={styles.field}>
                <span>Ваше имя</span>
                <input name="display_name" placeholder="Например, Артём" required />
              </label>
              <label className={styles.field}>
                <span>Электронная почта</span>
                <input name="email" type="email" placeholder="you@example.com" required />
              </label>
              {registerError ? (
                <p className={styles.error} role="alert">{registerError}</p>
              ) : null}
              <Button type="submit" disabled={isRegistering}>
                {isRegistering ? "СОЗДАЁМ АККАУНТ..." : "СОЗДАТЬ АККАУНТ"}
              </Button>
            </form>
          </SectionShell>

          <div className={styles.authNote}>
            <Link href="/dashboard" className={styles.backLink}>← На главную</Link>
          </div>
        </section>
      </main>
    );
  }

  return (
    <main className={styles.page}>
      <header className={styles.headerCompact}>
        <p className={styles.eyebrow}>ОНБОРДИНГ · ШАГ 1 ИЗ 2</p>
        <h1>ОПИШИ КРУГ. ПРИГЛАСИ ПАРТНЁРА.</h1>
      </header>

      {createdTitle ? (
        <StatePanel
          tone="success"
          title={`Круг «${createdTitle}» создан`}
          description="Переходим к приглашению партнёра..."
        />
      ) : null}

      <section className={styles.grid}>
        <SectionShell eyebrow="Шаг 1" title="Опишите круг">
          <form className={styles.form} onSubmit={handleSubmit}>
            <label className={styles.field}>
              <span>Название круга</span>
              <input
                ref={titleRef}
                name="title"
                defaultValue={acceptedVariant?.smart ?? ""}
                placeholder="Например, запустить новый лендинг"
                required
              />
            </label>
            <button
              type="button"
              className={styles.refineBtn}
              onClick={() => {
                const draft = titleRef.current?.value?.trim() ?? "";
                if (draft.length >= 3) {
                  setRefineDraft(draft);
                  setRefineOpen(true);
                }
              }}
            >
              ⚡ УТОЧНИТЬ КРУГ
            </button>
            {acceptedVariant && (
              <div className={styles.proofExamplesHint}>
                <span className={styles.proofHintLabel}>ЧТО СЧИТАЕТСЯ ПРУФОМ:</span>
                <ul className={styles.proofHintList}>
                  {acceptedVariant.proof_examples.map((ex, i) => (
                    <li key={i}>{ex}</li>
                  ))}
                </ul>
                <button
                  type="button"
                  className={styles.clearVariantBtn}
                  onClick={() => setAcceptedVariant(null)}
                >
                  СБРОСИТЬ
                </button>
              </div>
            )}
            <label className={styles.field}>
              <span>Что должно считаться прогрессом</span>
              <textarea
                name="description"
                rows={5}
                placeholder="Опишите, какие материалы, результаты или изменения вы будете отправлять на проверку."
              />
            </label>
            <div className={styles.inlineFields}>
              <label className={styles.field}>
                <span>Имя партнёра</span>
                <input name="buddy_name" placeholder="Например, Мария" required />
              </label>
              <label className={styles.field}>
                <span>Почта партнёра</span>
                <input name="buddy_email" type="email" placeholder="partner@example.com" required />
              </label>
            </div>
            {error ? (
              <p className={styles.error} role="alert">{error}</p>
            ) : null}
            <div className={styles.actions}>
              <Button type="submit" disabled={isPending}>
                {isPending ? "СОЗДАЁМ КРУГ..." : "СОЗДАТЬ КРУГ"}
              </Button>
              <Link className={styles.backLink} href="/dashboard">
                НА ГЛАВНУЮ
              </Link>
            </div>
          </form>
        </SectionShell>

        <SectionShell eyebrow="Шаг 2" title="Что произойдёт дальше">
          {/*
            Numbered timeline replaces the old bullet list — it gives the
            right column comparable visual weight to the form on its left,
            and the steps map 1:1 to the user's actual next actions.
          */}
          <ol className={styles.timeline}>
            <li>Создаёшь круг — он сразу появляется в твоём дашборде.</li>
            <li>Копируешь ссылку и отправляешь партнёру.</li>
            <li>Партнёр переходит по ссылке и принимает приглашение.</li>
            <li>Круг становится активным — и ты сдаёшь пруфы каждый день.</li>
          </ol>
        </SectionShell>
      </section>

      {refineOpen && refineDraft && (
        <GoalRefineSheet
          draftText={refineDraft}
          onAccept={(variant, category) => {
            setAcceptedVariant(variant);
            setAcceptedCategory(category);
            if (titleRef.current) {
              titleRef.current.value = variant.smart;
            }
            setRefineOpen(false);
          }}
          onClose={() => setRefineOpen(false)}
        />
      )}
    </main>
  );
}
