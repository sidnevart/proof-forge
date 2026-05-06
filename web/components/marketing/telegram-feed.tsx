"use client";

import { useEffect, useRef, useState } from "react";

import styles from "./telegram-feed.module.css";

export interface TgMessage {
  icon: string;
  text: string;
  kind: "win" | "warn" | "danger";
}

interface TelegramFeedProps {
  messages: TgMessage[];
}

interface FeedItem {
  msg: TgMessage;
  /** Unique sequence number — used as key to prevent collisions */
  seq: number;
  /** "enter" while animating in, "visible" while stable, "exit" while animating out */
  state: "enter" | "visible" | "exit";
}

/**
 * Cycling Telegram-style message feed.
 *
 * Rotates through `messages` every 2500ms. New messages slide in from the top;
 * the bottom message fades out before removal.
 *
 * Under prefers-reduced-motion: static snapshot of the first 4 messages, no cycling.
 */
export function TelegramFeed({ messages }: TelegramFeedProps) {
  const reduceMotion = useRef(false);
  const seqRef = useRef(messages.length);

  // Seed initial visible items
  const [items, setItems] = useState<FeedItem[]>(() =>
    messages.slice(0, 4).map((msg, i) => ({ msg, seq: i, state: "visible" }))
  );

  useEffect(() => {
    if (typeof window === "undefined") return;
    reduceMotion.current = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (reduceMotion.current) return;

    let msgIdx = 0;
    const CYCLE = 2500;

    const interval = setInterval(() => {
      const nextMsg = messages[msgIdx % messages.length];
      msgIdx++;
      const seq = seqRef.current++;

      setItems((prev) => {
        // Mark the last (oldest) item as "exit"
        const updated = prev.map((item, i) =>
          i === prev.length - 1 ? { ...item, state: "exit" as const } : item
        );
        // Prepend new item in "enter" state
        return [{ msg: nextMsg, seq, state: "enter" as const }, ...updated];
      });

      // After exit animation completes (200ms), remove the exiting item
      // and transition the entering item to "visible"
      setTimeout(() => {
        setItems((prev) =>
          prev
            .filter((item) => item.state !== "exit")
            .map((item) =>
              item.seq === seq ? { ...item, state: "visible" as const } : item
            )
        );
      }, 210);
    }, CYCLE);

    return () => clearInterval(interval);
  }, [messages]);

  return (
    <div className={styles.feed} aria-live="polite" aria-atomic="false">
      {items.map((item) => (
        <div
          key={`${item.msg.text}-${item.seq}`}
          className={[
            styles.msg,
            styles[`msg_${item.msg.kind}`],
            item.state === "enter" ? styles.msgEnter : "",
            item.state === "exit" ? styles.msgExit : "",
          ]
            .filter(Boolean)
            .join(" ")}
        >
          <span className={styles.icon}>{item.msg.icon}</span>
          <span className={styles.text}>{item.msg.text}</span>
        </div>
      ))}
    </div>
  );
}
