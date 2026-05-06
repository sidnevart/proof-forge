-- =============================================================================
-- ProofForge Demo Seed v2
-- 3 пользователя, 2 круга, 3 цели, богатая лента, публичные доказательства
-- =============================================================================

BEGIN;

-- ── 1. Users ─────────────────────────────────────────────────────────────────
-- public_alias — отображается в публичной ленте
-- share_default TRUE — чекины видны кругу по умолчанию
-- is_anonymous_public FALSE — alias виден в публичной ленте
INSERT INTO users (id, email, display_name, public_alias, share_default, is_anonymous_public, created_at, updated_at)
VALUES
  (1, 'alex@example.com',   'Алекс',   'alex_runs',    TRUE,  FALSE, NOW()-INTERVAL '30 days', NOW()),
  (2, 'marina@example.com', 'Марина',  'marina_reads',  TRUE,  FALSE, NOW()-INTERVAL '28 days', NOW()),
  (3, 'kirill@example.com', 'Кирилл', 'kirill_learns', TRUE,  FALSE, NOW()-INTERVAL '14 days', NOW())
ON CONFLICT (email) DO UPDATE
  SET display_name        = EXCLUDED.display_name,
      public_alias        = EXCLUDED.public_alias,
      share_default       = EXCLUDED.share_default,
      is_anonymous_public = EXCLUDED.is_anonymous_public;

SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));

-- ── 2. Circles ───────────────────────────────────────────────────────────────
-- Круг 1: все три участника, активный сезон
INSERT INTO circles (id, owner_user_id, name, invite_code, member_limit, daily_window_tz, daily_cutoff, created_at, updated_at)
VALUES
  (1, 1, 'Утренние чемпионы', 'DEMO2025', 8, 'Europe/Moscow', '23:59', NOW()-INTERVAL '21 days', NOW()),
  (2, 2, 'Книжный клуб',      'BOOKS42',  6, 'Europe/Moscow', '22:00', NOW()-INTERVAL '10 days', NOW())
ON CONFLICT (invite_code) DO NOTHING;

SELECT setval('circles_id_seq', (SELECT MAX(id) FROM circles));

-- ── 3. Circle memberships ────────────────────────────────────────────────────
INSERT INTO circle_memberships (circle_id, user_id, status, role, joined_at, created_at)
VALUES
  (1, 1, 'active', 'owner',    NOW()-INTERVAL '21 days', NOW()-INTERVAL '21 days'),
  (1, 2, 'active', 'buddy',    NOW()-INTERVAL '20 days', NOW()-INTERVAL '20 days'),
  (1, 3, 'active', 'observer', NOW()-INTERVAL '14 days', NOW()-INTERVAL '14 days'),
  (2, 2, 'active', 'owner',    NOW()-INTERVAL '10 days', NOW()-INTERVAL '10 days'),
  (2, 3, 'active', 'buddy',    NOW()-INTERVAL '9 days',  NOW()-INTERVAL '9 days')
ON CONFLICT (circle_id, user_id) DO NOTHING;

-- ── 4. Circle seasons ────────────────────────────────────────────────────────
INSERT INTO circle_seasons (id, circle_id, status, starts_at, ends_at, created_at)
VALUES
  (1, 1, 'active',    NOW()-INTERVAL '4 days', NOW()+INTERVAL '3 days', NOW()-INTERVAL '4 days'),
  (2, 2, 'active',    NOW()-INTERVAL '3 days', NOW()+INTERVAL '4 days', NOW()-INTERVAL '3 days')
ON CONFLICT DO NOTHING;

SELECT setval('circle_seasons_id_seq', (SELECT MAX(id) FROM circle_seasons));

-- ── 5. Goals ─────────────────────────────────────────────────────────────────
INSERT INTO goals (id, circle_id, owner_user_id, buddy_user_id,
  title, description, status,
  current_progress_health, current_streak_count,
  proof_examples, category, is_public_template,
  created_at, updated_at)
