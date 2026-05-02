"use client";

import Link from "next/link";
import { type FormEvent, useState, useTransition } from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/core/button";
import { SectionShell } from "@/components/core/section-shell";
import { StatePanel } from "@/components/core/state-panel";
import { ApiError, createGoal } from "@/lib/api";

import styles from "./goal-setup-screen.module.css";

export function GoalSetupScreen() {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [createdTitle, setCreatedTitle] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);

    const formData = new FormData(event.currentTarget);
    const title = String(formData.get("title") ?? "");

    startTransition(async () => {
      try {
        await createGoal({
          title,
          description: String(formData.get("description") ?? ""),
          buddy_name: String(formData.get("buddy_name") ?? ""),
          buddy_email: String(formData.get("buddy_email") ?? ""),
        });
        setCreatedTitle(title);
        setTimeout(() => router.push("/dashboard"), 1200);
      } catch (cause) {
        if (cause instanceof ApiError && cause.status === 401) {
          setError("Сначала войдите в систему, а затем вернитесь к созданию цели.");
          return;
        }

        setError(cause instanceof Error ? cause.message : "Не удалось создать цель.");
      }
    });
  }

  return (
    <main className={styles.page}>
      <header className={styles.header}>
        <div>
          <span className="eyebrow">Создание цели</span>
          <h1>Сначала фиксируем цель, потом ведём её по циклу</h1>
        </div>
      </header>

      {createdTitle ? (
        <StatePanel
          tone="success"
          title={`Цель «${createdTitle}» создана`}
          description="Приглашение партнёру отправлено. Возвращаем вас в операционный центр."
        />
      ) : null}

      <section className={styles.grid}>
        <SectionShell eyebrow="Шаг 1" title="Опишите цель">
          <form className={styles.form} onSubmit={handleSubmit}>
            <label className={styles.field}>
              <span>Название цели</span>
              <input name="title" placeholder="Например, запустить новый лендинг" required />
            </label>
            <label className={styles.field}>
              <span>Что должно считаться прогрессом</span>
              <textarea
                name="description"
                rows={5}
                placeholder="Опишите, какие материалы, результаты или изменения вы будете отправлять на проверку."
              />
            </label>
            <div className={styles.inlineFields}>
              <label className={styles.field}>
                <span>Имя партнёра</span>
                <input name="buddy_name" placeholder="Например, Мария" required />
              </label>
              <label className={styles.field}>
                <span>Почта партнёра</span>
                <input name="buddy_email" type="email" placeholder="partner@example.com" required />
              </label>
            </div>
            {error ? (
              <p className={styles.error} role="alert">
                {error}
              </p>
            ) : null}
            <div className={styles.actions}>
              <Button type="submit" disabled={isPending}>
                {isPending ? "Создаём цель..." : "Создать цель"}
              </Button>
              <Link className={styles.backLink} href="/dashboard">
                Вернуться в центр управления
              </Link>
            </div>
          </form>
        </SectionShell>

        <SectionShell eyebrow="Шаг 2" title="Что произойдёт дальше">
          <div className={styles.copy}>
            <p>
              После создания цели партнёр получит приглашение. Пока он не примет участие,
              цель будет находиться в статусе ожидания.
            </p>
            <ul className={styles.rules}>
              <li>Подтверждения прогресса вы отправляете уже после принятия приглашения.</li>
              <li>Партнёр принимает решение по движению цели, а не просто читает обновления.</li>
              <li>Еженедельная сводка начнёт собирать картину по мере появления проверок.</li>
            </ul>
          </div>
        </SectionShell>
      </section>
    </main>
  );
}
