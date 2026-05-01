import type { ButtonHTMLAttributes, PropsWithChildren } from "react";

import { cn } from "@/lib/cn";

import styles from "./button.module.css";

type Variant = "primary" | "secondary" | "ghost";

type Props = PropsWithChildren<ButtonHTMLAttributes<HTMLButtonElement>> & {
  variant?: Variant;
};

export function Button({ children, className, variant = "primary", ...props }: Props) {
  return (
    <button className={cn(styles.button, styles[variant], className)} {...props}>
      {children}
    </button>
  );
}
