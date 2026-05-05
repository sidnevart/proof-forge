"use client";

import { useCallback, useEffect, useState } from "react";

import {
  acceptCircleInvitation,
  declineCircleInvitation,
  listMyInvitations,
} from "@/lib/api";
import type { CircleInvitation } from "@/lib/types";

export default function InvitationsPage() {
  const [invitations, setInvitations] = useState<CircleInvitation[]>([]);
  const [loading, setLoading] = useState(false);
  const [fetched, setFetched] = useState(false);

  const load = useCallback(async () => {
    try {
      const data = await listMyInvitations();
      setInvitations(data.invitations ?? []);
    } catch {
      setInvitations([]);
    } finally {
      setFetched(true);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

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

  return (
    <main style={{ padding: "24px 16px", maxWidth: 480, margin: "0 auto" }}>
      <h1
        style={{
          fontSize: 20,
          fontWeight: 700,
          letterSpacing: "0.08em",
          marginBottom: 24,
          color: "#F5F5F5",
        }}
      >
        МОИ ПРИГЛАШЕНИЯ
      </h1>

      {fetched && invitations.length === 0 && (
        <p style={{ color: "#888", fontSize: 14 }}>
          У вас нет новых приглашений
        </p>
      )}

      {invitations.map((inv) => (
        <div
          key={inv.id}
          style={{
            background: "#1A1A1A",
            borderRadius: 4,
            padding: "16px",
            marginBottom: 12,
            border: "1px solid #333",
          }}
        >
          <div
            style={{
              fontWeight: 700,
              fontSize: 15,
              color: "#F5F5F5",
              marginBottom: 4,
            }}
          >
            {inv.circle_name}
          </div>
          {inv.message && (
            <div
              style={{
                fontSize: 13,
                color: "#999",
                marginBottom: 12,
                lineHeight: 1.4,
              }}
            >
              {inv.message}
            </div>
          )}
          <div style={{ display: "flex", gap: 8, marginTop: 8 }}>
            <button
              onClick={() => handleAccept(inv.id)}
              disabled={loading}
              style={{
                background: "#E5FF00",
                color: "#1A1A1A",
                fontWeight: 700,
                fontSize: 13,
                letterSpacing: "0.05em",
                padding: "8px 16px",
                border: "none",
                cursor: loading ? "not-allowed" : "pointer",
                minHeight: 36,
                borderRadius: 2,
                opacity: loading ? 0.6 : 1,
              }}
            >
              ПРИНЯТЬ
            </button>
            <button
              onClick={() => handleDecline(inv.id)}
              disabled={loading}
              style={{
                background: "transparent",
                color: "#F5F5F5",
                fontSize: 13,
                letterSpacing: "0.05em",
                padding: "8px 16px",
                border: "1px solid #555",
                cursor: loading ? "not-allowed" : "pointer",
                minHeight: 36,
                borderRadius: 2,
                opacity: loading ? 0.6 : 1,
              }}
            >
              ОТКЛОНИТЬ
            </button>
          </div>
        </div>
      ))}
    </main>
  );
}
