"use client";

import { type FormEvent, useCallback, useEffect, useState, useTransition } from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/core/button";
import { SectionShell } from "@/components/core/section-shell";
import { StatePanel } from "@/components/core/state-panel";
import { StatusPill } from "@/components/core/status-pill";
import { ApiError, acceptInvite, getInvite, registerUser } from "@/lib/api";
import type { InvitePreview } from "@/lib/types";

import styles from "./invite-accept-screen.module.css";

type ScreenState =
  | { kind: "loading" }
  | { kind: "not_found" }
  | { kind: "expired" }
  | { kind: "already_accepted" }
  | { kind: "error"; message: string }
  | { kind: "ready"; invite: InvitePreview }
  | { kind: "needs_auth"; invite: InvitePreview }
  | { kind: "accepted" };

type Props = { token: string };

export function InviteAcceptScreen({ token }: Props) {
  const router = useRouter();
  const [screenState, setScreenState] = useState<ScreenState>({ kind: "loading" });
  const [registerError, setRegisterError] = useState<string | null>(null);
  const [isAccepting, startAcceptTransition] = useTransition();
  const [isRegistering, startRegisterTransition] = useTransition();

  const loadInvite = useCallback(async () => {
    try {
      const data = await getInvite(token);
      setScreenState({ kind: "ready", invite: data.invite });
    } catch (error) {
      if (error instanceof ApiError) {
        if (error.status === 404) {
          setScreenState({ kind: "not_found" });
          return;
        }
        if (error.status === 410) {
          setScreenState({ kind: "expired" });
          return;
        }
      }
      setScreenState({
        kind: "error",
        message: error instanceof Error ? error.message : "Не удалось загрузить invite.",
      });
    }
  }, [token]);

  useEffect(() => {
    void loadInvite();
  }, [loadInvite]);

  function handleAccept() {
    startAcceptTransition(async () => {
      try {
        await acceptInvite(token);
        setScreenState({ kind: "accepted" });
        setTimeout(() => router.push("/dashboard"), 1500);
      } catch (error) {
        if (error instanceof ApiError && error.status === 401) {
          setScreenState((prev) =>
            prev.kind === "ready" ? { kind: "needs_auth", invite: prev.invite } : prev,
          );
          return;
        }
        if (error instanceof ApiError && error.status === 410) {
          setScreenState({ kind: "expired" });
          return;
        }
        if (error instanceof ApiError && error.status === 409) {
          setScreenState({ kind: "already_accepted" });
          return;
        }
        if (error instanceof ApiError && error.status === 403) {
          setScreenState((prev) =>
            prev.kind === "ready"
              ? {
                  kind: "error",
                  message: "Этот инвайт предназначен другому пользователю.",
                }
              : prev,
          );
          return;
        }
        setScreenState({
          kind: "error",
          message: error instanceof Error ? error.message : "Не удалось принять invite.",
        });
      }
    });
  }

  function handleRegisterAndAccept(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setRegisterError(null);
    const formData = new FormData(event.currentTarget);
    startRegisterTransition(async () => {
      try {
        await registerUser({
          email: String(formData.get("email") ?? ""),
          display_name: String(formData.get("display_name") ?? ""),
        });
        await acceptInvite(token);
        setScreenState({ kind: "accepted" });
        setTimeout(() => router.push("/dashboard"), 1500);
      } catch (error) {
        if (error instanceof ApiError && error.status === 403) {
          setRegisterError("Email должен совпадать с адресом, на который отправлен инвайт.");
          return;
        }
        setRegisterError(
          error instanceof Error ? error.message : "Не удалось зарегистрироваться.",
        );
      }
    });
  }

  if (screenState.kind === "loading") {
    return (
      <main className={styles.page}>
        <InviteHeader status="pending" label="Loading invite" />
        <StatePanel
          tone="loading"
          title="Загружаем данные pact"
          description="Проверяем токен и загружаем контекст accountability связки."
        />
      </main>
    );
  }

  if (screenState.kind === "not_found") {
    return (
      <main className={styles.page}>
        <InviteHeader status="rejected" label="Invite not found" />
        <StatePanel
          tone="error"
          title="Invite не найден"
          description="Ссылка недействительна или инвайт уже был отозван."
        />
      </main>
    );
  }

  if (screenState.kind === "expired") {
    return (
      <main className={styles.page}>
        <InviteHeader status="rejected" label="Invite expired" />
        <StatePanel
          tone="error"
          title="Invite истёк"
          description="Срок действия этой ссылки закончился. Попросите владельца goal создать новый invite."
        />
      </main>
    );
  }

  if (screenState.kind === "already_accepted") {
    return (
      <main className={styles.page}>
        <InviteHeader status="active" label="Already accepted" />
        <StatePanel
          tone="success"
          title="Pact уже принят"
          description="Вы уже приняли этот invite. Перейдите в dashboard, чтобы увидеть активный pact."
          meta={
            <Button variant="secondary" onClick={() => router.push("/dashboard")}>
              Открыть dashboard
            </Button>
          }
        />
      </main>
    );
  }

  if (screenState.kind === "accepted") {
    return (
      <main className={styles.page}>
        <InviteHeader status="active" label="Pact accepted" />
        <StatePanel
          tone="success"
          title="Pact принят — контур активирован"
          description="Вы вошли в accountability loop. Переходим на dashboard..."
        />
      </main>
    );
  }

  if (screenState.kind === "error") {
    return (
      <main className={styles.page}>
        <InviteHeader status="rejected" label="Error" />
        <StatePanel
          tone="error"
          title="Ошибка"
          description={screenState.message}
          meta={
            <Button variant="secondary" onClick={() => void loadInvite()}>
              Попробовать снова
            </Button>
          }
        />
      </main>
    );
  }

  if (screenState.kind === "needs_auth") {
    const { invite } = screenState;
    return (
      <main className={styles.page}>
        <InviteHeader status="pending" label="Auth required" />
        <div className={styles.grid}>
          <InviteCard invite={invite} />
          <SectionShell eyebrow="Identity required" title="Подтвердите участие">
            <p className={styles.authHint}>
              Чтобы принять pact, нужно войти или создать аккаунт с адресом{" "}
              <strong>{invite.invitee_email}</strong>.
            </p>
            <form className={styles.form} onSubmit={handleRegisterAndAccept}>
              <label className={styles.field}>
                <span>Ваше имя</span>
                <input name="display_name" placeholder="Как вас зовут" required />
              </label>
              <label className={styles.field}>
                <span>Email</span>
                <input
                  name="email"
                  type="email"
                  defaultValue={invite.invitee_email}
                  required
                />
              </label>
              {registerError ? (
                <p className={styles.formError} role="alert">
                  {registerError}
                </p>
              ) : null}
              <Button type="submit" disabled={isRegistering}>
                {isRegistering ? "Регистрируем и принимаем..." : "Войти и принять pact"}
              </Button>
            </form>
          </SectionShell>
        </div>
      </main>
    );
  }

  const { invite } = screenState;
  return (
    <main className={styles.page}>
      <InviteHeader status="pending" label="Invite pending" />
      <div className={styles.grid}>
        <InviteCard invite={invite} />
        <SectionShell eyebrow="Your decision" title="Принять accountability контур">
          <div className={styles.acceptBlock}>
            <p>
              <strong>{invite.owner_name}</strong> приглашает вас стать buddy для серьёзной цели. Вы
              будете подтверждать прогресс, а не просто наблюдать.
            </p>
            <ul className={styles.ruleList}>
              <li>Buddy review остаётся единственным источником правды о прогрессе.</li>
              <li>Принятие pact — это обязательство, а не мягкое согласие.</li>
              <li>Вы сможете запрашивать доработку или отклонять недостаточный check-in.</li>
            </ul>
            <Button onClick={handleAccept} disabled={isAccepting}>
              {isAccepting ? "Принимаем pact..." : "Принять pact"}
            </Button>
          </div>
        </SectionShell>
      </div>
    </main>
  );
}

function InviteHeader({ status, label }: { status: string; label: string }) {
  return (
    <header className={styles.header}>
      <div>
        <span className="eyebrow">Buddy invite</span>
        <h1>Вход в accountability loop</h1>
      </div>
      <StatusPill status={status as Parameters<typeof StatusPill>[0]["status"]} label={label} />
    </header>
  );
}

function InviteCard({ invite }: { invite: InvitePreview }) {
  return (
    <SectionShell eyebrow="Pact details" title={invite.goal_title}>
      <dl className={styles.inviteMeta}>
        <div>
          <dt>От кого</dt>
          <dd>{invite.owner_name}</dd>
        </div>
        <div>
          <dt>Для</dt>
          <dd>{invite.invitee_email}</dd>
        </div>
        <div>
          <dt>Invite истекает</dt>
          <dd>{formatDate(invite.expires_at)}</dd>
        </div>
        <div>
          <dt>Статус</dt>
          <dd>{invite.status === "pending" ? "Ожидает принятия" : "Принят"}</dd>
        </div>
      </dl>
    </SectionShell>
  );
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "long",
    year: "numeric",
  }).format(new Date(value));
}
