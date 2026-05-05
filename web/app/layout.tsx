import type { Metadata } from "next";
import type { ReactNode } from "react";
import { Bebas_Neue, Inter, JetBrains_Mono } from "next/font/google";

import "./globals.css";

const displayFont = Bebas_Neue({
  weight: "400",
  subsets: ["latin"],
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
  title: "ProofForge — круг друзей, где не сдать = заморозка",
  description:
    "Соревнуйся с друзьями за дисциплину. Сдавай пруф каждый день. Пропустил — все видят.",
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