VALUES
  -- Цель 1: Алекс бегает, бадди Марина, в круге 1
  (1, 1, 1, 2,
   'Бегать 5 км каждое утро',
   'Подъём в 6:30, пробежка в парке минимум 5 км. Нет дождя, нет оправданий.',
   'active', 'strong', 7,
   'Скриншот из Strava с дистанцией и временем',
   'Спорт и здоровье', TRUE,
   NOW()-INTERVAL '4 days', NOW()),

  -- Цель 2: Марина читает, бадди Алекс, в круге 1
  (2, 1, 2, 1,
   'Читать 20 минут каждый день',
   'Нон-фикшн книги. Не новости, не соцсети — только книги.',
   'active', 'improving', 4,
   'Фото страницы с закладкой или заметка с цитатой',
   'Образование', TRUE,
   NOW()-INTERVAL '3 days', NOW()),

  -- Цель 3: Кирилл учит английский, бадди Марина, в круге 2
  (3, 2, 3, 2,
   'Учить английский 30 минут в день',
   'Duolingo + чтение статей на английском. Цель — B2 к концу года.',
   'active', 'stable', 3,
   'Скриншот Duolingo или краткий конспект прочитанной статьи',
   'Образование', TRUE,
   NOW()-INTERVAL '3 days', NOW())
ON CONFLICT DO NOTHING;

SELECT setval('goals_id_seq', (SELECT MAX(id) FROM goals));

-- ── 6. Pacts ─────────────────────────────────────────────────────────────────
INSERT INTO pacts (id, goal_id, owner_user_id, buddy_user_id, status, accepted_at, created_at, updated_at)
VALUES
  (1, 1, 1, 2, 'active', NOW()-INTERVAL '4 days', NOW()-INTERVAL '4 days', NOW()),
  (2, 2, 2, 1, 'active', NOW()-INTERVAL '3 days', NOW()-INTERVAL '3 days', NOW()),
  (3, 3, 3, 2, 'active', NOW()-INTERVAL '3 days', NOW()-INTERVAL '3 days', NOW())
ON CONFLICT DO NOTHING;

SELECT setval('pacts_id_seq', (SELECT MAX(id) FROM pacts));

-- ── 7. Invites (нужны для GoalView JOIN в dashboard) ────────────────────────
-- token_hash — sha256 от фиктивного токена, статус accepted
INSERT INTO invites (id, goal_id, pact_id, inviter_user_id, invitee_user_id, token_hash, status, expires_at, accepted_at, created_at)
VALUES
  (1, 1, 1, 1, 2, md5('demo-invite-token-1'), 'accepted', NOW()+INTERVAL '30 days', NOW()-INTERVAL '4 days', NOW()-INTERVAL '4 days'),
  (2, 2, 2, 2, 1, md5('demo-invite-token-2'), 'accepted', NOW()+INTERVAL '30 days', NOW()-INTERVAL '3 days', NOW()-INTERVAL '3 days'),
  (3, 3, 3, 3, 2, md5('demo-invite-token-3'), 'accepted', NOW()+INTERVAL '30 days', NOW()-INTERVAL '3 days', NOW()-INTERVAL '3 days')
ON CONFLICT DO NOTHING;

SELECT setval('invites_id_seq', (SELECT MAX(id) FROM invites));

-- ── 8. Check-ins ─────────────────────────────────────────────────────────────
-- is_public_example = TRUE → попадает в публичную ленту "ПОХОЖИЕ ЦЕЛИ"
-- status submitted/approved → попадает в ленту круга "МОЙ КРУГ"

-- Goal 1 — Алекс бегает (4 approved + 1 submitted сегодня)
INSERT INTO check_ins (id, goal_id, owner_user_id, status, is_public_example, submitted_at, approved_at, created_at, updated_at)
VALUES
  (1,  1, 1, 'approved', TRUE,  NOW()-INTERVAL '4 days 8h',  NOW()-INTERVAL '4 days 10h', NOW()-INTERVAL '4 days',  NOW()-INTERVAL '4 days 9h'),
  (2,  1, 1, 'approved', TRUE,  NOW()-INTERVAL '3 days 8h',  NOW()-INTERVAL '3 days 11h', NOW()-INTERVAL '3 days',  NOW()-INTERVAL '3 days 10h'),
  (3,  1, 1, 'approved', TRUE,  NOW()-INTERVAL '2 days 8h',  NOW()-INTERVAL '2 days 9h',  NOW()-INTERVAL '2 days',  NOW()-INTERVAL '2 days 8h30m'),
  (4,  1, 1, 'approved', TRUE,  NOW()-INTERVAL '1 day 8h',   NOW()-INTERVAL '1 day 10h',  NOW()-INTERVAL '1 day',   NOW()-INTERVAL '1 day 9h'),
  -- сегодня: submitted, ждёт одобрения Марины
  (5,  1, 1, 'submitted', FALSE, NOW()-INTERVAL '1h',         NULL,                         NOW()-INTERVAL '2h',   NOW()-INTERVAL '1h')
ON CONFLICT DO NOTHING;

