"use client";

import { type FormEvent, useCallback, useEffect, useState, useTransition } from "react";

import { Button } from "@/components/core/button";
import { SectionShell } from "@/components/core/section-shell";
import { StatePanel } from "@/components/core/state-panel";
import { StatusPill } from "@/components/core/status-pill";
import { ApiError, cancelStake, createStake, forfeitStake, listStakes } from "@/lib/api";
import { cn } from "@/lib/cn";
import type { StakeView } from "@/lib/types";

import styles from "./stake-panel.module.css";

type ScreenState =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "ready"; stakes: StakeView[] };

type Props = {
  goalID: number;
  isOwner: boolean;
  isBuddy: boolean;
};

export function StakePanel({ goalID, isOwner, isBuddy }: Props) {
  const [state, setState] = useState<ScreenState>({ kind: "loading" });
  const [description, setDescription] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const [isCreating, startCreate] = useTransition();

  const load = useCallback(async () => {
    try {
      const data = await listStakes(goalID);
      setState({ kind: "ready", stakes: data.stakes ?? [] });
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setState({ kind: "error", message: "Сессия не найдена." });
        return;
      }
      setState({
        kind: "error",
        message: error instanceof Error ? error.message : "Не удалось загрузить ставки.",
      });
    }
  }, [goalID]);

  useEffect(() => {
    void load();
  }, [load]);

  function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(null);
    const text = description.trim();
    if (!text) return;

    startCreate(async () => {
      try {
        await createStake(goalID, text);
        setDescription("");
        await load();
      } catch (error) {
        setFormError(error instanceof Error ? error.message : "Не удалось создать ставку.");
      }
    });
  }

  if (state.kind === "loading") {
    return (
      <StatePanel
        tone="loading"
        title="Загружаем ставки"
        description="Получаем данные по ставкам цели."
      />
    );
  }

  if (state.kind === "error") {
    return <StatePanel tone="error" title="Ошибка" description={state.message} />;
  }

  const { stakes } = state;
  const activeCount = stakes.filter((s) => s.stake.status === "active").length;

  return (
    <SectionShell
      eyebrow="На кону"
      title={stakes.length > 0 ? `Ставки (${activeCount} активных)` : "Ставки"}
    >
      {stakes.length > 0 ? (
        <ol className={styles.list}>
          {stakes.map((sv) => (
            <StakeCard key={sv.stake.id} view={sv} isBuddy={isBuddy} isOwner={isOwner} onUpdate={load} />
          ))}
        </ol>
      ) : (
        <p style={{ margin: 0, fontSize: 14, color: "var(--text-dim)" }}>
          {isOwner
            ? "Пока ставок нет. Добавьте ставку, чтобы повысить серьёзность цели."
            : "Владелец пока не добавил ставок."}
        </p>
      )}

      {isOwner ? (
        <form className={styles.addForm} onSubmit={handleCreate}>
          <textarea
            className={styles.textarea}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Что поставлено на кон? Например: «5000₽ на благотворительность» или «побрить голову»"
            rows={2}
          />
          {formError ? <p className={styles.error} role="alert">{formError}</p> : null}
          <Button type="submit" disabled={isCreating || !description.trim()}>
            {isCreating ? "Добавляем..." : "Добавить ставку"}
          </Button>
        </form>
      ) : null}
    </SectionShell>
  );
}

function StakeCard({
  view,
  isBuddy,
  isOwner,
  onUpdate,
}: {
  view: StakeView;
  isBuddy: boolean;
  isOwner: boolean;
  onUpdate: () => Promise<void>;
}) {
  const { stake, forfeiture } = view;
  const [confirming, setConfirming] = useState(false);
  const [reason, setReason] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isForfeiting, startForfeit] = useTransition();
  const [isCancelling, startCancel] = useTransition();

  const statusTone =
    stake.status === "active"
      ? "active"
      : stake.status === "forfeited"
        ? "rejected"
        : stake.status === "completed"
          ? "approved"
          : "pending";

  const statusLabel =
    stake.status === "active"
      ? "Активна"
      : stake.status === "forfeited"
        ? "Сгорела"
        : stake.status === "completed"
          ? "Выжила"
          : "Отменена";

  function handleForfeit() {
    setError(null);
    startForfeit(async () => {
      try {
        await forfeitStake(stake.id, reason.trim());
        setConfirming(false);
        setReason("");
        await onUpdate();
      } catch (err) {
        setError(err instanceof Error ? err.message : "Не удалось списать ставку.");
      }
    });
  }

  function handleCancel() {
    setError(null);
    startCancel(async () => {
      try {
        await cancelStake(stake.id);
        await onUpdate();
      } catch (err) {
        setError(err instanceof Error ? err.message : "Не удалось отменить ставку.");
      }
    });
  }

  return (
    <li className={cn(styles.card, stake.status === "forfeited" && styles.forfeited, stake.status === "cancelled" && styles.cancelled)}>
      <div className={styles.cardHeader}>
        <p className={styles.description}>{stake.description}</p>
        <StatusPill status={statusTone} label={statusLabel} />
      </div>

      {stake.status === "forfeited" && forfeiture?.reason ? (
        <p className={styles.forfeitReason}>Причина: {forfeiture.reason}</p>
      ) : null}

      {stake.status === "forfeited" && stake.forfeited_at ? (
        <p className={styles.forfeitDate}>
          Сгорела {formatDate(stake.forfeited_at)}
        </p>
      ) : null}

      {stake.status === "active" ? (
        <div className={styles.actions}>
          {isBuddy && !confirming ? (
            <Button variant="secondary" onClick={() => setConfirming(true)}>
              Ставка сгорела
            </Button>
          ) : null}

          {isBuddy && confirming ? (
            <div className={styles.confirmRow}>
              <input
                className={styles.reasonInput}
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Причина (необязательно)"
              />
              <Button variant="primary" onClick={handleForfeit} disabled={isForfeiting}>
                {isForfeiting ? "Списываем..." : "Подтвердить"}
              </Button>
              <Button variant="ghost" onClick={() => { setConfirming(false); setReason(""); }}>
                Отмена
              </Button>
            </div>
          ) : null}

          {isOwner ? (
            <Button variant="ghost" onClick={handleCancel} disabled={isCancelling}>
              {isCancelling ? "Отменяем..." : "Убрать ставку"}
            </Button>
          ) : null}
        </div>
      ) : null}

      {error ? <p className={styles.error} role="alert">{error}</p> : null}
    </li>
  );
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("ru-RU", { day: "numeric", month: "short", year: "numeric" });
}
