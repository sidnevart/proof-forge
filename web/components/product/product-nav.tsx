"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";

import { ApiError, getDashboard } from "@/lib/api";
import { InvitationInbox } from "./invitation-inbox";
import styles from "./product-nav.module.css";

type User = { display_name: string; email: string };

export function ProductNav() {
  const pathname = usePathname();
  const [user, setUser] = useState<User | null>(null);
  // authReady gates the right-side render so we don't flash "ВОЙТИ" for an
  // already-logged-in user during the dashboard fetch on hard reload.
  // Mirrors the trichotomy used in landing-nav.tsx.
  const [authReady, setAuthReady] = useState(false);

  useEffect(() => {
    getDashboard()
      .then((d) => setUser(d.user))
      .catch((err) => {
        // Only a real 401 means the session is gone — wipe the user.
        // Network errors, 5xx, timeouts must NOT log the user out: that's the
        // root cause behind «меня выкинуло» reports — a transient failure here
        // showed «ВОЙТИ» to an already-authenticated person.
        if (err instanceof ApiError && err.status === 401) {
          setUser(null);
        }
      })
      .finally(() => setAuthReady(true));
  }, []);

  const links = [
    { href: "/dashboard", label: "ДАШБОРД" },
    { href: "/goals/new", label: "КРУГ" },
    { href: "/feed", label: "ЛЕНТА" },
  ];

  return (
    <nav className={styles.nav}>
      <Link href="/dashboard" className={styles.logo}>PF</Link>

      <div className={styles.links}>
        {links.map(({ href, label }) => (
          <Link
            key={href}
            href={href}
            className={`${styles.link} ${pathname === href ? styles.active : ""}`}
          >
            {label}
          </Link>
        ))}
      </div>

      <div className={styles.right}>
        <InvitationInbox />
        {!authReady ? (
          <span
            className={styles.authSlot}
            aria-busy="true"
            aria-label="Загрузка профиля"
          />
        ) : user ? (
          <Link href="/me" className={styles.userBtn}>
            <span className={styles.avatar}>{user.display_name[0]?.toUpperCase()}</span>
            <span className={styles.userName}>{user.display_name.toUpperCase()}</span>
          </Link>
        ) : (
          <Link href="/dashboard" className={styles.userBtn}>ВОЙТИ</Link>
        )}
      </div>
    </nav>
  );
}