-- Goal 2 — Марина читает (3 approved + 1 submitted)
INSERT INTO check_ins (id, goal_id, owner_user_id, status, is_public_example, submitted_at, approved_at, created_at, updated_at)
VALUES
  (6,  2, 2, 'approved', TRUE,  NOW()-INTERVAL '3 days 21h', NOW()-INTERVAL '3 days 23h', NOW()-INTERVAL '3 days', NOW()-INTERVAL '3 days 22h'),
  (7,  2, 2, 'approved', TRUE,  NOW()-INTERVAL '2 days 21h', NOW()-INTERVAL '2 days 22h', NOW()-INTERVAL '2 days', NOW()-INTERVAL '2 days 21h30m'),
  (8,  2, 2, 'approved', TRUE,  NOW()-INTERVAL '1 day 20h',  NOW()-INTERVAL '1 day 22h',  NOW()-INTERVAL '1 day',  NOW()-INTERVAL '1 day 21h'),
  -- сегодня: submitted, ждёт одобрения Алекса
  (9,  2, 2, 'submitted', FALSE, NOW()-INTERVAL '30m',        NULL,                         NOW()-INTERVAL '1h',  NOW()-INTERVAL '30m')
ON CONFLICT DO NOTHING;

-- Goal 3 — Кирилл английский (3 approved + 1 submitted)
INSERT INTO check_ins (id, goal_id, owner_user_id, status, is_public_example, submitted_at, approved_at, created_at, updated_at)
VALUES
  (10, 3, 3, 'approved', TRUE,  NOW()-INTERVAL '3 days 15h', NOW()-INTERVAL '3 days 16h', NOW()-INTERVAL '3 days', NOW()-INTERVAL '3 days 15h'),
  (11, 3, 3, 'approved', TRUE,  NOW()-INTERVAL '2 days 15h', NOW()-INTERVAL '2 days 16h', NOW()-INTERVAL '2 days', NOW()-INTERVAL '2 days 15h'),
  (12, 3, 3, 'approved', TRUE,  NOW()-INTERVAL '1 day 15h',  NOW()-INTERVAL '1 day 16h',  NOW()-INTERVAL '1 day',  NOW()-INTERVAL '1 day 15h'),
  -- сегодня: submitted, ждёт одобрения Марины
  (13, 3, 3, 'submitted', FALSE, NOW()-INTERVAL '45m',        NULL,                         NOW()-INTERVAL '2h',  NOW()-INTERVAL '45m')
ON CONFLICT DO NOTHING;

SELECT setval('check_ins_id_seq', (SELECT MAX(id) FROM check_ins));

-- ── 8. Evidence items ────────────────────────────────────────────────────────
INSERT INTO evidence_items (check_in_id, kind, text_content, external_url, created_at)
VALUES
  -- Goal 1: Алекс бегает (check-ins 1-5)
  (1,  'text', '5.2 км за 28 минут. Новый личный рекорд по темпу!', NULL, NOW()-INTERVAL '4 days 8h'),
  (1,  'link', NULL, 'https://strava.com/activities/demo1', NOW()-INTERVAL '4 days 8h5m'),
  (2,  'text', '5.1 км — дождь, но я добежал. Чувствую себя чемпионом 🔥', NULL, NOW()-INTERVAL '3 days 8h'),
  (3,  'text', '5.3 км — лучшее время за неделю. Тело привыкает к режиму.', NULL, NOW()-INTERVAL '2 days 8h'),
  (3,  'link', NULL, 'https://strava.com/activities/demo3', NOW()-INTERVAL '2 days 8h5m'),
  (4,  'text', '5.1 км за 27 минут 40 секунд. 4 дня подряд — горжусь собой 💪', NULL, NOW()-INTERVAL '1 day 8h'),
  (5,  'text', '5.0 км сегодня утром. Устал после вчерашнего, но вышел как обещал.', NULL, NOW()-INTERVAL '1h'),

  -- Goal 2: Марина читает (check-ins 6-9)
  (6,  'text', 'Начала «Атомные привычки». Глава 1-3: маленькие изменения = большие результаты.', NULL, NOW()-INTERVAL '3 days 21h'),
  (7,  'text', 'Глава 4-5. Про identity-based habits — меняй не действия, а то кем себя считаешь. 🤯', NULL, NOW()-INTERVAL '2 days 21h'),
  (8,  'text', 'Дочитала первую часть. Самая важная цитата: "You do not rise to the level of your goals, you fall to the level of your systems."', NULL, NOW()-INTERVAL '1 day 20h'),
  (9,  'text', 'Начала вторую часть. Привычки-маяки работают лучше всего утром.', NULL, NOW()-INTERVAL '30m'),

  -- Goal 3: Кирилл английский (check-ins 10-13)
  (10, 'text', 'Day 1: Duolingo streak + статья на BBC Learning English про искусственный интеллект. Новые слова: "breakthrough", "pivotal", "unprecedented".', NULL, NOW()-INTERVAL '3 days 15h'),
  (11, 'text', 'Day 2: Посмотрел TED talk с субтитрами, записал 15 новых выражений. Уровень слушания растёт.', NULL, NOW()-INTERVAL '2 days 15h'),
  (12, 'text', 'Day 3: Прочитал статью The Guardian, написал краткое резюме на английском. Грамматические ошибки — всё меньше.', NULL, NOW()-INTERVAL '1 day 15h'),
  (13, 'text', 'Сегодня 35 минут: Duolingo + разговорный урок с носителем в italki. Первый раз не стеснялся говорить!', NULL, NOW()-INTERVAL '45m')
