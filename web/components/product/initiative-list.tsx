"use client";

import { useCallback, useEffect, useState, useTransition } from "react";

import { ApiError, createInitiative, joinInitiative, listInitiatives } from "@/lib/api";
import type { CreateInitiativeInput, Initiative, InitiativeSpaceType } from "@/lib/types";

import styles from "./initiative-list.module.css";

type Props = {
  spaceType: InitiativeSpaceType;
  spaceId: number;
  canCreate?: boolean;
};

type State =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "ready"; items: Initiative[] };

export function InitiativeList({ spaceType, spaceId, canCreate = false }: Props) {
  const [state, setState] = useState<State>({ kind: "loading" });
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState<CreateInitiativeInput>({ title: "", proof_criteria: "" });
  const [formError, setFormError] = useState<string | null>(null);
  const [joiningId, setJoiningId] = useState<number | null>(null);
  const [isPending, startTransition] = useTransition();

  const load = useCallback(async () => {
    try {
      const items = await listInitiatives(spaceType, spaceId);
      setState({ kind: "ready", items });
    } catch {
      setState({ kind: "error", message: "Не удалось загрузить инициативы" });
    }
  }, [spaceType, spaceId]);

  useEffect(() => { load(); }, [load]);

  const handleCreate = () => {
    setFormError(null);
    startTransition(async () => {
      try {
        await createInitiative(spaceType, spaceId, form);
        setCreating(false);
        setForm({ title: "", proof_criteria: "" });
        await load();
      } catch (err) {
        setFormError(err instanceof ApiError ? err.message : "Ошибка создания инициативы");
      }
    });
  };

  const handleJoin = (id: number) => {
    setJoiningId(id);
    startTransition(async () => {
      try {
        await joinInitiative(id);
        await load();
      } catch {
        // ignore — idempotent
      } finally {
        setJoiningId(null);
      }
    });
  };

  if (state.kind === "loading") {
    return <div className={styles.loading}>Загрузка...</div>;
  }
  if (state.kind === "error") {
    return <div className={styles.error}>{state.message}</div>;
  }

  const { items } = state;

  return (
    <div className={styles.root}>
      <div className={styles.header}>
        <h2 className={styles.title}>Инициативы</h2>
        {canCreate && !creating && (
          <button className={styles.createBtn} onClick={() => setCreating(true)}>
            + Создать
          </button>
        )}
      </div>

      {creating && (
        <div className={styles.form}>
          <input
            className={styles.input}
            placeholder="Название инициативы"
            value={form.title}
            onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
          />
          <textarea
            className={styles.textarea}
            placeholder="Критерий пруфа — что считается доказательством"
            value={form.proof_criteria}
            onChange={(e) => setForm((f) => ({ ...f, proof_criteria: e.target.value }))}
            rows={3}
          />
          {formError && <p className={styles.formError}>{formError}</p>}
          <div className={styles.formActions}>
            <button className={styles.submitBtn} onClick={handleCreate} disabled={isPending}>
              {isPending ? "Создаём..." : "Создать"}
            </button>
            <button className={styles.cancelBtn} onClick={() => setCreating(false)}>
              Отмена
            </button>
          </div>
        </div>
      )}

      {items.length === 0 && !creating && (
        <p className={styles.empty}>Инициатив пока нет</p>
      )}

      <ul className={styles.list}>
        {items.map((ini) => (
          <li key={ini.id} className={styles.card}>
            <div className={styles.cardMain}>
              <h3 className={styles.cardTitle}>{ini.title}</h3>
              {ini.description && <p className={styles.cardDesc}>{ini.description}</p>}
              <p className={styles.cardCriteria}>{ini.proof_criteria}</p>
              <div className={styles.cardMeta}>
                <span>{ini.participant_count} участников</span>
                <span>{ini.active_today} активных сегодня</span>
                {ini.pending_proof_count > 0 && (
                  <span className={styles.pending}>{ini.pending_proof_count} ждут ревью</span>
                )}
              </div>
            </div>
            <button
              className={styles.joinBtn}
              onClick={() => handleJoin(ini.id)}
              disabled={joiningId === ini.id || isPending}
            >
              {joiningId === ini.id ? "..." : "Вступить"}
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
