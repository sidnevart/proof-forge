"use client";

import { type ChangeEvent, type FormEvent, useCallback, useEffect, useRef, useState, useTransition } from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/core/button";
import { SectionShell } from "@/components/core/section-shell";
import { StatePanel } from "@/components/core/state-panel";
import { StatusPill } from "@/components/core/status-pill";
import {
  ApiError,
  addFileEvidence,
  addLinkEvidence,
  addTextEvidence,
  createCheckIn,
  listCheckIns,
  submitCheckIn,
} from "@/lib/api";
import type { CheckIn, EvidenceItem } from "@/lib/types";

import styles from "./checkin-screen.module.css";

type ScreenState =
  | { kind: "loading" }
  | { kind: "unauthenticated" }
  | { kind: "error"; message: string }
  | { kind: "no_draft" }
  | { kind: "draft"; checkIn: CheckIn; evidence: EvidenceItem[] }
  | { kind: "submitted" };

const ALLOWED_TYPES = ["image/jpeg", "image/png", "image/gif", "image/webp", "text/plain", "application/pdf"];

export function CheckInScreen({ goalID }: { goalID: number }) {
  const router = useRouter();
  const [state, setState] = useState<ScreenState>({ kind: "loading" });
  const [textContent, setTextContent] = useState("");
  const [linkURL, setLinkURL] = useState("");
  const [evidenceError, setEvidenceError] = useState<string | null>(null);
  const [isAddingText, startAddText] = useTransition();
  const [isAddingLink, startAddLink] = useTransition();
  const [isAddingFile, startAddFile] = useTransition();
  const [isSubmitting, startSubmit] = useTransition();
  const [isStarting, startCreate] = useTransition();
  const fileRef = useRef<HTMLInputElement>(null);

  const loadDraft = useCallback(async () => {
    try {
      const data = await listCheckIns(goalID);
      const drafts = (data.check_ins ?? []).filter(
        (ci) => ci.status === "draft" || ci.status === "changes_requested",
      );
      if (drafts.length > 0) {
        setState({ kind: "draft", checkIn: drafts[0], evidence: [] });
      } else {
        setState({ kind: "no_draft" });
      }
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setState({ kind: "unauthenticated" });
        return;
      }
      setState({
        kind: "error",
        message: error instanceof Error ? error.message : "Не удалось загрузить check-in.",
      });
    }
  }, [goalID]);

  useEffect(() => {
    void loadDraft();
  }, [loadDraft]);

  function handleStartCheckIn() {
    startCreate(async () => {
      try {
        const data = await createCheckIn(goalID);
        setState({ kind: "draft", checkIn: data.check_in, evidence: [] });
      } catch (error) {
        setState({
          kind: "error",
          message: error instanceof Error ? error.message : "Не удалось создать check-in.",
        });
      }
    });
  }

  function handleAddText(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setEvidenceError(null);
    const content = textContent.trim();
    if (!content) return;
    if (state.kind !== "draft") return;

    const checkInID = state.checkIn.id;
    startAddText(async () => {
      try {
        const data = await addTextEvidence(checkInID, content);
        setTextContent("");
        setState((prev) =>
          prev.kind === "draft"
            ? { ...prev, evidence: [...prev.evidence, data.evidence] }
            : prev,
        );
      } catch (error) {
        setEvidenceError(error instanceof Error ? error.message : "Не удалось добавить текст.");
      }
    });
  }

  function handleAddLink(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setEvidenceError(null);
    const url = linkURL.trim();
    if (!url) return;
    if (state.kind !== "draft") return;

    const checkInID = state.checkIn.id;
    startAddLink(async () => {
      try {
        const data = await addLinkEvidence(checkInID, url);
        setLinkURL("");
        setState((prev) =>
          prev.kind === "draft"
            ? { ...prev, evidence: [...prev.evidence, data.evidence] }
            : prev,
        );
      } catch (error) {
        setEvidenceError(error instanceof Error ? error.message : "Не удалось добавить ссылку.");
      }
    });
  }

  function handleFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file || state.kind !== "draft") return;
    setEvidenceError(null);

    if (!ALLOWED_TYPES.includes(file.type)) {
      setEvidenceError("Поддерживаются только изображения, .txt и .pdf файлы.");
      return;
    }
    if (file.size > 10 * 1024 * 1024) {
      setEvidenceError("Файл превышает лимит 10 МБ.");
      return;
    }

    const checkInID = state.checkIn.id;
    startAddFile(async () => {
      try {
        const data = await addFileEvidence(checkInID, file);
        if (fileRef.current) fileRef.current.value = "";
        setState((prev) =>
          prev.kind === "draft"
            ? { ...prev, evidence: [...prev.evidence, data.evidence] }
            : prev,
        );
      } catch (error) {
        setEvidenceError(error instanceof Error ? error.message : "Не удалось загрузить файл.");
      }
    });
  }

  function handleSubmit() {
    if (state.kind !== "draft") return;
    const checkInID = state.checkIn.id;
    startSubmit(async () => {
      try {
        await submitCheckIn(checkInID);
        setState({ kind: "submitted" });
        setTimeout(() => router.push("/dashboard"), 2000);
      } catch (error) {
        setEvidenceError(error instanceof Error ? error.message : "Не удалось отправить check-in.");
      }
    });
  }

  if (state.kind === "loading") {
    return (
      <main className={styles.page}>
        <CheckInHeader status="pending" label="Loading" />
        <StatePanel tone="loading" title="Загружаем черновик" description="Проверяем, есть ли активный check-in для этого goal." />
      </main>
    );
  }

  if (state.kind === "unauthenticated") {
    return (
      <main className={styles.page}>
        <CheckInHeader status="rejected" label="Auth required" />
        <StatePanel
          tone="error"
          title="Сессия не найдена"
          description="Войдите в систему, чтобы создать check-in."
          meta={<Button variant="secondary" onClick={() => router.push("/dashboard")}>Перейти ко входу</Button>}
        />
      </main>
    );
  }

  if (state.kind === "error") {
    return (
      <main className={styles.page}>
        <CheckInHeader status="rejected" label="Error" />
        <StatePanel
          tone="error"
          title="Ошибка"
          description={state.message}
          meta={<Button variant="secondary" onClick={() => void loadDraft()}>Повторить</Button>}
        />
      </main>
    );
  }

  if (state.kind === "submitted") {
    return (
      <main className={styles.page}>
        <CheckInHeader status="active" label="Submitted" />
        <StatePanel
          tone="success"
          title="Check-in отправлен на buddy review"
          description="Buddy получит proof и вынесет решение. Возвращаемся на dashboard..."
        />
      </main>
    );
  }

  if (state.kind === "no_draft") {
    return (
      <main className={styles.page}>
        <CheckInHeader status="pending" label="Ready" />
        <SectionShell eyebrow="New check-in" title="Зафиксировать движение по goal">
          <div className={styles.startBlock}>
            <p>
              Check-in — это конкретный proof-артефакт, который buddy должен рассмотреть и
              подтвердить. Он не обновляет progress автоматически — только через явное approval.
            </p>
            <Button onClick={handleStartCheckIn} disabled={isStarting}>
              {isStarting ? "Создаём черновик..." : "Начать check-in"}
            </Button>
          </div>
        </SectionShell>
      </main>
    );
  }

  const { checkIn, evidence } = state;
  const statusLabel = checkIn.status === "changes_requested" ? "Changes requested" : "Draft";

  return (
    <main className={styles.page}>
      <CheckInHeader status="pending" label={statusLabel} />

      {checkIn.status === "changes_requested" && (
        <StatePanel
          tone="pending"
          title="Buddy запросил доработку"
          description="Добавьте или уточните proof-артефакты, затем отправьте check-in повторно."
        />
      )}

      <div className={styles.grid}>
        <div className={styles.forms}>
          <SectionShell eyebrow="Text evidence" title="Текстовое доказательство">
            <form className={styles.form} onSubmit={handleAddText}>
              <label className={styles.field}>
                <span>Что именно было сделано</span>
                <textarea
                  value={textContent}
                  onChange={(e) => setTextContent(e.target.value)}
                  placeholder="Опишите конкретный результат или сделанный шаг"
                  rows={5}
                  disabled={isAddingText}
                />
              </label>
              <Button type="submit" variant="secondary" disabled={isAddingText || !textContent.trim()}>
                {isAddingText ? "Добавляем..." : "Добавить текст"}
              </Button>
            </form>
          </SectionShell>

          <SectionShell eyebrow="Link evidence" title="Внешняя ссылка">
            <form className={styles.form} onSubmit={handleAddLink}>
              <label className={styles.field}>
                <span>URL</span>
                <input
                  type="url"
                  value={linkURL}
                  onChange={(e) => setLinkURL(e.target.value)}
                  placeholder="https://github.com/..."
                  disabled={isAddingLink}
                />
              </label>
              <Button type="submit" variant="secondary" disabled={isAddingLink || !linkURL.trim()}>
                {isAddingLink ? "Добавляем..." : "Добавить ссылку"}
              </Button>
            </form>
          </SectionShell>

          <SectionShell eyebrow="File evidence" title="Файл или скриншот">
            <div className={styles.form}>
              <label className={styles.field}>
                <span>Файл (до 10 МБ · PNG, JPG, GIF, WEBP, PDF, TXT)</span>
                <input
                  ref={fileRef}
                  type="file"
                  accept={ALLOWED_TYPES.join(",")}
                  onChange={handleFileChange}
                  disabled={isAddingFile}
                  className={styles.fileInput}
                />
              </label>
              {isAddingFile && <p className={styles.uploadHint}>Загружаем файл...</p>}
            </div>
          </SectionShell>

          {evidenceError && (
            <p className={styles.formError} role="alert">
              {evidenceError}
            </p>
          )}
        </div>

        <div className={styles.sidebar}>
          <SectionShell eyebrow="Collected evidence" title={`Proof (${evidence.length})`}>
            {evidence.length === 0 ? (
              <StatePanel
                tone="empty"
                title="Пока нет proof-артефактов"
                description="Добавьте хотя бы один текст, ссылку или файл перед отправкой."
              />
            ) : (
              <ol className={styles.evidenceList}>
                {evidence.map((item) => (
                  <EvidenceCard key={item.id} item={item} />
                ))}
              </ol>
            )}
          </SectionShell>

          <SectionShell eyebrow="Submit" title="Отправить на buddy review">
            <div className={styles.submitBlock}>
              <p>
                После отправки buddy получит уведомление и вынесет решение. Вы не сможете добавить
                новый артефакт пока check-in на review.
              </p>
              <Button onClick={handleSubmit} disabled={isSubmitting || evidence.length === 0}>
                {isSubmitting ? "Отправляем..." : "Отправить check-in"}
              </Button>
            </div>
          </SectionShell>
        </div>
      </div>
    </main>
  );
}

function CheckInHeader({ status, label }: { status: string; label: string }) {
  return (
    <header className={styles.header}>
      <div>
        <span className="eyebrow">Proof check-in</span>
        <h1>Зафиксируйте подтверждаемый прогресс</h1>
      </div>
      <StatusPill status={status as Parameters<typeof StatusPill>[0]["status"]} label={label} />
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
