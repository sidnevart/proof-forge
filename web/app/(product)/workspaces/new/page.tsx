import Link from "next/link";
import { WorkspaceSetupScreen } from "@/components/product/workspace-setup-screen";
import styles from "./page.module.css";

export const metadata = { title: "Создать пространство" };

export default function NewWorkspacePage() {
  return (
    <div className={`page-shell ${styles.page}`}>
      <header className={styles.header}>
        <Link href="/workspaces" className={styles.back}>
          ← Пространства
        </Link>
      </header>
      <WorkspaceSetupScreen />
    </div>
  );
}
