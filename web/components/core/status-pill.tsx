import { cn } from "@/lib/cn";

import styles from "./status-pill.module.css";

type Status = "approved" | "pending" | "changes_requested" | "rejected" | "active";

type Props = {
  status: Status;
  label?: string;
};

const LABELS: Record<Status, string> = {
  active: "Active",
  approved: "Approved",
  pending: "Pending",
  changes_requested: "Needs proof",
  rejected: "Rejected",
};

export function StatusPill({ status, label }: Props) {
  return (
    <span className={cn(styles.pill, styles[status])}>
      <span className="status-dot" />
      {label ?? LABELS[status]}
    </span>
  );
}
