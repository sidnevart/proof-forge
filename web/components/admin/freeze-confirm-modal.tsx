"use client";

import { useEffect, useState } from "react";
import styles from "./freeze-confirm-modal.module.css";

interface WorkspaceRef {
  id: number;
  name: string;
}

interface Props {
  workspace: WorkspaceRef;
  onConfirm: (reason: string) => void;
  onClose: () => void;
}

export function FreezeConfirmModal({ workspace, onConfirm, onClose }: Props) {
  const [reason, setReason] = useState("");

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div className={styles.overlay} onClick={onClose}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <h2 className={styles.title}>Заморозить workspace?</h2>
        <p className={styles.desc}>
          «{workspace.name}» будет заморожен. Пользователи не смогут создавать
          новые цели и пруфы. Данные сохранятся.
        </p>
        <input
          className={styles.input}
          placeholder="Причина заморозки"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          autoFocus
        />
        <div className={styles.actions}>
          <button className={styles.cancelBtn} type="button" onClick={onClose}>
            Отмена
          </button>
          <button
            className={styles.dangerBtn}
            type="button"
            disabled={!reason.trim()}
            onClick={() => onConfirm(reason)}
          >
            Заморозить
          </button>
        </div>
      </div>
    </div>
  );
}
