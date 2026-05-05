-- =============================================================================
-- ProofForge Demo Seed
-- Загружает 3 пользователей, круг, 2 активные цели, чекины, рекапы.
-- Запуск:
--   docker compose -f infra/docker/compose.dev.yml exec -T postgres \
--     psql -U proofforge -d proofforge < infra/docker/seed.sql
-- =============================================================================

BEGIN;

-- ── 1. Users ─────────────────────────────────────────────────────────────────
INSERT INTO users (id, email, display_name, share_default, created_at, updated_at)
VALUES
  (1, 'alex@example.com',    'Алекс',   TRUE, NOW() - INTERVAL '30 days', NOW()),
  (2, 'marina@example.com',  'Марина',  TRUE, NOW() - INTERVAL '28 days', NOW()),
  (3, 'kirill@example.com',  'Кирилл', TRUE, NOW() - INTERVAL '14 days', NOW())
ON CONFLICT (email) DO UPDATE
  SET display_name  = EXCLUDED.display_name,
      share_default = EXCLUDED.share_default;

-- Сбросить последовательность после ручных id
SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));

-- ── 2. Sessions (чтобы можно было войти по email без реального SMTP) ──────────
-- Токен = sha256('demo-token-alex') итд — в dev можно использовать API напрямую
-- Сессии тут не нужны — вход через форму на /dashboard

-- ── 3. Circle ────────────────────────────────────────────────────────────────
INSERT INTO circles (id, owner_user_id, name, invite_code, member_limit, daily_window_tz, daily_cutoff, created_at, updated_at)
VALUES (
  1,
  1,
  'Утренние чемпионы',
  'DEMO2025',
  8,
  'Europe/Moscow',
  '23:59',
  NOW() - INTERVAL '21 days',
  NOW()
)
ON CONFLICT (invite_code) DO NOTHING;

SELECT setval('circles_id_seq', (SELECT MAX(id) FROM circles));

-- ── 4. Circle memberships ────────────────────────────────────────────────────
INSERT INTO circle_memberships (circle_id, user_id, status, role, joined_at, created_at)
VALUES
  (1, 1, 'active', 'owner',   NOW() - INTERVAL '21 days', NOW() - INTERVAL '21 days'),
  (1, 2, 'active', 'buddy',   NOW() - INTERVAL '20 days', NOW() - INTERVAL '20 days'),
  (1, 3, 'active', 'observer', NOW() - INTERVAL '14 days', NOW() - INTERVAL '14 days')
ON CONFLICT (circle_id, user_id) DO NOTHING;

-- ── 5. Circle season ─────────────────────────────────────────────────────────
INSERT INTO circle_seasons (id, circle_id, status, starts_at, ends_at, created_at)
VALUES (
  1,
  1,
  'active',
  NOW() - INTERVAL '4 days',
  NOW() + INTERVAL '3 days',
  NOW() - INTERVAL '4 days'
)
ON CONFLICT DO NOTHING;

SELECT setval('circle_seasons_id_seq', (SELECT MAX(id) FROM circle_seasons));

-- ── 6. Goals ─────────────────────────────────────────────────────────────────
-- Goal 1: Алекс хочет бегать каждое утро, бадди — Марина
INSERT INTO goals (
  id, circle_id, owner_user_id, buddy_user_id,
  title, description, status,
  current_progress_health, current_streak_count,
  proof_examples, category,
  is_public_template,
  created_at, updated_at
) VALUES (
  1, 1, 1, 2,
  'Бегать 5 км каждое утро',
  'Подъём в 6:30, пробежка в парке минимум 5 км. Доказательство — скриншот из Strava.',
  'active',
  'strong', 5,
  'Скриншот из Strava с дистанцией и временем',
  'Спорт и здоровье',
  TRUE,
  NOW() - INTERVAL '20 days', NOW()
) ON CONFLICT DO NOTHING;

-- Goal 2: Марина читает по 20 минут в день, бадди — Алекс
INSERT INTO goals (
  id, circle_id, owner_user_id, buddy_user_id,
  title, description, status,
  current_progress_health, current_streak_count,
  proof_examples, category,
  is_public_template,
  created_at, updated_at
) VALUES (
  2, 1, 2, 1,
  'Читать 20 минут каждый день',
  'Ежедневное чтение нон-фикшн книги. Не новости, не соцсети — только книги.',
  'active',
  'improving', 3,
  'Фото страницы с закладкой или заметка с цитатой дня',
  'Образование',
  TRUE,
  NOW() - INTERVAL '18 days', NOW()
) ON CONFLICT DO NOTHING;

