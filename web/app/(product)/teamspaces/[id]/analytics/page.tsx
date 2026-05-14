import Link from "next/link";
import { AINotificationCenter } from "@/components/product/ai-notification-center";
import { TeamspaceAnalytics } from "@/components/product/teamspace-analytics";
import styles from "./page.module.css";

interface Props {
  params: Promise<{ id: string }>;
}

export default async function TeamspaceAnalyticsPage({ params }: Props) {
  const { id } = await params;
  return (
    <div className={`page-shell ${styles.page}`}>
      <header className={styles.header}>
        <Link href={`/teams/${id}`} className={styles.back}>
          ← Команда
        </Link>
        <h1 className={styles.title}>Аналитика</h1>
      </header>
      <AINotificationCenter />
      <TeamspaceAnalytics teamspaceId={Number(id)} />
    </div>
  );
}
