import Link from "next/link";

import { landingScenarios, landingSignals } from "@/lib/demo-data";
import { Button } from "@/components/core/button";
import { SectionShell } from "@/components/core/section-shell";
import { StatusPill } from "@/components/core/status-pill";

import styles from "./landing-page.module.css";

export function LandingPage() {
  return (
    <main className={styles.page}>
      <section className={styles.hero}>
        <div className={styles.heroCopy}>
          <span className="eyebrow">Внешняя система ответственности</span>
          <h1>Внешняя система ответственности для серьёзных целей</h1>
          <p>
            ProofForge помогает держать цель под наблюдением: вы формулируете
            обязательство, отправляете подтверждение движения, партнёр выносит решение,
            а еженедельная сводка собирает общую картину.
          </p>
          <div className={styles.actions}>
            <Link href="/goals/new">
              <Button>Начать первую цель</Button>
            </Link>
            <Link href="/dashboard">
              <Button variant="secondary">Посмотреть рабочий экран</Button>
            </Link>
          </div>
        </div>

        <div className={styles.heroPanel}>
          <div className={styles.commandBar}>
            <span>Как это выглядит в работе</span>
            <StatusPill status="active" label="Цикл активен" />
          </div>

          <div className={styles.signalGrid}>
            {landingSignals.map((item) => (
              <div className={styles.signalCard} key={item}>
                <span className={styles.signalLabel}>Сигнал</span>
                <strong>{item}</strong>
              </div>
            ))}
          </div>

          <div className={styles.roleGrid}>
            <div className={styles.roleCard}>
              <span className={styles.signalLabel}>Вы</span>
              <p>Формулируете цель и отправляете подтверждение прогресса.</p>
            </div>
            <div className={styles.roleCard}>
              <span className={styles.signalLabel}>Партнёр</span>
              <p>Проверяет подтверждение и подтверждает или возвращает на доработку.</p>
            </div>
            <div className={styles.roleCard}>
              <span className={styles.signalLabel}>Сводка</span>
              <p>Собирает недельную картину, не вмешиваясь в решение по прогрессу.</p>
            </div>
          </div>
        </div>
      </section>

      <section className={styles.sections}>
        <SectionShell eyebrow="Как работает продукт" title="Как работает цикл">
          <ol className={styles.loop}>
            <li>Вы создаёте цель и приглашаете партнёра к наблюдению за ней.</li>
            <li>По ходу работы отправляете подтверждение: текст, ссылку, файл или изображение.</li>
            <li>Партнёр проверяет подтверждение и выносит решение по прогрессу.</li>
            <li>Еженедельная сводка показывает ритм движения и узкие места по цели.</li>
          </ol>
        </SectionShell>

        <SectionShell eyebrow="Сценарии использования" title="Где это особенно полезно">
          <div className={styles.scenarioGrid}>
            {landingScenarios.map((scenario) => (
              <article className={styles.scenarioCard} key={scenario.title}>
                <strong>{scenario.title}</strong>
                <p>{scenario.detail}</p>
              </article>
            ))}
          </div>
        </SectionShell>

        <SectionShell eyebrow="Почему не трекер привычек" title="Что здесь считается прогрессом">
          <div className={styles.copyGrid}>
            <p>
              В ProofForge нельзя просто отметить действие и объявить его прогрессом.
              Важен наблюдаемый результат, который можно показать и проверить.
            </p>
            <p>
              Решение о движении цели принимает не система и не сам пользователь, а
              партнёр, который видит подтверждение и подтверждает его достаточность.
            </p>
          </div>
        </SectionShell>
      </section>
    </main>
  );
}
