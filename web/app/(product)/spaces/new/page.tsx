import Link from "next/link";
import { SpaceSetupWizard } from "@/components/product/space-setup-wizard";
import styles from "./page.module.css";

export const metadata = { title: "Создать пространство" };

interface Props {
  searchParams: Promise<{ workspace_id?: string; workspace_type?: string }>;
}

export default async function NewSpacePage({ searchParams }: Props) {
  const params = await searchParams;
  const workspaceId = params.workspace_id ? Number(params.workspace_id) : undefined;
  const workspaceType = (params.workspace_type as "organization" | "community") ?? undefined;

  return (
    <div className={`page-shell ${styles.page}`}>
      <header className={styles.header}>
        <Link href={workspaceId ? `/workspaces/${workspaceId}` : "/workspaces"} className={styles.back}>
          ← Назад
        </Link>
      </header>
      <SpaceSetupWizard workspaceId={workspaceId} workspaceType={workspaceType} />
    </div>
  );
}