-- Goal 3: Кирилл учит английский, бадди — Марина (pending — для демонстрации invite flow)
INSERT INTO goals (
  id, circle_id, owner_user_id, buddy_user_id,
  title, description, status,
  current_progress_health, current_streak_count,
  proof_examples, category,
  is_public_template,
  created_at, updated_at
) VALUES (
  3, 1, 3, 2,
  'Учить английский 30 минут в день',
  'Duolingo + чтение статей на английском. Цель — B2 к концу года.',
  'active',
  'unknown', 1,
  'Скриншот прогресса в Duolingo или прочитанная статья с кратким конспектом',
  'Образование',
  FALSE,
  NOW() - INTERVAL '13 days', NOW()
) ON CONFLICT DO NOTHING;

SELECT setval('goals_id_seq', (SELECT MAX(id) FROM goals));

-- ── 7. Pacts ─────────────────────────────────────────────────────────────────
INSERT INTO pacts (id, goal_id, owner_user_id, buddy_user_id, status, accepted_at, created_at, updated_at)
VALUES
  (1, 1, 1, 2, 'active', NOW() - INTERVAL '20 days', NOW() - INTERVAL '20 days', NOW()),
  (2, 2, 2, 1, 'active', NOW() - INTERVAL '18 days', NOW() - INTERVAL '18 days', NOW()),
  (3, 3, 3, 2, 'active', NOW() - INTERVAL '13 days', NOW() - INTERVAL '13 days', NOW())
ON CONFLICT DO NOTHING;

SELECT setval('pacts_id_seq', (SELECT MAX(id) FROM pacts));

-- ── 8. Check-ins (Goal 1 — Алекс бегает) ────────────────────────────────────
-- Одобренные чекины прошлых дней
INSERT INTO check_ins (id, goal_id, owner_user_id, status, is_public_example, submitted_at, approved_at, created_at, updated_at)
VALUES
  (1,  1, 1, 'approved', TRUE,  NOW()-INTERVAL '5 days 8h', NOW()-INTERVAL '5 days 10h', NOW()-INTERVAL '5 days', NOW()-INTERVAL '5 days 9h'),
  (2,  1, 1, 'approved', TRUE,  NOW()-INTERVAL '4 days 8h', NOW()-INTERVAL '4 days 11h', NOW()-INTERVAL '4 days', NOW()-INTERVAL '4 days 10h'),
  (3,  1, 1, 'approved', FALSE, NOW()-INTERVAL '3 days 9h', NOW()-INTERVAL '3 days 12h', NOW()-INTERVAL '3 days', NOW()-INTERVAL '3 days 11h'),
  (4,  1, 1, 'approved', TRUE,  NOW()-INTERVAL '2 days 8h', NOW()-INTERVAL '2 days 9h',  NOW()-INTERVAL '2 days', NOW()-INTERVAL '2 days 8h30m'),
  (5,  1, 1, 'approved', TRUE,  NOW()-INTERVAL '1 day 8h',  NOW()-INTERVAL '1 day 10h',  NOW()-INTERVAL '1 day',  NOW()-INTERVAL '1 day 9h'),
  -- Сегодня submitted — ждёт одобрения бадди
  (6,  1, 1, 'submitted', FALSE, NOW()-INTERVAL '2h', NULL, NOW()-INTERVAL '3h', NOW()-INTERVAL '2h')
ON CONFLICT DO NOTHING;

-- Check-ins Goal 2 — Марина читает
INSERT INTO check_ins (id, goal_id, owner_user_id, status, is_public_example, submitted_at, approved_at, created_at, updated_at)
VALUES
  (7,  2, 2, 'approved', TRUE,  NOW()-INTERVAL '3 days 20h', NOW()-INTERVAL '3 days 22h', NOW()-INTERVAL '3 days', NOW()-INTERVAL '3 days 21h'),
  (8,  2, 2, 'approved', TRUE,  NOW()-INTERVAL '2 days 21h', NOW()-INTERVAL '2 days 23h', NOW()-INTERVAL '2 days', NOW()-INTERVAL '2 days 22h'),
  (9,  2, 2, 'approved', FALSE, NOW()-INTERVAL '1 day 20h',  NOW()-INTERVAL '1 day 22h',  NOW()-INTERVAL '1 day',  NOW()-INTERVAL '1 day 21h'),
  -- Один rejected — для демонстрации flow
  (10, 2, 2, 'rejected', FALSE, NOW()-INTERVAL '4 days 20h', NULL, NOW()-INTERVAL '4 days', NOW()-INTERVAL '4 days 21h'),
  -- Draft — в процессе написания
  (11, 2, 2, 'draft', FALSE, NULL, NULL, NOW()-INTERVAL '1h', NOW()-INTERVAL '1h')
