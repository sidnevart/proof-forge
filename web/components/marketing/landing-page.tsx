import Link from "next/link";

import { landingHighlights } from "@/lib/demo-data";
import { Button } from "@/components/core/button";
import { SectionShell } from "@/components/core/section-shell";
import { StatusPill } from "@/components/core/status-pill";

import styles from "./landing-page.module.css";

export function LandingPage() {
  return (
    <main className={styles.page}>
      <section className={styles.hero}>
        <div className={styles.heroCopy}>
          <span className="eyebrow">Proof-based accountability</span>
          <h1>
            Держите серьёзные обязательства через проверяемый прогресс, а не через
            самонаблюдение.
          </h1>
          <p>
            Поставьте цель, отправляйте proof, получайте подтверждение от buddy и
            видьте реальное движение вперёд без дрейфа в habit tracker.
          </p>
          <div className={styles.actions}>
            <Button>Начать первый checkpoint</Button>
            <Link href="/dashboard">
              <Button variant="secondary">Открыть demo dashboard</Button>
            </Link>
          </div>
        </div>
        <div className={styles.heroPanel}>
          <div className={styles.commandBar}>
            <span>Mission control</span>
            <StatusPill status="active" label="Loop active" />
          </div>
          <div className={styles.signalGrid}>
            {landingHighlights.map((item) => (
              <div className={styles.signalCard} key={item}>
                <span className={styles.signalLabel}>Signal</span>
                <strong>{item}</strong>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className={styles.sections}>
        <SectionShell eyebrow="Why this is different" title="Это не трекер привычек">
          <div className={styles.copyGrid}>
            <p>
              Здесь не считают streaks ради галочек. Здесь подтверждают, что важное
              обязательство действительно двигается.
            </p>
            <p>
              Buddy approval остаётся источником правды, а AI weekly recap только
              помогает собрать картину недели, не подменяя review.
            </p>
          </div>
        </SectionShell>

        <SectionShell eyebrow="Core loop" title="Goal → proof → review → visible progress">
          <ol className={styles.loop}>
            <li>Цель формулируется явно.</li>
            <li>Check-in уходит с text, link и file/image proof.</li>
            <li>Buddy approves, requests changes или rejects.</li>
            <li>Система обновляет статус движения и готовит weekly recap.</li>
          </ol>
        </SectionShell>
      </section>
    </main>
  );
}
