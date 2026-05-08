import Link from "next/link";
import { BuddyDashboard } from "@/components/product/buddy-dashboard";
import styles from "./page.module.css";

export const metadata = { title: "Buddy Dashboard" };

export default function BuddyPage() {
  return (
    <div className={`page-shell ${styles.page}`}>
      <header className={styles.header}>
        <Link href="/dashboard" className={styles.back}>
          ← Главная
        </Link>
        <h1 className={styles.title}>Buddy</h1>
      </header>
      <BuddyDashboard />
    </div>
  );
}