ON CONFLICT DO NOTHING;

-- ── 9. Обновляем public_attachment_ids для публичных чекинов ─────────────────
-- Чтобы публичная лента показывала evidence
UPDATE check_ins SET public_attachment_ids = ARRAY(
  SELECT id FROM evidence_items WHERE check_in_id = 1 AND kind = 'text' LIMIT 1
) WHERE id = 1 AND is_public_example = TRUE;

UPDATE check_ins SET public_attachment_ids = ARRAY(
  SELECT id FROM evidence_items WHERE check_in_id = 2 LIMIT 1
) WHERE id = 2 AND is_public_example = TRUE;

UPDATE check_ins SET public_attachment_ids = ARRAY(
  SELECT id FROM evidence_items WHERE check_in_id = 3 AND kind = 'text' LIMIT 1
) WHERE id = 3 AND is_public_example = TRUE;

UPDATE check_ins SET public_attachment_ids = ARRAY(
  SELECT id FROM evidence_items WHERE check_in_id = 4 LIMIT 1
) WHERE id = 4 AND is_public_example = TRUE;

UPDATE check_ins SET public_attachment_ids = ARRAY(
  SELECT id FROM evidence_items WHERE check_in_id = 6 LIMIT 1
) WHERE id = 6 AND is_public_example = TRUE;

UPDATE check_ins SET public_attachment_ids = ARRAY(
  SELECT id FROM evidence_items WHERE check_in_id = 7 LIMIT 1
) WHERE id = 7 AND is_public_example = TRUE;

UPDATE check_ins SET public_attachment_ids = ARRAY(
  SELECT id FROM evidence_items WHERE check_in_id = 8 LIMIT 1
) WHERE id = 8 AND is_public_example = TRUE;

UPDATE check_ins SET public_attachment_ids = ARRAY(
  SELECT id FROM evidence_items WHERE check_in_id = 10 LIMIT 1
) WHERE id = 10 AND is_public_example = TRUE;

UPDATE check_ins SET public_attachment_ids = ARRAY(
  SELECT id FROM evidence_items WHERE check_in_id = 11 LIMIT 1
) WHERE id = 11 AND is_public_example = TRUE;

UPDATE check_ins SET public_attachment_ids = ARRAY(
  SELECT id FROM evidence_items WHERE check_in_id = 12 LIMIT 1
) WHERE id = 12 AND is_public_example = TRUE;

-- ── 10. Reviews ──────────────────────────────────────────────────────────────
INSERT INTO check_in_reviews (check_in_id, reviewer_user_id, decision, comment, created_at)
VALUES
  -- Марина одобряет Алекса (check-ins 1-4)
  (1,  2, 'approved', 'Красавчик! 5 км это серьёзно 💪',        NOW()-INTERVAL '4 days 10h'),
  (2,  2, 'approved', 'Дождь не помеха, уважаю!',              NOW()-INTERVAL '3 days 11h'),
  (3,  2, 'approved', 'Рост темпа виден, так держать 🚀',      NOW()-INTERVAL '2 days 9h'),
  (4,  2, 'approved', '4 дня подряд — это уже привычка! 🏆',   NOW()-INTERVAL '1 day 10h'),
  -- Алекс одобряет Марину (check-ins 6-8)
  (6,  1, 'approved', 'Атомные привычки — топ книга, сам читал', NOW()-INTERVAL '3 days 23h'),
  (7,  1, 'approved', 'Identity-based habits — про изменение личности, сильная идея', NOW()-INTERVAL '2 days 22h'),
  (8,  1, 'approved', 'Отличная цитата! Системы важнее целей.', NOW()-INTERVAL '1 day 22h'),
  -- Марина одобряет Кирилла (check-ins 10-12)
  (10, 2, 'approved', 'Хорошее начало! BBC Learning — отличный ресурс', NOW()-INTERVAL '3 days 16h'),
  (11, 2, 'approved', 'TED talks с субтитрами — умно, я так и делала', NOW()-INTERVAL '2 days 16h'),
  (12, 2, 'approved', 'Резюме на английском — это уже уровень 🎯', NOW()-INTERVAL '1 day 16h')
