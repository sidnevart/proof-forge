import type { ReactNode } from "react";
import Link from "next/link";
import styles from "./settings.module.css";

const SETTINGS_LINKS = [
  { href: "/settings/sharing", label: "ПУБЛИЧНОСТЬ" },
  { href: "/settings/telegram", label: "TELEGRAM" },
];

export default function SettingsLayout({ children }: { children: ReactNode }) {
  return (
    <div className={styles.layout}>
      <aside className={styles.sidebar}>
        <div className={styles.sidebarHead}>
          <span className={styles.sidebarTitle}>НАСТРОЙКИ</span>
        </div>
        <nav className={styles.sidebarNav}>
          {SETTINGS_LINKS.map((link) => (
            <Link key={link.href} href={link.href} className={styles.sidebarLink}>
              {link.label}
              <span className={styles.sidebarChevron}>›</span>
            </Link>
          ))}
        </nav>
      </aside>
      <div className={styles.content}>{children}</div>
    </div>
  );
}
