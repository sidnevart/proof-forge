import { cn } from "@/lib/cn";
import { STATUS_LABELS } from "@/lib/ui-copy";

import styles from "./status-pill.module.css";

type Status = "approved" | "pending" | "changes_requested" | "rejected" | "active";

type Props = {
  status: Status;
  label?: string;
};

export function StatusPill({ status, label }: Props) {
  return (
    <span className={cn(styles.pill, styles[status])}>
      <span className="status-dot" />
      {label ?? STATUS_LABELS[status]}
    </span>
  );
}