ON CONFLICT DO NOTHING;

-- Check-in Goal 3 — Кирилл (1 одобренный)
INSERT INTO check_ins (id, goal_id, owner_user_id, status, is_public_example, submitted_at, approved_at, created_at, updated_at)
VALUES
  (12, 3, 3, 'approved', FALSE, NOW()-INTERVAL '1 day 15h', NOW()-INTERVAL '1 day 16h', NOW()-INTERVAL '1 day', NOW()-INTERVAL '1 day 15h')
ON CONFLICT DO NOTHING;

SELECT setval('check_ins_id_seq', (SELECT MAX(id) FROM check_ins));

-- ── 9. Evidence items ────────────────────────────────────────────────────────
INSERT INTO evidence_items (check_in_id, kind, text_content, external_url, created_at)
VALUES
  -- Goal 1 чекины
  (1, 'text', '5.2 км за 28 минут. Новый личный рекорд по темпу!', NULL, NOW()-INTERVAL '5 days 8h'),
  (1, 'link', NULL, 'https://strava.com/activities/demo1', NOW()-INTERVAL '5 days 8h5m'),
  (2, 'text', '5.1 км, дождь, но добежал. Чувствую себя огнём 🔥', NULL, NOW()-INTERVAL '4 days 8h'),
  (3, 'text', '5.0 км ровно. Попробовал новый маршрут через набережную.', NULL, NOW()-INTERVAL '3 days 9h'),
  (4, 'text', '5.3 км — лучшее время за неделю. Тело привыкает к режиму.', NULL, NOW()-INTERVAL '2 days 8h'),
  (4, 'link', NULL, 'https://strava.com/activities/demo4', NOW()-INTERVAL '2 days 8h5m'),
  (5, 'text', '5.0 км за 27 минут 40 секунд. 5 дней подряд без пропусков!', NULL, NOW()-INTERVAL '1 day 8h'),
  (6, 'text', '5.1 км сегодня утром. Немного устал, но вышел.', NULL, NOW()-INTERVAL '2h'),
  -- Goal 2 чекины
  (7, 'text', 'Дочитал главу 4 книги «Атомные привычки». Заметка: маленькие изменения создают большие результаты.', NULL, NOW()-INTERVAL '3 days 20h'),
  (8, 'text', 'Глава 5-6. Интересная мысль про identity-based habits — меняй не действия, а то кем себя считаешь.', NULL, NOW()-INTERVAL '2 days 21h'),
  (9, 'text', 'Сегодня 25 минут, немного перебрала норму :) Дочитала до конца части 2.', NULL, NOW()-INTERVAL '1 day 20h'),
  (10, 'text', 'Читала, но только 10 минут — потом отвлеклась на сериал. Честно.', NULL, NOW()-INTERVAL '4 days 20h'),
  (12, 'text', 'Duolingo streak 1 день + прочитал статью на BBC Learning English про технологии.', NULL, NOW()-INTERVAL '1 day 15h')
ON CONFLICT DO NOTHING;

