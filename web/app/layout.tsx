import type { Metadata } from "next";
import type { ReactNode } from "react";
import { Oswald, Inter, JetBrains_Mono } from "next/font/google";

import "./globals.css";

const displayFont = Oswald({
  weight: "700",
  subsets: ["latin", "cyrillic"],
  variable: "--display",
  display: "swap",
});

const bodyFont = Inter({
  subsets: ["latin", "cyrillic"],
  variable: "--body",
  weight: ["400", "500", "700"],
  display: "swap",
});

const monoFont = JetBrains_Mono({
  subsets: ["latin", "cyrillic"],
  variable: "--mono",
  weight: ["400", "500"],
  display: "swap",
});

export const metadata: Metadata = {
  title: "ProofForge — среда где движение становится нормой",
  description:
    "Сильная среда. Здоровая конкуренция. Не планируешь — делаешь. Проверено кругом людей рядом.",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="ru" className={`${displayFont.variable} ${bodyFont.variable} ${monoFont.variable}`}>
      <body>
        <div className="app-shell">{children}</div>
      </body>
    </html>
  );
}
