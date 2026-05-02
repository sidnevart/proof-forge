"use client";

import { type FormEvent, useCallback, useEffect, useState, useTransition } from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/core/button";
import { SectionShell } from "@/components/core/section-shell";
import { StatePanel } from "@/components/core/state-panel";
import { StatusPill } from "@/components/core/status-pill";
import { ApiError, acceptInvite, getInvite, registerUser } from "@/lib/api";
import { formatDateLabel } from "@/lib/ui-copy";
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
        message: error instanceof Error ? error.message : "Не удалось загрузить приглашение.",
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
          message: error instanceof Error ? error.message : "Не удалось принять приглашение.",
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
        <InviteHeader status="pending" label="Загружаем приглашение" />
        <StatePanel
          tone="loading"
          title="Загружаем приглашение"
          description="Проверяем ссылку и собираем контекст по цели."
        />
      </main>
    );
  }

  if (screenState.kind === "not_found") {
    return (
      <main className={styles.page}>
        <InviteHeader status="rejected" label="Приглашение не найдено" />
        <StatePanel
          tone="error"
          title="Приглашение не найдено"
          description="Ссылка недействительна или приглашение уже было отозвано."
        />
      </main>
    );
  }

  if (screenState.kind === "expired") {
    return (
      <main className={styles.page}>
        <InviteHeader status="rejected" label="Срок приглашения истёк" />
        <StatePanel
          tone="error"
          title="Срок приглашения истёк"
          description="Ссылка больше не действует. Попросите владельца цели отправить новое приглашение."
        />
      </main>
    );
  }

  if (screenState.kind === "already_accepted") {
    return (
      <main className={styles.page}>
        <InviteHeader status="active" label="Уже принято" />
        <StatePanel
          tone="success"
          title="Приглашение уже принято"
          description="Вы уже подключены к этой цели. Вернитесь в центр управления, чтобы увидеть активный цикл."
          meta={
            <Button variant="secondary" onClick={() => router.push("/dashboard")}>
              Открыть центр управления
            </Button>
          }
        />
      </main>
    );
  }

  if (screenState.kind === "accepted") {
    return (
      <main className={styles.page}>
        <InviteHeader status="active" label="Принято" />
        <StatePanel
          tone="success"
          title="Приглашение принято"
          description="Вы подключены к цели. Переходим в центр управления..."
        />
      </main>
    );
  }

  if (screenState.kind === "error") {
    return (
      <main className={styles.page}>
        <InviteHeader status="rejected" label="Ошибка" />
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
        <InviteHeader status="pending" label="Нужно подтвердить личность" />
        <div className={styles.grid}>
          <InviteCard invite={invite} />
          <SectionShell eyebrow="Вход или регистрация" title="Подтвердите участие">
            <p className={styles.authHint}>
              Чтобы принять приглашение, войдите или создайте аккаунт с адресом{" "}
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
                {isRegistering ? "Подключаем..." : "Войти и принять приглашение"}
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
      <InviteHeader status="pending" label="Ожидает принятия" />
      <div className={styles.grid}>
        <InviteCard invite={invite} />
        <SectionShell eyebrow="Ваше решение" title="Принять приглашение к цели">
          <div className={styles.acceptBlock}>
            <p>
              <strong>{invite.owner_name}</strong> приглашает вас участвовать в цели как
              внешний проверяющий. Вы будете видеть подтверждения движения и принимать
              решение по прогрессу.
            </p>
            <ul className={styles.ruleList}>
              <li>Вы подтверждаете, что движение по цели действительно произошло.</li>
              <li>Если подтверждения недостаточно, вы можете вернуть его на доработку.</li>
              <li>Если движение не подтверждается, вы можете отклонить результат.</li>
            </ul>
            <Button onClick={handleAccept} disabled={isAccepting}>
              {isAccepting ? "Принимаем приглашение..." : "Принять приглашение"}
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
        <span className="eyebrow">Приглашение к цели</span>
        <h1>Присоединиться к цели</h1>
      </div>
      <StatusPill status={status as Parameters<typeof StatusPill>[0]["status"]} label={label} />
    </header>
  );
}

function InviteCard({ invite }: { invite: InvitePreview }) {
  const sectionTitle =
    invite.status === "pending" ? "Карточка приглашения" : "Приглашение принято";

  return (
    <SectionShell eyebrow="Детали цели" title={invite.goal_title}>
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
          <dt>{sectionTitle}</dt>
          <dd>{invite.status === "pending" ? "Ожидает вашего решения" : "Участие подтверждено"}</dd>
        </div>
        <div>
          <dt>Действует до</dt>
          <dd>{formatDate(invite.expires_at)}</dd>
        </div>
      </dl>
    </SectionShell>
  );
}

function formatDate(value: string) {
  return formatDateLabel(value, { year: "numeric" });
}