-- ── 10. Reviews ──────────────────────────────────────────────────────────────
INSERT INTO check_in_reviews (check_in_id, reviewer_user_id, decision, comment, created_at)
VALUES
  (1,  2, 'approved',  'Красавчик! 5 км это серьёзно 💪', NOW()-INTERVAL '5 days 10h'),
  (2,  2, 'approved',  'Дождь не помеха, уважаю', NOW()-INTERVAL '4 days 11h'),
  (3,  2, 'approved',  'Принято', NOW()-INTERVAL '3 days 12h'),
  (4,  2, 'approved',  'Рост темпа виден, так держать!', NOW()-INTERVAL '2 days 9h'),
  (5,  2, 'approved',  '5 из 5! Неделя без пропусков 🏆', NOW()-INTERVAL '1 day 10h'),
  (7,  1, 'approved',  'Атомные привычки — топ книга, сам читал', NOW()-INTERVAL '3 days 22h'),
  (8,  1, 'approved',  'Хорошая заметка, именно это и зацепило меня в книге', NOW()-INTERVAL '2 days 23h'),
  (9,  1, 'approved',  '25 минут — не нарушение, это бонус :)', NOW()-INTERVAL '1 day 22h'),
  (10, 1, 'rejected',  '10 минут не считается, сериал это не форс-мажор 😄', NOW()-INTERVAL '4 days 21h'),
  (12, 2, 'approved',  'Хорошее начало! BBC Learning — отличный ресурс', NOW()-INTERVAL '1 day 16h')
ON CONFLICT DO NOTHING;

-- ── 11. Circle events (для ленты Пульс) ──────────────────────────────────────
INSERT INTO circle_events (circle_id, kind, actor_user_id, payload, created_at)
VALUES
  (1, 'approved',  1, '{"streak": 5, "goal_title": "Бегать 5 км каждое утро"}',    NOW()-INTERVAL '1 day 10h'),
  (1, 'approved',  2, '{"streak": 3, "goal_title": "Читать 20 минут каждый день"}', NOW()-INTERVAL '1 day 22h'),
  (1, 'approved',  3, '{"streak": 1, "goal_title": "Учить английский 30 минут в день"}', NOW()-INTERVAL '1 day 16h'),
  (1, 'at_risk',   2, '{"reason": "rejected_checkin"}',                             NOW()-INTERVAL '4 days 21h'),
  (1, 'comeback',  2, '{"streak": 2}',                                              NOW()-INTERVAL '2 days 23h')
ON CONFLICT DO NOTHING;

-- ── 12. Weekly recaps ────────────────────────────────────────────────────────
INSERT INTO weekly_recaps (id, goal_id, owner_user_id, period_start, period_end, status, summary_text, model_name, generated_at, created_at)
VALUES (
  1, 1, 1,
  NOW()-INTERVAL '14 days',
  NOW()-INTERVAL '7 days',
  'ready',
  'Отличная неделя! Алекс выполнил все 7 чекинов подряд, средняя дистанция 5.1 км. Бадди Марина одобрила все сдачи с позитивными комментариями. Streak растёт — хороший темп для первой недели.',
  'demo',
  NOW()-INTERVAL '7 days',
  NOW()-INTERVAL '7 days'
),
(
  2, 2, 2,
  NOW()-INTERVAL '14 days',
  NOW()-INTERVAL '7 days',
  'ready',
  'Марина читала 5 из 7 дней. Один день был пропущен, один — отклонён бадди (только 10 минут вместо 20). «Атомные привычки» — хороший выбор для старта, заметки по книге показывают глубокое вовлечение.',
  'demo',
  NOW()-INTERVAL '7 days',
  NOW()-INTERVAL '7 days'
)
ON CONFLICT DO NOTHING;

SELECT setval('weekly_recaps_id_seq', (SELECT MAX(id) FROM weekly_recaps));

-- ── 13. Library templates (публичные шаблоны целей) ──────────────────────────
-- Обновляем уже вставленные цели + добавляем standalone шаблоны
UPDATE goals SET is_public_template = TRUE WHERE id IN (1, 2);

COMMIT;

-- Verify
SELECT 'users' AS tbl, COUNT(*) FROM users
UNION ALL SELECT 'circles',          COUNT(*) FROM circles
UNION ALL SELECT 'circle_memberships', COUNT(*) FROM circle_memberships
UNION ALL SELECT 'circle_seasons',   COUNT(*) FROM circle_seasons
UNION ALL SELECT 'goals',            COUNT(*) FROM goals
UNION ALL SELECT 'pacts',            COUNT(*) FROM pacts
UNION ALL SELECT 'check_ins',        COUNT(*) FROM check_ins
UNION ALL SELECT 'evidence_items',   COUNT(*) FROM evidence_items
UNION ALL SELECT 'check_in_reviews', COUNT(*) FROM check_in_reviews
UNION ALL SELECT 'circle_events',    COUNT(*) FROM circle_events
UNION ALL SELECT 'weekly_recaps',    COUNT(*) FROM weekly_recaps;
