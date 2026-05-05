"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import {
  acceptCircleInvitation,
  declineCircleInvitation,
  listMyInvitations,
} from "@/lib/api";
import type { CircleInvitation } from "@/lib/types";

import styles from "./invitation-inbox.module.css";

export function InvitationInbox() {
  const [invitations, setInvitations] = useState<CircleInvitation[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);

  const load = useCallback(async () => {
    try {
      const data = await listMyInvitations();
      setInvitations(data.invitations ?? []);
    } catch {
      // silently ignore – user may not be logged in
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  // Close panel on outside click
  useEffect(() => {
    if (!open) return;
    function handleClick(e: MouseEvent) {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [open]);

  async function handleAccept(id: number) {
    if (loading) return;
    setLoading(true);
    try {
      await acceptCircleInvitation(id);
      await load();
    } finally {
      setLoading(false);
    }
  }

  async function handleDecline(id: number) {
    if (loading) return;
    setLoading(true);
    try {
      await declineCircleInvitation(id);
      await load();
    } finally {
      setLoading(false);
    }
  }

  if (invitations.length === 0) return null;

  return (
    <div className={styles.trigger} ref={panelRef}>
      <button
        className={styles.iconBtn}
        onClick={() => setOpen((v) => !v)}
        aria-label="Приглашения в круги"
      >
        КРУГИ
        <span className={styles.badge}>{invitations.length}</span>
      </button>

      {open && (
        <div className={styles.panel}>
          {invitations.map((inv) => (
            <div key={inv.id} className={styles.item}>
              <div>
                <strong>{inv.circle_name}</strong>
              </div>
              {inv.message && (
                <div style={{ fontSize: 12, color: "#aaa", marginTop: 2 }}>
                  {inv.message}
                </div>
              )}
              <div className={styles.actions}>
                <button
                  className={styles.acceptBtn}
                  onClick={() => handleAccept(inv.id)}
                  disabled={loading}
                >
                  ПРИНЯТЬ
                </button>
                <button
                  className={styles.declineBtn}
                  onClick={() => handleDecline(inv.id)}
                  disabled={loading}
                >
                  ОТКЛОНИТЬ
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
