import type { Metadata } from "next";
import type { ReactNode } from "react";

import "./globals.css";

export const metadata: Metadata = {
  title: "ProofForge",
  description: "Система внешней ответственности для серьёзных целей и подтверждённого прогресса.",
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
