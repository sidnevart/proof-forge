import { cn } from "@/lib/cn";
import { STATUS_LABELS, type UIStatus } from "@/lib/ui-copy";

import styles from "./status-pill.module.css";

type Props = {
  status: UIStatus;
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
