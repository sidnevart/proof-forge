"use client";

import { useEffect, useState, useCallback } from "react";
import { HealthScoreBadge } from "@/components/admin/health-score-badge";
import { FreezeConfirmModal } from "@/components/admin/freeze-confirm-modal";
import styles from "./workspaces.module.css";

interface WorkspaceRow {
  id: number;
  name: string;
  slug: string;
  type: "organization" | "community";
  health_score: number;
  member_count: number;
  churn_risk: "low" | "medium" | "high";
  is_frozen: boolean;
  created_at: string;
}

interface AdminWorkspaceDTO {
  id: number;
  name: string;
  slug: string;
  type: "organization" | "community";
  is_active: boolean;
  created_at: string;
  members_count: number;
}

type ChurnRisk = WorkspaceRow["churn_risk"];

const RISK_LABEL: Record<ChurnRisk, string> = { low: "Низкий", medium: "Средний", high: "Высокий" };
const RISK_COLOR: Record<ChurnRisk, string> = {
  low: "var(--win)",
  medium: "var(--warn, #f6c90e)",
  high: "var(--danger, #e53e3e)",
};
const TYPE_LABEL: Record<string, string> = { organization: "Организация", community: "Сообщество" };

function apiFetch<T>(path: string, opts?: RequestInit): Promise<T> {
  return fetch(path, { credentials: "include", ...opts }).then((r) => {
    if (!r.ok) throw new Error(r.statusText);
    return r.json() as Promise<T>;
  });
}

export default function AdminWorkspacesPage() {
  const [rows, setRows] = useState<WorkspaceRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [freezeTarget, setFreezeTarget] = useState<WorkspaceRow | null>(null);
  const [actionLoading, setActionLoading] = useState<number | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    apiFetch<{ data: AdminWorkspaceDTO[] }>("/v1/admin/workspaces")
      .then((d) => setRows((d.data ?? []).map((ws) => ({
        id: ws.id,
        name: ws.name,
        slug: ws.slug,
        type: ws.type,
        health_score: 0,
        member_count: ws.members_count,
        churn_risk: ws.members_count > 0 ? "medium" : "high",
        is_frozen: !ws.is_active,
        created_at: ws.created_at,
      }))))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  async function handleFreeze(reason: string) {
    if (!freezeTarget) return;
    setActionLoading(freezeTarget.id);
    try {
      await apiFetch(`/v1/admin/workspaces/${freezeTarget.id}/freeze`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reason }),
      });
      setRows((prev) => prev.map((r) => r.id === freezeTarget.id ? { ...r, is_frozen: true } : r));
    } catch { /* silent */ } finally {
      setActionLoading(null);
      setFreezeTarget(null);
    }
  }

  async function handleUnfreeze(ws: WorkspaceRow) {
    setActionLoading(ws.id);
    try {
      await apiFetch(`/v1/admin/workspaces/${ws.id}/unfreeze`, { method: "POST" });
      setRows((prev) => prev.map((r) => r.id === ws.id ? { ...r, is_frozen: false } : r));
    } catch { /* silent */ } finally {
      setActionLoading(null);
    }
  }

  return (
    <>
      <div className={styles.header}>
        <h1 className={styles.title}>Пространства</h1>
        <span className={styles.count}>{rows.length}</span>
      </div>

      {loading ? (
        <p className={styles.empty}>Загрузка…</p>
      ) : rows.length === 0 ? (
        <p className={styles.empty}>Нет пространств</p>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Название</th>
                <th>Тип</th>
                <th>Health</th>
                <th>Участники</th>
                <th>Риск</th>
                <th>Статус</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {rows.map((ws) => (
                <tr key={ws.id} className={ws.is_frozen ? styles.frozen : ""}>
                  <td className={styles.nameCell}>
                    <span className={styles.wsName}>{ws.name}</span>
                    <span className={styles.wsSlug}>{ws.slug}</span>
                  </td>
                  <td>{TYPE_LABEL[ws.type] ?? ws.type}</td>
                  <td><HealthScoreBadge score={ws.health_score} /></td>
                  <td className={styles.num}>{ws.member_count}</td>
                  <td>
                    <span className={styles.riskBadge} style={{ color: RISK_COLOR[ws.churn_risk] }}>
                      {RISK_LABEL[ws.churn_risk]}
                    </span>
                  </td>
                  <td>{ws.is_frozen ? <span className={styles.frozenTag}>Заморожен</span> : "—"}</td>
                  <td className={styles.actions}>
                    {ws.is_frozen ? (
                      <button
                        className={styles.unfreezeBtn}
                        disabled={actionLoading === ws.id}
                        onClick={() => handleUnfreeze(ws)}
                      >
                        Разморозить
                      </button>
                    ) : (
                      <button
                        className={styles.freezeBtn}
                        disabled={actionLoading === ws.id}
                        onClick={() => setFreezeTarget(ws)}
                      >
                        Заморозить
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {freezeTarget && (
        <FreezeConfirmModal
          workspace={{ id: freezeTarget.id, name: freezeTarget.name }}
          onConfirm={handleFreeze}
          onClose={() => setFreezeTarget(null)}
        />
      )}
    </>
  );
}
