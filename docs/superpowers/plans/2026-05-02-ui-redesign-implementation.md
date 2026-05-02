# UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Перестроить web UI ProofForge в русский guided-интерфейс с операционным центром, понятным лендингом и оформленным письмом-приглашением.

**Architecture:** Сначала обновить общие UI-примитивы и словарь, затем перестроить маркетинговый и продуктовый экраны вокруг одной приоритетной цели и блока `Следующий шаг`, после чего довести короткие сценарии приглашения/подтверждения/проверки и письмо-приглашение. Изменения остаются в существующем Next.js web app и Go email sender без новой инфраструктуры.

**Tech Stack:** Next.js App Router, React, CSS Modules, Vitest/Testing Library, Go SMTP sender

---

### Task 1: Общий словарь и визуальная база

**Files:**
- Modify: `web/app/layout.tsx`
- Modify: `web/app/globals.css`
- Modify: `web/components/core/status-pill.tsx`
- Modify: `web/components/core/state-panel.tsx`
- Test: `web/components/core/state-panel.test.tsx`

- [ ] Зафиксировать русский словарь в shared-компонентах и метаданных.
- [ ] Обновить глобальную типографику и токены под более строгую и читаемую иерархию.
- [ ] Проверить, что shared-тесты покрывают новые подписи статусов.

### Task 2: Лендинг

**Files:**
- Modify: `web/components/marketing/landing-page.tsx`
- Modify: `web/components/marketing/landing-page.module.css`
- Modify: `web/lib/demo-data.ts`
- Test: `web/app/page.tsx` through component tests if needed

- [ ] Пересобрать лендинг как объяснение цикла `цель → подтверждение → проверка → сводка`.
- [ ] Показать сценарии использования и роли пользователя/партнёра.
- [ ] Заменить CTA и англоязычные подписи на русский operational tone.

### Task 3: Операционный центр

**Files:**
- Modify: `web/components/product/dashboard-screen.tsx`
- Modify: `web/components/product/dashboard-screen.module.css`
- Modify: `web/lib/demo-data.ts`
- Test: `web/components/product/dashboard-screen.test.tsx`

- [ ] Перестроить главный экран вокруг одной приоритетной цели и блока `Следующий шаг`.
- [ ] Вынести обзорные surfaces в `Карточка цели`, `Состояние прогресса`, `Статус партнёра`, `История подтверждений`, `Еженедельная сводка`.
- [ ] Убрать смешение регистрации, обзора и создания цели в один визуальный слой.

### Task 4: Guided flows

**Files:**
- Modify: `web/components/product/invite-accept-screen.tsx`
- Modify: `web/components/product/invite-accept-screen.module.css`
- Modify: `web/components/product/checkin-screen.tsx`
- Modify: `web/components/product/checkin-screen.module.css`
- Modify: `web/components/product/approval-panel.tsx`
- Modify: `web/components/product/approval-panel.module.css`
- Test: `web/components/product/invite-accept-screen.test.tsx`
- Test: `web/components/product/checkin-screen.test.tsx`
- Test: `web/components/product/approval-panel.test.tsx`

- [ ] Сделать приглашение, подтверждение и проверку короткими русскоязычными сценариями.
- [ ] Усилить mobile-first и action-first поведение на этих экранах.
- [ ] Вычистить англоязычные статусы и CTA.

### Task 5: Email-приглашение

**Files:**
- Modify: `backend/internal/platform/email/smtp_sender.go`
- Test: `backend/internal/platform/email` via new unit tests if needed

- [ ] Перевести письмо-приглашение с plain text-only на multipart HTML + text.
- [ ] Оформить письмо как строгую карточку приглашения с одним главным действием.
- [ ] Сохранить простой SMTP sender без внешнего шаблонизатора.

### Task 6: Проверка

**Files:**
- Run: `npm test` в `web`
- Run: targeted Go tests for email package or backend package set

- [ ] Прогнать фронтенд-тесты по изменённым компонентам.
- [ ] Прогнать Go-тесты для письма-приглашения.
- [ ] Зафиксировать риски: отсутствие отдельного route для полного мастера создания цели, возможные остатки английского текста вне основных сценариев.
