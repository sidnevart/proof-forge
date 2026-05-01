import type { Metadata } from "next";
import type { ReactNode } from "react";

import "./globals.css";

export const metadata: Metadata = {
  title: "ProofForge",
  description: "Proof-based social accountability for serious commitments.",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="ru">
      <body>
        <div className="grid-lines" />
        <div className="app-shell">{children}</div>
      </body>
    </html>
  );
}
