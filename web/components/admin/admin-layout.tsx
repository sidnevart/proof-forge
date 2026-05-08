"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import styles from "./admin-layout.module.css";

function AdminNavLink({ href, label }: { href: string; label: string }) {
  const pathname = usePathname();
  const active = pathname.startsWith(href);
  return (
    <Link href={href} className={`${styles.navLink} ${active ? styles.navLinkActive : ""}`}>
      {label}
    </Link>
  );
}

export function AdminLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className={styles.layout}>
      <aside className={styles.sidebar}>
        <div className={styles.logo}>ADMIN</div>
        <nav className={styles.nav}>
          <AdminNavLink href="/admin/workspaces" label="Workspace" />
          <AdminNavLink href="/admin/users" label="Пользователи" />
        </nav>
        <div className={styles.sidebarFooter}>
          <Link href="/dashboard" className={styles.exitLink}>← В продукт</Link>
        </div>
      </aside>
      <main className={styles.main}>{children}</main>
    </div>
  );
}
