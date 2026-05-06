"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import styles from "./bottom-nav.module.css";

function IconDashboard({ active }: { active: boolean }) {
  return (
    <svg
      width="22"
      height="22"
      viewBox="0 0 24 24"
      fill="none"
      stroke={active ? "var(--ink-primary)" : "var(--ink-mono)"}
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <rect x="3" y="3" width="7" height="7" />
      <rect x="14" y="3" width="7" height="7" />
      <rect x="14" y="14" width="7" height="7" />
      <rect x="3" y="14" width="7" height="7" />
    </svg>
  );
}

function IconFeed({ active }: { active: boolean }) {
  return (
    <svg
      width="22"
      height="22"
      viewBox="0 0 24 24"
      fill="none"
      stroke={active ? "var(--ink-primary)" : "var(--ink-mono)"}
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M4 6h16M4 12h16M4 18h10" />
    </svg>
  );
}

function IconProfile({ active }: { active: boolean }) {
  return (
    <svg
      width="22"
      height="22"
      viewBox="0 0 24 24"
      fill="none"
      stroke={active ? "var(--ink-primary)" : "var(--ink-mono)"}
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <circle cx="12" cy="8" r="4" />
      <path d="M4 20c0-4 3.6-7 8-7s8 3 8 7" />
    </svg>
  );
}

const tabs = [
  {
    href: "/dashboard",
    label: "ДАШБОРД",
    Icon: IconDashboard,
  },
  {
    href: "/feed",
    label: "ЛЕНТА",
    Icon: IconFeed,
  },
  {
    href: "/me",
    label: "ПРОФИЛЬ",
    Icon: IconProfile,
  },
];

export function BottomNav() {
  const pathname = usePathname();

  return (
    <nav className={styles.nav} aria-label="Основная навигация">
      {tabs.map(({ href, label, Icon }) => {
        const active = pathname === href || pathname.startsWith(href + "/");
        return (
          <Link
            key={href}
            href={href}
            className={`${styles.tab} ${active ? styles.active : ""}`}
            aria-current={active ? "page" : undefined}
          >
            {active && <span className={styles.indicator} aria-hidden="true" />}
            <Icon active={active} />
            <span className={styles.label}>{label}</span>
          </Link>
        );
      })}
    </nav>
  );
}
