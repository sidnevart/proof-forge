"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";

import { ApiError, getDashboard, logoutUser } from "@/lib/api";
import { InvitationInbox } from "./invitation-inbox";
import styles from "./product-nav.module.css";

type User = { display_name: string; email: string };

export function ProductNav() {
  const pathname = usePathname();
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  // authReady gates the right-side render so we don't flash "ВОЙТИ" for an
  // already-logged-in user during the dashboard fetch on hard reload.
  // Mirrors the trichotomy used in landing-nav.tsx.
  const [authReady, setAuthReady] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const menuRef = useRef<HTMLDivElement | null>(null);

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

  // Close the dropdown on outside click and on Escape.
  useEffect(() => {
    if (!menuOpen) return;

    const onClick = (e: MouseEvent) => {
      if (!menuRef.current) return;
      if (!menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setMenuOpen(false);
    };

    document.addEventListener("mousedown", onClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [menuOpen]);

  async function handleLogout() {
    if (loggingOut) return;
    setLoggingOut(true);
    try {
      await logoutUser();
    } finally {
      // Always navigate even if the server call fails — the server clears
      // cookies regardless, and the user expects to land on the landing.
      setUser(null);
      setMenuOpen(false);
      router.push("/");
    }
  }

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
          <div className={styles.userMenu} ref={menuRef}>
            <button
              type="button"
              className={styles.userBtn}
              aria-haspopup="menu"
              aria-expanded={menuOpen}
              onClick={() => setMenuOpen((v) => !v)}
            >
              <span className={styles.avatar}>{user.display_name[0]?.toUpperCase()}</span>
              <span className={styles.userName}>{user.display_name.toUpperCase()}</span>
              <span className={styles.caret} aria-hidden="true">▾</span>
            </button>
            {menuOpen && (
              <div className={styles.menuPanel} role="menu">
                <Link
                  href="/me"
                  className={styles.menuItem}
                  role="menuitem"
                  onClick={() => setMenuOpen(false)}
                >
                  ПРОФИЛЬ
                </Link>
                <button
                  type="button"
                  className={`${styles.menuItem} ${styles.menuItemDanger}`}
                  role="menuitem"
                  onClick={handleLogout}
                  disabled={loggingOut}
                >
                  {loggingOut ? "ВЫХОД…" : "ВЫЙТИ"}
                </button>
              </div>
            )}
          </div>
        ) : (
          <Link href="/dashboard" className={styles.userBtn}>ВОЙТИ</Link>
        )}
      </div>
    </nav>
  );
}
