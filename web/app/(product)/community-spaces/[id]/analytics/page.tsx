import Link from "next/link";
import { AINotificationCenter } from "@/components/product/ai-notification-center";
import { CommunityAnalytics } from "@/components/product/community-analytics";
import styles from "./page.module.css";

interface Props {
  params: Promise<{ id: string }>;
}

export default async function CommunityAnalyticsPage({ params }: Props) {
  const { id } = await params;
  return (
    <div className={`page-shell ${styles.page}`}>
      <header className={styles.header}>
        <Link href="/dashboard" className={styles.back}>← Назад</Link>
        <h1 className={styles.title}>Аналитика сообщества</h1>
      </header>
      <AINotificationCenter />
      <CommunityAnalytics communityId={Number(id)} />
    </div>
  );
}
