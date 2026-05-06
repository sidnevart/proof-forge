"use client";

import { useEffect, useRef, useState } from "react";

import { usePolling } from "@/lib/use-polling";
import styles from "./circle-pulse.module.css";
import type { CircleEvent } from "@/lib/types";

type Props = {
  events: CircleEvent[];
  circleId?: number;
  onRefresh?: () => Promise<CircleEvent[]>;
  pollIntervalMs?: number;
};

export function CirclePulse({ events: initialEvents, onRefresh, pollIntervalMs = 15_000 }: Props) {
  const [events, setEvents] = useState<CircleEvent[]>(initialEvents);
  const prevCountRef = useRef(initialEvents.length);

  useEffect(() => {
    setEvents(initialEvents);
  }, [initialEvents]);

  usePolling(
    async () => {
      if (!onRefresh) return;
      const fresh = await onRefresh();
      setEvents(fresh);
    },
    pollIntervalMs,
    { enabled: Boolean(onRefresh), visibilityAware: true },
  );

  const silenceMinutes = getSilenceMinutes(events);

  return (
    <div className={styles.root}>
      <div className={styles.header}>
        {silenceMinutes > 0 ? (
          <span className={styles.silenceTag}>{silenceMinutes} МИН ТИШИНА</span>
        ) : (
          <span className={styles.activeTag}>ТОЛЬКО ЧТО · АКТИВНОСТЬ</span>
        )}
        <span className={styles.title}>ПУЛЬС</span>
      </div>

      <div className={styles.feed}>
        {events.length === 0 ? (
          <div className={styles.empty}>
            <span className={styles.emptyText}>ТИШИНА В КРУГЕ. ИДИ ПЕРВЫМ.</span>
          </div>
        ) : (
          events.map((event, idx) => (
            <div
              key={`${event.kind}-${event.user_id}-${event.occurred_at}`}
              className={`${styles.event} ${getEventClass(event.kind)} ${idx < (events.length - prevCountRef.current) ? styles.newEvent : ""}`}
            >
              <span className={styles.eventIcon}>{getEventIcon(event.kind)}</span>
              <span className={styles.eventMessage}>{event.message.toUpperCase()}</span>
              <span className={styles.eventTime}>{formatTime(event.occurred_at)}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

function getSilenceMinutes(events: CircleEvent[]): number {
  if (events.length === 0) return 999;
  const latest = new Date(events[0].occurred_at).getTime();
  return Math.floor((Date.now() - latest) / 60_000);
}

function getEventClass(kind: CircleEvent["kind"]): string {
  switch (kind) {
    case "approved": return styles.eventApproved;
    case "comeback": return styles.eventComeback;
    case "dropped": return styles.eventDropped;
    case "at_risk": return styles.eventAtRisk;
    default: return "";
  }
}

function getEventIcon(kind: CircleEvent["kind"]): string {
  switch (kind) {
    case "approved": return "✅";
    case "comeback": return "⚡";
    case "dropped": return "🥶";
    case "at_risk": return "⚠️";
    default: return "·";
  }
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  const h = String(d.getHours()).padStart(2, "0");
  const m = String(d.getMinutes()).padStart(2, "0");
  return `${h}:${m}`;
}