ON CONFLICT DO NOTHING;

-- ── 11. Circle events ────────────────────────────────────────────────────────
INSERT INTO circle_events (circle_id, kind, actor_user_id, payload, created_at)
VALUES
  (1, 'approved', 1, '{"streak":4,"goal_title":"Бегать 5 км каждое утро"}',       NOW()-INTERVAL '1 day 10h'),
  (1, 'approved', 2, '{"streak":3,"goal_title":"Читать 20 минут каждый день"}',    NOW()-INTERVAL '1 day 22h'),
  (2, 'approved', 3, '{"streak":3,"goal_title":"Учить английский 30 минут в день"}', NOW()-INTERVAL '1 day 16h')
ON CONFLICT DO NOTHING;

-- ── 12. Weekly recaps ────────────────────────────────────────────────────────
INSERT INTO weekly_recaps (id, goal_id, owner_user_id, period_start, period_end, status, summary_text, model_name, generated_at, created_at)
VALUES
(1, 1, 1,
  NOW()-INTERVAL '14 days', NOW()-INTERVAL '7 days', 'ready',
  'Отличная первая неделя! Алекс выполнил все 7 чекинов без пропусков, средняя дистанция 5.1 км, лучшее время — 27:40. Марина одобрила каждый с позитивным фидбэком. Streak набирает силу.',
  'demo', NOW()-INTERVAL '7 days', NOW()-INTERVAL '7 days'),
(2, 2, 2,
  NOW()-INTERVAL '14 days', NOW()-INTERVAL '7 days', 'ready',
  'Марина читала 5 из 7 дней. Один день пропущен, один отклонён (10 минут вместо 20). «Атомные привычки» — сильный выбор для старта. Заметки по книге показывают вовлечённость.',
  'demo', NOW()-INTERVAL '7 days', NOW()-INTERVAL '7 days'),
(3, 3, 3,
  NOW()-INTERVAL '10 days', NOW()-INTERVAL '3 days', 'ready',
  'Кирилл стартовал уверенно: Duolingo streak 3 дня, плюс активная работа с оригинальными текстами. Итальянский разговорный урок — хороший признак готовности выйти за пределы зоны комфорта.',
  'demo', NOW()-INTERVAL '3 days', NOW()-INTERVAL '3 days')
ON CONFLICT DO NOTHING;

SELECT setval('weekly_recaps_id_seq', (SELECT MAX(id) FROM weekly_recaps));

-- ── 13. Обновить streaks и health у целей ────────────────────────────────────
UPDATE goals SET current_streak_count = 4, current_progress_health = 'strong'    WHERE id = 1;
UPDATE goals SET current_streak_count = 3, current_progress_health = 'improving' WHERE id = 2;
UPDATE goals SET current_streak_count = 3, current_progress_health = 'stable'    WHERE id = 3;

COMMIT;

-- ── Verify ───────────────────────────────────────────────────────────────────
SELECT tbl, cnt FROM (
  SELECT 'users'               AS tbl, COUNT(*)::int AS cnt FROM users
  UNION ALL SELECT 'circles',               COUNT(*) FROM circles
  UNION ALL SELECT 'circle_memberships',    COUNT(*) FROM circle_memberships
  UNION ALL SELECT 'circle_seasons',        COUNT(*) FROM circle_seasons
  UNION ALL SELECT 'goals',                 COUNT(*) FROM goals
  UNION ALL SELECT 'pacts',                 COUNT(*) FROM pacts
  UNION ALL SELECT 'check_ins',             COUNT(*) FROM check_ins
  UNION ALL SELECT 'evidence_items',        COUNT(*) FROM evidence_items
  UNION ALL SELECT 'check_in_reviews',      COUNT(*) FROM check_in_reviews
  UNION ALL SELECT 'circle_events',         COUNT(*) FROM circle_events
  UNION ALL SELECT 'weekly_recaps',         COUNT(*) FROM weekly_recaps
) t ORDER BY tbl;
