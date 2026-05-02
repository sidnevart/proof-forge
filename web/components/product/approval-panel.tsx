"use client";

import { useCallback, useEffect, useState, useTransition } from "react";

import { Button } from "@/components/core/button";
import { SectionShell } from "@/components/core/section-shell";
import { StatePanel } from "@/components/core/state-panel";
import { StatusPill } from "@/components/core/status-pill";
import { ApiError, approveCheckIn, getCheckIn, rejectCheckIn, requestChanges } from "@/lib/api";
import type { CheckIn, EvidenceItem } from "@/lib/types";

import styles from "./approval-panel.module.css";

type ScreenState =
  | { kind: "loading" }
  | { kind: "unauthenticated" }
  | { kind: "error"; message: string }
  | { kind: "not_submitted" }
  | { kind: "reviewing"; checkIn: CheckIn; evidence: EvidenceItem[] }
  | { kind: "decided"; decision: "approved" | "rejected" | "changes_requested" };

export function ApprovalPanel({ checkInID }: { checkInID: number }) {
  const [state, setState] = useState<ScreenState>({ kind: "loading" });
  const [comment, setComment] = useState("");
  const [actionError, setActionError] = useState<string | null>(null);
  const [isApproving, startApprove] = useTransition();
  const [isRejecting, startReject] = useTransition();
  const [isRequesting, startRequest] = useTransition();

  const load = useCallback(async () => {
    try {
      const data = await getCheckIn(checkInID);
      if (data.check_in.status !== "submitted") {
        setState({ kind: "not_submitted" });
        return;
      }
      setState({ kind: "reviewing", checkIn: data.check_in, evidence: data.evidence ?? [] });
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setState({ kind: "unauthenticated" });
        return;
      }
      setState({
        kind: "error",
        message: error instanceof Error ? error.message : "Не удалось загрузить подтверждение.",
      });
    }
  }, [checkInID]);

  useEffect(() => {
    void load();
  }, [load]);

  function handleApprove() {
    setActionError(null);
    startApprove(async () => {
      try {
        await approveCheckIn(checkInID, comment.trim() || undefined);
        setState({ kind: "decided", decision: "approved" });
      } catch (error) {
        setActionError(error instanceof Error ? error.message : "Не удалось подтвердить результат.");
      }
    });
  }

  function handleReject() {
    setActionError(null);
    startReject(async () => {
      try {
        await rejectCheckIn(checkInID, comment.trim() || undefined);
        setState({ kind: "decided", decision: "rejected" });
      } catch (error) {
        setActionError(error instanceof Error ? error.message : "Не удалось отклонить результат.");
      }
    });
  }

  function handleRequestChanges() {
    setActionError(null);
    startRequest(async () => {
      try {
        await requestChanges(checkInID, comment.trim() || undefined);
        setState({ kind: "decided", decision: "changes_requested" });
      } catch (error) {
        setActionError(error instanceof Error ? error.message : "Не удалось запросить доработку.");
      }
    });
  }

  const busy = isApproving || isRejecting || isRequesting;

  if (state.kind === "loading") {
    return (
      <main className={styles.page}>
        <ApprovalHeader />
        <StatePanel tone="loading" title="Загружаем подтверждение" description="Получаем материалы для проверки." />
      </main>
    );
  }

  if (state.kind === "unauthenticated") {
    return (
      <main className={styles.page}>
        <ApprovalHeader />
        <StatePanel tone="error" title="Сессия не найдена" description="Войдите в систему, чтобы просмотреть подтверждение." />
      </main>
    );
  }

  if (state.kind === "error") {
    return (
      <main className={styles.page}>
        <ApprovalHeader />
        <StatePanel
          tone="error"
          title="Ошибка"
          description={state.message}
          meta={<Button variant="secondary" onClick={() => void load()}>Повторить</Button>}
        />
      </main>
    );
  }

  if (state.kind === "not_submitted") {
    return (
      <main className={styles.page}>
        <ApprovalHeader />
        <StatePanel
          tone="empty"
          title="Подтверждение ещё не отправлено"
          description="Владелец цели ещё не отправил материалы на проверку. Как только они появятся, вы сможете вынести решение."
        />
      </main>
    );
  }

  if (state.kind === "decided") {
    const titles: Record<typeof state.decision, string> = {
      approved: "Подтверждение принято",
      rejected: "Подтверждение отклонено",
      changes_requested: "Запрошена доработка",
    };
    const descs: Record<typeof state.decision, string> = {
      approved: "Владелец цели получит подтверждение, что результат принят.",
      rejected: "Владелец цели получит отклонение и сможет начать новый цикл позже.",
      changes_requested: "Владелец цели получит запрос на уточнение материалов.",
    };
    const tones: Record<typeof state.decision, "success" | "error" | "pending"> = {
      approved: "success",
      rejected: "error",
      changes_requested: "pending",
    };
    return (
      <main className={styles.page}>
        <ApprovalHeader />
        <StatePanel tone={tones[state.decision]} title={titles[state.decision]} description={descs[state.decision]} />
      </main>
    );
  }

  const { checkIn, evidence } = state;

  return (
    <main className={styles.page}>
      <ApprovalHeader />

      <div className={styles.grid}>
        <SectionShell eyebrow="Подтверждение" title={`Артефакты (${evidence.length})`}>
          {evidence.length === 0 ? (
            <StatePanel tone="empty" title="Нет материалов" description="Владелец цели пока не добавил подтверждения." />
          ) : (
            <ol className={styles.evidenceList}>
              {evidence.map((item) => (
                <EvidenceCard key={item.id} item={item} />
              ))}
            </ol>
          )}
        </SectionShell>

        <SectionShell eyebrow="Проверка" title="Вынести решение">
          <div className={styles.actions}>
            <p>
              Проверьте материалы и вынесите решение. Подтверждение фиксирует движение
              по цели, возврат на доработку просит уточнить материалы, отклонение
              завершает этот цикл без подтверждения результата.
            </p>

            <label className={styles.commentField}>
              <span>Комментарий (необязательно)</span>
              <textarea
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                placeholder="Поясните решение для владельца"
                rows={3}
                disabled={busy}
              />
            </label>

            <div className={styles.buttonRow}>
              <Button onClick={handleApprove} disabled={busy}>
                {isApproving ? "Подтверждаем..." : "Подтвердить"}
              </Button>
              <Button variant="secondary" onClick={handleRequestChanges} disabled={busy}>
                {isRequesting ? "Отправляем..." : "Вернуть на доработку"}
              </Button>
              <Button variant="ghost" onClick={handleReject} disabled={busy}>
                {isRejecting ? "Отклоняем..." : "Отклонить"}
              </Button>
            </div>

            {actionError && (
              <p className={styles.actionError} role="alert">
                {actionError}
              </p>
            )}

            <div>
              <p style={{ margin: 0, fontSize: 13, color: "var(--text-dim)" }}>
                Отправлено: {checkIn.submitted_at ? new Date(checkIn.submitted_at).toLocaleString("ru-RU") : "—"}
              </p>
            </div>
          </div>
        </SectionShell>
      </div>
    </main>
  );
}

function ApprovalHeader() {
  return (
    <header className={styles.header}>
      <div>
        <span className="eyebrow">Проверка партнёром</span>
        <h1>Проверьте подтверждение по цели</h1>
      </div>
      <StatusPill status="pending" label="Ожидает решения" />
    </header>
  );
}

function EvidenceCard({ item }: { item: EvidenceItem }) {
  return (
    <li className={styles.evidenceItem}>
      <span className={styles.evidenceKind}>{item.kind}</span>
      {item.kind === "text" && <p>{item.text_content}</p>}
      {item.kind === "link" && (
        <a href={item.external_url} target="_blank" rel="noopener noreferrer">
          {item.external_url}
        </a>
      )}
      {(item.kind === "file" || item.kind === "image") && (
        <p>
          {item.mime_type} · {formatBytes(item.file_size_bytes ?? 0)}
        </p>
      )}
    </li>
  );
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}
