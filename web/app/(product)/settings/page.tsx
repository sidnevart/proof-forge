"use client";

import Link from "next/link";
import styles from "./settings.module.css";

const SETTINGS_LINKS = [
  {
    href: "/settings/sharing",
    label: "ПУБЛИЧНОСТЬ",
    desc: "Настройки видимости пруфов и псевдонима",
  },
  {
    href: "/settings/telegram",
    label: "TELEGRAM",
    desc: "Подключение бота для уведомлений",
  },
];

export default function SettingsIndexPage() {
  return (
    <div>
      <h1 className={styles.indexTitle}>НАСТРОЙКИ</h1>
      <nav className={styles.indexNav}>
        {SETTINGS_LINKS.map((link) => (
          <Link key={link.href} href={link.href} className={styles.indexLink}>
            <span className={styles.indexLinkLabel}>{link.label}</span>
            <span className={styles.indexLinkDesc}>{link.desc}</span>
            <span className={styles.indexLinkArrow}>→</span>
          </Link>
        ))}
      </nav>
    </div>
  );
}
