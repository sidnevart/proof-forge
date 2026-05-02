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
        message: error instanceof Error ? error.message : "Не удалось загрузить подтверждение.",
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
          message: error instanceof Error ? error.message : "Не удалось открыть черновик подтверждения.",
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
        setEvidenceError(error instanceof Error ? error.message : "Не удалось отправить подтверждение.");
      }
    });
  }

  if (state.kind === "loading") {
    return (
      <main className={styles.page}>
        <CheckInHeader status="pending" label="Загрузка" />
        <StatePanel
          tone="loading"
          title="Загружаем черновик"
          description="Проверяем, есть ли активное подтверждение по этой цели."
        />
      </main>
    );
  }

  if (state.kind === "unauthenticated") {
    return (
      <main className={styles.page}>
        <CheckInHeader status="rejected" label="Нужен вход" />
        <StatePanel
          tone="error"
          title="Сессия не найдена"
          description="Войдите в систему, чтобы подготовить подтверждение."
          meta={<Button variant="secondary" onClick={() => router.push("/dashboard")}>Перейти ко входу</Button>}
        />
      </main>
    );
  }

  if (state.kind === "error") {
    return (
      <main className={styles.page}>
        <CheckInHeader status="rejected" label="Ошибка" />
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
        <CheckInHeader status="active" label="Отправлено" />
        <StatePanel
          tone="success"
          title="Подтверждение отправлено на проверку"
          description="Партнёр получит материалы и вынесет решение. Возвращаемся в центр управления..."
        />
      </main>
    );
  }

  if (state.kind === "no_draft") {
    return (
      <main className={styles.page}>
        <CheckInHeader status="pending" label="Можно начинать" />
        <SectionShell eyebrow="Новое подтверждение" title="Подготовьте материалы по цели">
          <div className={styles.startBlock}>
            <p>
              Подтверждение должно показать, что по цели действительно произошло движение.
              После отправки партнёр проверит материалы и примет решение.
            </p>
            <Button onClick={handleStartCheckIn} disabled={isStarting}>
              {isStarting ? "Открываем черновик..." : "Подготовить подтверждение"}
            </Button>
          </div>
        </SectionShell>
      </main>
    );
  }

  const { checkIn, evidence } = state;
  const statusLabel = checkIn.status === "changes_requested" ? "Нужна доработка" : "Черновик";

  return (
    <main className={styles.page}>
      <CheckInHeader status="pending" label={statusLabel} />

      {checkIn.status === "changes_requested" && (
        <StatePanel
          tone="pending"
          title="Нужно дополнить подтверждение"
          description="Добавьте или уточните материалы, затем отправьте их на проверку повторно."
        />
      )}

      <div className={styles.grid}>
        <div className={styles.forms}>
          <SectionShell eyebrow="Текст" title="Опишите сделанное">
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

          <SectionShell eyebrow="Ссылка" title="Добавьте внешний материал">
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

          <SectionShell eyebrow="Файл" title="Загрузите изображение или документ">
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
          <SectionShell eyebrow="Собранные материалы" title={`Подтверждение (${evidence.length})`}>
            {evidence.length === 0 ? (
              <StatePanel
                tone="empty"
                title="Пока нет материалов"
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

          <SectionShell eyebrow="Отправка" title="Передать партнёру на проверку">
            <div className={styles.submitBlock}>
              <p>
                После отправки партнёр получит уведомление и вынесет решение. Пока
                подтверждение на проверке, новые материалы добавить нельзя.
              </p>
              <Button onClick={handleSubmit} disabled={isSubmitting || evidence.length === 0}>
                {isSubmitting ? "Отправляем..." : "Отправить на проверку"}
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
        <span className="eyebrow">Подтверждение прогресса</span>
        <h1>Соберите материалы для проверки</h1>
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
