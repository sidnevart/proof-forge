"use client";

import { useEffect, useReducer, useRef, useState } from "react";

import styles from "./live-circle-mock.module.css";

type MockMember = {
  name: string;
  streak: number;
  status: "approved" | "waiting" | "at_risk" | "dropped";
  rank: number;
};

type MockEvent = {
  icon: string;
  text: string;
  kind: "win" | "warn" | "danger";
};

const MEMBERS: MockMember[] = [
  { name: "АРТЁМ", streak: 14, status: "approved", rank: 1 },
  { name: "МАША", streak: 11, status: "approved", rank: 2 },
  { name: "ДЕНИС", streak: 9, status: "waiting", rank: 3 },
  { name: "ИЛЬЯ", streak: 6, status: "at_risk", rank: 4 },
  { name: "КАТЯ", streak: 3, status: "at_risk", rank: 5 },
  { name: "ЮЛЯ", streak: 0, status: "dropped", rank: 6 },
  { name: "ТЫ", streak: 0, status: "at_risk", rank: 7 },
];

const EVENTS: MockEvent[] = [
  { icon: "✅", text: "АРТЁМ СДАЛ ПЕРВЫМ.", kind: "win" },
  { icon: "🥶", text: "ЮЛЯ ЗАМОРОЖЕНА. НЕ ПРИСОЕДИНЯЙСЯ.", kind: "danger" },
  { icon: "⚡", text: "МАША. СЕРИЯ 11. РЕКОРД КРУГА.", kind: "win" },
  { icon: "⚠️", text: "ТЫ В ШАГЕ ОТ ЗАМОРОЗКИ.", kind: "warn" },
  { icon: "✅", text: "ДЕНИС СДАЛ. ЖДЁТ ПРОВЕРКИ.", kind: "win" },
  { icon: "⚡", text: "КАМБЭК. КАТЯ ВЕРНУЛАСЬ.", kind: "win" },
  { icon: "⚠️", text: "ИЛЬЯ НЕ СДАЛ ВТО́РУЮ НЕДЕЛЮ.", kind: "warn" },
  { icon: "🔥", text: "АРТЁМ ОБОГНАЛ ВСЕХ. РАНГ 1.", kind: "win" },
];

const TICK_MS = 3500;

function formatCountdown(ms: number): string {
  const t = Math.floor(ms / 1000);
  const h = Math.floor(t / 3600);
  const m = Math.floor((t % 3600) / 60);
  const s = t % 60;
  return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
}

function getDeadlineMs(): number {
  const target = new Date();
  target.setHours(23, 59, 0, 0);
  const now = Date.now();
  if (target.getTime() <= now) target.setDate(target.getDate() + 1);
  return Math.max(0, target.getTime() - now);
}

export function LiveCircleMock() {
  const [countdown, setCountdown] = useState(() => formatCountdown(getDeadlineMs()));
  const [visibleEvents, setVisibleEvents] = useState<MockEvent[]>(EVENTS.slice(0, 3));
  const eventIndexRef = useRef(3);
  const [, forceUpdate] = useReducer((x: number) => x + 1, 0);

  useEffect(() => {
    const id = setInterval(() => setCountdown(formatCountdown(getDeadlineMs())), 1000);
    return () => clearInterval(id);
  }, []);

  useEffect(() => {
    const id = setInterval(() => {
      const next = EVENTS[eventIndexRef.current % EVENTS.length];
      eventIndexRef.current++;
      setVisibleEvents((prev) => [next, ...prev.slice(0, 2)]);
      forceUpdate();
    }, TICK_MS);
    return () => clearInterval(id);
  }, []);

  const visibleMembers = MEMBERS.slice(0, 7);

  return (
    <div className={styles.root}>
      <div className={styles.header}>
        <div className={styles.headerLeft}>
          <span className={styles.circleName}>СТАЯ</span>
          <span className={styles.circleDay}>ДЕНЬ 12 / 28</span>
        </div>
        <span className={styles.countdown}>{countdown}</span>
      </div>

      <div className={styles.standings}>
        {visibleMembers.map((member) => (
          <div
            key={member.name}
            className={`${styles.member} ${member.status === "dropped" ? styles.memberFrozen : ""} ${member.name === "ТЫ" ? styles.memberMe : ""}`}
          >
            <div className={`${styles.rank} ${getRankClass(member.rank)}`}>{member.rank}</div>
            <div className={`${styles.avatar} ${member.status === "dropped" ? styles.avatarFrozen : ""}`}>
              {member.name[0]}
              {member.status === "dropped" && <div className={styles.frozenCross} />}
            </div>
            <span className={styles.memberName}>{member.name}</span>
            <span className={styles.memberStreak}>
              {member.streak > 7 ? "🔥" : member.streak > 0 ? "⚡" : ""}
              {member.streak > 0 && ` ${member.streak}`}
            </span>
            <span className={`${styles.statusBadge} ${getStatusClass(member.status)}`}>
              {formatMemberStatus(member.status)}
            </span>
          </div>
        ))}
      </div>

      <div className={styles.pulseSection}>
        <div className={styles.pulseHeader}>ПУЛЬС</div>
        <div className={styles.pulseEvents}>
          {visibleEvents.map((event, i) => (
            <div
              key={`${event.text}-${i}`}
              className={`${styles.pulseEvent} ${styles[`event_${event.kind}`]} ${i === 0 ? styles.newEvent : ""}`}
            >
              <span className={styles.pulseIcon}>{event.icon}</span>
              <span className={styles.pulseText}>{event.text}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function getRankClass(rank: number): string {
  if (rank === 1) return styles.rankGold;
  if (rank <= 3) return styles.rankSilver;
  return styles.rankGray;
}

function getStatusClass(status: MockMember["status"]): string {
  switch (status) {
    case "approved": return styles.statusWin;
    case "waiting": return styles.statusPending;
    case "at_risk": return styles.statusWarn;
    case "dropped": return styles.statusDanger;
  }
}

function formatMemberStatus(status: MockMember["status"]): string {
  switch (status) {
    case "approved": return "СДАЛ";
    case "waiting": return "ЖДЁТ";
    case "at_risk": return "НЕ СДАЛ";
    case "dropped": return "🥶 ЗАМОРОЖЕН";
  }
}
