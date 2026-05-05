"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { getDashboard } from "@/lib/api";
import styles from "./landing-nav.module.css";

type AuthState = "loading" | "in" | "out";

export function LandingNav() {
  const [auth, setAuth] = useState<AuthState>("loading");
  const [name, setName] = useState("");

  useEffect(() => {
    getDashboard()
      .then((d) => {
        setName(d.user.display_name.toUpperCase());
        setAuth("in");
      })
      .catch(() => setAuth("out"));
  }, []);

  return (
    <nav className={styles.nav}>
      <Link href="/" className={styles.logo}>PROOFFORGE</Link>

      <div className={styles.actions}>
        {auth === "loading" && (
          <span className={styles.loading}>·</span>
        )}
        {auth === "out" && (
          // Single CTA for anonymous visitors. Both the previous "ВОЙТИ" and
          // "ВОЙТИ В КРУГ" linked to /dashboard, where the same form handles
          // login (existing email) and registration (new email). One button =
          // less decision fatigue, no false "join a circle" promise before
          // the user has an account.
          <Link href="/dashboard" className={styles.cta}>НАЧАТЬ</Link>
        )}
        {auth === "in" && (
          <>
            <span className={styles.name}>{name}</span>
            <Link href="/dashboard" className={styles.cta}>В ДАШБОРД</Link>
          </>
        )}
      </div>
    </nav>
  );
}
