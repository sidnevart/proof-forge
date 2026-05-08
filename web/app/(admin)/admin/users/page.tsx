"use client";

import { useEffect, useState, useRef, useCallback } from "react";
import styles from "./users.module.css";

interface UserRow {
  id: number;
  name: string;
  email: string;
  role: string;
  workspace_count: number;
  last_active_at: string | null;
  is_blocked: boolean;
}

interface AdminUserDTO {
  id: number;
  display_name: string;
  email: string;
  is_platform_admin: boolean;
  created_at: string;
  goals_count: number;
}

const ROLE_LABELS: Record<string, string> = {
  platform_admin: "Платформ. администратор",
  workspace_owner: "Владелец",
  community_leader: "Лидер сообщества",
  lead: "Тимлид",
  trusted_approver: "Одобряющий",
  member: "Участник",
};

function apiFetch<T>(path: string, opts?: RequestInit): Promise<T> {
  return fetch(path, { credentials: "include", ...opts }).then((r) => {
    if (!r.ok) throw new Error(r.statusText);
    return r.json() as Promise<T>;
  });
}

function formatDate(iso: string | null): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("ru-RU", { day: "numeric", month: "short" });
}

export default function AdminUsersPage() {
  const [query, setQuery] = useState("");
  const [rows, setRows] = useState<UserRow[]>([]);
  const [loading, setLoading] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const search = useCallback((q: string) => {
    setLoading(true);
    apiFetch<{ data: AdminUserDTO[] }>(`/v1/admin/users?q=${encodeURIComponent(q)}`)
      .then((d) => setRows((d.data ?? []).map((u) => ({
        id: u.id,
        name: u.display_name,
        email: u.email,
        role: u.is_platform_admin ? "platform_admin" : "member",
        workspace_count: u.goals_count,
        last_active_at: u.created_at,
        is_blocked: false,
      }))))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => search(query), 350);
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  }, [query, search]);

  return (
    <>
      <div className={styles.header}>
        <h1 className={styles.title}>Пользователи</h1>
      </div>

      <div className={styles.searchWrap}>
        <input
          className={styles.search}
          type="search"
          placeholder="Поиск по имени или email…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          autoFocus
        />
        {loading && <span className={styles.spinner}>…</span>}
      </div>

      {!loading && rows.length === 0 && (
        <p className={styles.empty}>{query ? "Ничего не найдено" : "Начните вводить имя или email"}</p>
      )}

      {rows.length > 0 && (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Пользователь</th>
                <th>Роль</th>
                <th>Пространств</th>
                <th>Последняя активность</th>
                <th>Статус</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((u) => (
                <tr key={u.id} className={u.is_blocked ? styles.blocked : ""}>
                  <td className={styles.nameCell}>
                    <span className={styles.userName}>{u.name}</span>
                    <span className={styles.userEmail}>{u.email}</span>
                  </td>
                  <td>
                    <span className={styles.roleBadge}>
                      {ROLE_LABELS[u.role] ?? u.role}
                    </span>
                  </td>
                  <td className={styles.num}>{u.workspace_count}</td>
                  <td className={styles.date}>{formatDate(u.last_active_at)}</td>
                  <td>{u.is_blocked ? <span className={styles.blockedTag}>Заблокирован</span> : "Активен"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}
