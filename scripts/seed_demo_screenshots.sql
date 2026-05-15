-- Demo data for screenshot sessions.
-- Idempotent: removes and recreates only records tied to the demo emails/codes below.

BEGIN;

DO $$
DECLARE
  demo_user_id BIGINT;
  buddy_id BIGINT;
  observer_id BIGINT;
  circle_id BIGINT;
  launch_circle_id BIGINT;
  team_id BIGINT;
  goal_pitch_id BIGINT;
  goal_launch_id BIGINT;
  goal_training_id BIGINT;
  pact_id BIGINT;
  ci_id BIGINT;
  evidence_id BIGINT;
BEGIN
  SELECT id INTO demo_user_id FROM users WHERE email = 'demo@proof-forge.local';
  SELECT id INTO buddy_id FROM users WHERE email = 'buddy@proof-forge.local';
  SELECT id INTO observer_id FROM users WHERE email = 'observer@proof-forge.local';

  DELETE FROM ai_notifications
  WHERE user_id IN (
    SELECT id FROM users
    WHERE email IN ('demo@proof-forge.local', 'buddy@proof-forge.local', 'observer@proof-forge.local')
  );

  DELETE FROM ai_proof_drafts
  WHERE user_id IN (
    SELECT id FROM users
    WHERE email IN ('demo@proof-forge.local', 'buddy@proof-forge.local', 'observer@proof-forge.local')
  );

  DELETE FROM teams WHERE invite_code = 'PFDEMOAI';

  DELETE FROM goals
  WHERE owner_user_id IN (
    SELECT id FROM users
    WHERE email IN ('demo@proof-forge.local', 'buddy@proof-forge.local', 'observer@proof-forge.local')
  )
  OR title IN (
    'Собрать публичный investor update',
    'Запустить страницу предзаказа',
    'Вернуть силовые тренировки'
  );

  DELETE FROM circles WHERE invite_code IN ('PFDEMO', 'PFDEMO2');

  INSERT INTO users (email, display_name, public_alias, share_default, is_anonymous_public, created_at, updated_at)
  VALUES ('demo@proof-forge.local', 'Демо Основатель', 'demo_founder', TRUE, FALSE, NOW() - INTERVAL '45 days', NOW())
  ON CONFLICT (email) DO UPDATE
  SET display_name = EXCLUDED.display_name,
      public_alias = EXCLUDED.public_alias,
      share_default = EXCLUDED.share_default,
      is_anonymous_public = EXCLUDED.is_anonymous_public,
      updated_at = NOW()
  RETURNING id INTO demo_user_id;

  INSERT INTO users (email, display_name, public_alias, share_default, is_anonymous_public, created_at, updated_at)
  VALUES ('buddy@proof-forge.local', 'Марина Ревьюер', 'marina_seal', TRUE, FALSE, NOW() - INTERVAL '43 days', NOW())
  ON CONFLICT (email) DO UPDATE
  SET display_name = EXCLUDED.display_name,
      public_alias = EXCLUDED.public_alias,
      share_default = EXCLUDED.share_default,
      is_anonymous_public = EXCLUDED.is_anonymous_public,
      updated_at = NOW()
  RETURNING id INTO buddy_id;

  INSERT INTO users (email, display_name, public_alias, share_default, is_anonymous_public, created_at, updated_at)
  VALUES ('observer@proof-forge.local', 'Илья Спринтер', 'ilya_builds', TRUE, FALSE, NOW() - INTERVAL '28 days', NOW())
  ON CONFLICT (email) DO UPDATE
  SET display_name = EXCLUDED.display_name,
      public_alias = EXCLUDED.public_alias,
      share_default = EXCLUDED.share_default,
      is_anonymous_public = EXCLUDED.is_anonymous_public,
      updated_at = NOW()
  RETURNING id INTO observer_id;

  INSERT INTO circles (owner_user_id, name, invite_code, member_limit, daily_window_tz, daily_cutoff, created_at, updated_at)
  VALUES (demo_user_id, 'Публичный запуск', 'PFDEMO', 8, 'Europe/Moscow', '22:30', NOW() - INTERVAL '21 days', NOW())
  ON CONFLICT (invite_code) DO UPDATE
  SET owner_user_id = EXCLUDED.owner_user_id,
      name = EXCLUDED.name,
      member_limit = EXCLUDED.member_limit,
      daily_window_tz = EXCLUDED.daily_window_tz,
      daily_cutoff = EXCLUDED.daily_cutoff,
      updated_at = NOW()
  RETURNING id INTO circle_id;

  INSERT INTO circles (owner_user_id, name, invite_code, member_limit, daily_window_tz, daily_cutoff, created_at, updated_at)
  VALUES (demo_user_id, 'Предзаказ без самообмана', 'PFDEMO2', 8, 'Europe/Moscow', '22:30', NOW() - INTERVAL '12 days', NOW())
  ON CONFLICT (invite_code) DO UPDATE
  SET owner_user_id = EXCLUDED.owner_user_id,
      name = EXCLUDED.name,
      member_limit = EXCLUDED.member_limit,
      daily_window_tz = EXCLUDED.daily_window_tz,
      daily_cutoff = EXCLUDED.daily_cutoff,
      updated_at = NOW()
  RETURNING id INTO launch_circle_id;

  INSERT INTO circle_memberships (circle_id, user_id, status, role, joined_at, created_at)
  VALUES
    (circle_id, demo_user_id, 'active', 'owner', NOW() - INTERVAL '21 days', NOW() - INTERVAL '21 days'),
    (circle_id, buddy_id, 'active', 'buddy', NOW() - INTERVAL '20 days', NOW() - INTERVAL '20 days'),
    (circle_id, observer_id, 'active', 'observer', NOW() - INTERVAL '12 days', NOW() - INTERVAL '12 days'),
    (launch_circle_id, demo_user_id, 'active', 'owner', NOW() - INTERVAL '12 days', NOW() - INTERVAL '12 days'),
    (launch_circle_id, buddy_id, 'active', 'buddy', NOW() - INTERVAL '11 days', NOW() - INTERVAL '11 days')
  ON CONFLICT ON CONSTRAINT circle_memberships_circle_id_user_id_key DO UPDATE
  SET status = EXCLUDED.status,
      role = EXCLUDED.role,
      joined_at = EXCLUDED.joined_at;

  INSERT INTO circle_seasons (circle_id, status, starts_at, ends_at, created_at)
  VALUES (circle_id, 'active', NOW() - INTERVAL '90 days', NOW() + INTERVAL '365 days', NOW() - INTERVAL '90 days');

  INSERT INTO circle_seasons (circle_id, status, starts_at, ends_at, created_at)
  VALUES (launch_circle_id, 'active', NOW() - INTERVAL '90 days', NOW() + INTERVAL '365 days', NOW() - INTERVAL '90 days');

  INSERT INTO teams (lead_user_id, name, invite_code, member_limit, ai_mode, created_at, updated_at)
  VALUES (demo_user_id, 'ProofForge demo lab', 'PFDEMOAI', 12, 'metadata-only', NOW() - INTERVAL '21 days', NOW())
  ON CONFLICT (invite_code) DO UPDATE
  SET lead_user_id = EXCLUDED.lead_user_id,
      name = EXCLUDED.name,
      ai_mode = EXCLUDED.ai_mode,
      updated_at = NOW()
  RETURNING id INTO team_id;

  INSERT INTO team_memberships (team_id, user_id, role, status, ai_consent, timezone, joined_at)
  VALUES
    (team_id, demo_user_id, 'lead', 'active', TRUE, 'Europe/Moscow', NOW() - INTERVAL '21 days'),
    (team_id, buddy_id, 'trusted_approver', 'active', TRUE, 'Europe/Moscow', NOW() - INTERVAL '20 days'),
    (team_id, observer_id, 'member', 'active', TRUE, 'Europe/Moscow', NOW() - INTERVAL '12 days')
  ON CONFLICT ON CONSTRAINT team_memberships_team_id_user_id_key DO UPDATE
  SET role = EXCLUDED.role,
      status = EXCLUDED.status,
      ai_consent = EXCLUDED.ai_consent,
      timezone = EXCLUDED.timezone;

  INSERT INTO goals (
    circle_id, owner_user_id, buddy_user_id, title, description, status,
    current_progress_health, current_streak_count, proof_examples, category,
    is_public_template, movement_mode, rhythm_cadence, created_at, updated_at
  )
  VALUES (
    launch_circle_id, demo_user_id, buddy_id,
    'Собрать публичный investor update',
    'Каждый рабочий день превращать прогресс в видимый артефакт: цифры, скрины, ссылки, решения.',
    'active', 'stable', 9,
    'Ссылка на опубликованный update, скрин метрик или changelog с принятым решением.',
    'startup', TRUE, 'free_goal', NULL, NOW() - INTERVAL '18 days', NOW()
  )
  RETURNING id INTO goal_pitch_id;

  INSERT INTO goals (
    circle_id, owner_user_id, buddy_user_id, title, description, status,
    current_progress_health, current_streak_count, proof_examples, category,
    is_public_template, movement_mode, rhythm_cadence, created_at, updated_at
  )
  VALUES (
    circle_id, demo_user_id, buddy_id,
    'Запустить страницу предзаказа',
    'Двигать landing, оффер и список ожидания до первого реального платежа.',
    'active', 'stable', 6,
    'Скрин страницы, список изменений, новый signup или запись разговора с пользователем.',
    'product', TRUE, 'free_goal', NULL, NOW() - INTERVAL '11 days', NOW()
  )
  RETURNING id INTO goal_launch_id;

  INSERT INTO goals (
    circle_id, owner_user_id, buddy_user_id, title, description, status,
    current_progress_health, current_streak_count, proof_examples, category,
    is_public_template, movement_mode, rhythm_cadence, created_at, updated_at
  )
  VALUES (
    circle_id, observer_id, demo_user_id,
    'Вернуть силовые тренировки',
    'Три тренировки в неделю с фото журнала подходов и коротким самочувствием после.',
    'active', 'at_risk', 3,
    'Фото тренировочного журнала или скрин из Strong с весами и подходами.',
    'health', TRUE, 'free_goal', NULL, NOW() - INTERVAL '9 days', NOW()
  )
  RETURNING id INTO goal_training_id;

  INSERT INTO pacts (goal_id, owner_user_id, buddy_user_id, status, accepted_at, created_at, updated_at)
  VALUES (goal_pitch_id, demo_user_id, buddy_id, 'active', NOW() - INTERVAL '18 days', NOW() - INTERVAL '18 days', NOW())
  RETURNING id INTO pact_id;
  INSERT INTO invites (goal_id, pact_id, inviter_user_id, invitee_user_id, token_hash, status, expires_at, accepted_at, created_at)
  VALUES (goal_pitch_id, pact_id, demo_user_id, buddy_id, md5('pf-demo-investor-update'), 'accepted', NOW() + INTERVAL '30 days', NOW() - INTERVAL '18 days', NOW() - INTERVAL '18 days');

  INSERT INTO pacts (goal_id, owner_user_id, buddy_user_id, status, accepted_at, created_at, updated_at)
  VALUES (goal_launch_id, demo_user_id, buddy_id, 'active', NOW() - INTERVAL '11 days', NOW() - INTERVAL '11 days', NOW())
  RETURNING id INTO pact_id;
  INSERT INTO invites (goal_id, pact_id, inviter_user_id, invitee_user_id, token_hash, status, expires_at, accepted_at, created_at)
  VALUES (goal_launch_id, pact_id, demo_user_id, buddy_id, md5('pf-demo-preorder-page'), 'accepted', NOW() + INTERVAL '30 days', NOW() - INTERVAL '11 days', NOW() - INTERVAL '11 days');

  INSERT INTO pacts (goal_id, owner_user_id, buddy_user_id, status, accepted_at, created_at, updated_at)
  VALUES (goal_training_id, observer_id, demo_user_id, 'active', NOW() - INTERVAL '9 days', NOW() - INTERVAL '9 days', NOW())
  RETURNING id INTO pact_id;
  INSERT INTO invites (goal_id, pact_id, inviter_user_id, invitee_user_id, token_hash, status, expires_at, accepted_at, created_at)
  VALUES (goal_training_id, pact_id, observer_id, demo_user_id, md5('pf-demo-training'), 'accepted', NOW() + INTERVAL '30 days', NOW() - INTERVAL '9 days', NOW() - INTERVAL '9 days');

  -- Investor update proofs.
  INSERT INTO check_ins (goal_id, owner_user_id, status, is_public_example, submitted_at, approved_at, created_at, updated_at)
  VALUES (goal_pitch_id, demo_user_id, 'approved', TRUE, NOW() - INTERVAL '4 days 4 hours', NOW() - INTERVAL '4 days 3 hours', NOW() - INTERVAL '4 days 4 hours', NOW() - INTERVAL '4 days 3 hours')
  RETURNING id INTO ci_id;
  INSERT INTO evidence_items (check_in_id, kind, text_content, external_url, created_at)
  VALUES (ci_id, 'text', 'Опубликовал update: 37 waitlist-заявок, 4 демо-звонка, 2 новых интро к потенциальным angel-инвесторам.', 'https://example.com/demo/investor-update-01', NOW() - INTERVAL '4 days 4 hours')
  RETURNING id INTO evidence_id;
  UPDATE check_ins SET public_attachment_ids = ARRAY[evidence_id] WHERE id = ci_id;
  INSERT INTO check_in_reviews (check_in_id, reviewer_user_id, decision, comment, created_at)
  VALUES (ci_id, buddy_id, 'approved', 'Цифры есть, выводы есть, следующий шаг понятен.', NOW() - INTERVAL '4 days 3 hours');

  INSERT INTO check_ins (goal_id, owner_user_id, status, is_public_example, submitted_at, approved_at, created_at, updated_at)
  VALUES (goal_pitch_id, demo_user_id, 'approved', TRUE, NOW() - INTERVAL '3 days 5 hours', NOW() - INTERVAL '3 days 4 hours', NOW() - INTERVAL '3 days 5 hours', NOW() - INTERVAL '3 days 4 hours')
  RETURNING id INTO ci_id;
  INSERT INTO evidence_items (check_in_id, kind, text_content, created_at)
  VALUES (ci_id, 'text', 'Собрал таблицу objections из 6 разговоров. Главный паттерн: людям нужен не трекер, а внешний контур ответственности.', NOW() - INTERVAL '3 days 5 hours')
  RETURNING id INTO evidence_id;
  UPDATE check_ins SET public_attachment_ids = ARRAY[evidence_id] WHERE id = ci_id;
  INSERT INTO check_in_reviews (check_in_id, reviewer_user_id, decision, comment, created_at)
  VALUES (ci_id, buddy_id, 'approved', 'Это уже usable insight, не просто заметки.', NOW() - INTERVAL '3 days 4 hours');

  INSERT INTO check_ins (goal_id, owner_user_id, status, is_public_example, submitted_at, approved_at, created_at, updated_at)
  VALUES (goal_pitch_id, demo_user_id, 'approved', TRUE, NOW() - INTERVAL '2 days 6 hours', NOW() - INTERVAL '2 days 5 hours', NOW() - INTERVAL '2 days 6 hours', NOW() - INTERVAL '2 days 5 hours')
  RETURNING id INTO ci_id;
  INSERT INTO evidence_items (check_in_id, kind, text_content, created_at)
  VALUES (ci_id, 'text', 'Переписал позиционирование: “proof-backed accountability для серьезных личных целей”. Убрал wellness-тон.', NOW() - INTERVAL '2 days 6 hours')
  RETURNING id INTO evidence_id;
  UPDATE check_ins SET public_attachment_ids = ARRAY[evidence_id] WHERE id = ci_id;
  INSERT INTO check_in_reviews (check_in_id, reviewer_user_id, decision, comment, created_at)
  VALUES (ci_id, buddy_id, 'approved', 'Формулировка стала жестче и точнее.', NOW() - INTERVAL '2 days 5 hours');

  INSERT INTO check_ins (goal_id, owner_user_id, status, is_public_example, submitted_at, approved_at, created_at, updated_at)
  VALUES (goal_pitch_id, demo_user_id, 'approved', TRUE, NOW() - INTERVAL '1 day 7 hours', NOW() - INTERVAL '1 day 6 hours', NOW() - INTERVAL '1 day 7 hours', NOW() - INTERVAL '1 day 6 hours')
  RETURNING id INTO ci_id;
  INSERT INTO evidence_items (check_in_id, kind, text_content, external_url, created_at)
  VALUES (ci_id, 'text', 'Сделал черновик pitch deck: problem, loop, wedge, proof examples, demo screenshots. Осталось вычистить метрики.', 'https://example.com/demo/pitch-v3', NOW() - INTERVAL '1 day 7 hours')
  RETURNING id INTO evidence_id;
  UPDATE check_ins SET public_attachment_ids = ARRAY[evidence_id] WHERE id = ci_id;
  INSERT INTO check_in_reviews (check_in_id, reviewer_user_id, decision, comment, created_at)
  VALUES (ci_id, buddy_id, 'approved', 'Deck уже можно показывать на warm intro.', NOW() - INTERVAL '1 day 6 hours');

  INSERT INTO check_ins (goal_id, owner_user_id, status, is_public_example, submitted_at, created_at, updated_at)
  VALUES (goal_pitch_id, demo_user_id, 'submitted', FALSE, NOW() - INTERVAL '42 minutes', NOW() - INTERVAL '52 minutes', NOW() - INTERVAL '42 minutes')
  RETURNING id INTO ci_id;
  INSERT INTO evidence_items (check_in_id, kind, text_content, created_at)
  VALUES (ci_id, 'text', 'Сегодня: собрал финальный screenshot pack для демо и отметил 5 кадров, которые лучше всего объясняют loop.', NOW() - INTERVAL '42 minutes');

  -- Preorder page proofs.
  INSERT INTO check_ins (goal_id, owner_user_id, status, is_public_example, submitted_at, approved_at, created_at, updated_at)
  VALUES (goal_launch_id, demo_user_id, 'approved', TRUE, NOW() - INTERVAL '3 days 2 hours', NOW() - INTERVAL '3 days 90 minutes', NOW() - INTERVAL '3 days 2 hours', NOW() - INTERVAL '3 days 90 minutes')
  RETURNING id INTO ci_id;
  INSERT INTO evidence_items (check_in_id, kind, text_content, created_at)
  VALUES (ci_id, 'text', 'Добавил первый экран landing: сразу виден продукт, proof loop и CTA “начать с одним кругом”.', NOW() - INTERVAL '3 days 2 hours')
  RETURNING id INTO evidence_id;
  UPDATE check_ins SET public_attachment_ids = ARRAY[evidence_id] WHERE id = ci_id;
  INSERT INTO check_in_reviews (check_in_id, reviewer_user_id, decision, comment, created_at)
  VALUES (ci_id, buddy_id, 'approved', 'Теперь это не generic SaaS hero, а понятный продукт.', NOW() - INTERVAL '3 days 90 minutes');

  INSERT INTO check_ins (goal_id, owner_user_id, status, is_public_example, submitted_at, approved_at, created_at, updated_at)
  VALUES (goal_launch_id, demo_user_id, 'approved', TRUE, NOW() - INTERVAL '2 days 3 hours', NOW() - INTERVAL '2 days 2 hours', NOW() - INTERVAL '2 days 3 hours', NOW() - INTERVAL '2 days 2 hours')
  RETURNING id INTO ci_id;
  INSERT INTO evidence_items (check_in_id, kind, text_content, created_at)
  VALUES (ci_id, 'text', 'Подключил waitlist form и записал первые 12 заявок в таблицу. 3 человека оставили конкретные цели.', NOW() - INTERVAL '2 days 3 hours')
  RETURNING id INTO evidence_id;
  UPDATE check_ins SET public_attachment_ids = ARRAY[evidence_id] WHERE id = ci_id;
  INSERT INTO check_in_reviews (check_in_id, reviewer_user_id, decision, comment, created_at)
  VALUES (ci_id, buddy_id, 'approved', 'Это уже traction artifact, не просто дизайн.', NOW() - INTERVAL '2 days 2 hours');

  INSERT INTO check_ins (goal_id, owner_user_id, status, is_public_example, submitted_at, created_at, updated_at)
  VALUES (goal_launch_id, demo_user_id, 'submitted', FALSE, NOW() - INTERVAL '24 minutes', NOW() - INTERVAL '34 minutes', NOW() - INTERVAL '24 minutes')
  RETURNING id INTO ci_id;
  INSERT INTO evidence_items (check_in_id, kind, text_content, created_at)
  VALUES (ci_id, 'text', 'Сегодня сверстал pricing block: solo accountability, buddy circle, team pilot. Нужен review по тексту оффера.', NOW() - INTERVAL '24 minutes');

  -- Buddy-owned proof, so demo user sees reviewable activity in the circle feed.
  INSERT INTO check_ins (goal_id, owner_user_id, status, is_public_example, submitted_at, approved_at, created_at, updated_at)
  VALUES (goal_training_id, observer_id, 'approved', TRUE, NOW() - INTERVAL '2 days 8 hours', NOW() - INTERVAL '2 days 7 hours', NOW() - INTERVAL '2 days 8 hours', NOW() - INTERVAL '2 days 7 hours')
  RETURNING id INTO ci_id;
  INSERT INTO evidence_items (check_in_id, kind, text_content, created_at)
  VALUES (ci_id, 'text', 'Тренировка A: присед 5x5, жим 4x6, тяга 3x8. Вес ниже прежнего, зато техника чистая.', NOW() - INTERVAL '2 days 8 hours')
  RETURNING id INTO evidence_id;
  UPDATE check_ins SET public_attachment_ids = ARRAY[evidence_id] WHERE id = ci_id;
  INSERT INTO check_in_reviews (check_in_id, reviewer_user_id, decision, comment, created_at)
  VALUES (ci_id, demo_user_id, 'approved', 'Форма важнее эго-веса. Засчитано.', NOW() - INTERVAL '2 days 7 hours');

  INSERT INTO check_ins (goal_id, owner_user_id, status, is_public_example, submitted_at, created_at, updated_at)
  VALUES (goal_training_id, observer_id, 'submitted', FALSE, NOW() - INTERVAL '18 minutes', NOW() - INTERVAL '28 minutes', NOW() - INTERVAL '18 minutes')
  RETURNING id INTO ci_id;
  INSERT INTO evidence_items (check_in_id, kind, text_content, created_at)
  VALUES (ci_id, 'text', 'Тренировка B: становая 4x5, подтягивания 5 подходов, планка 3 минуты. Жду печать бадди.', NOW() - INTERVAL '18 minutes');

  INSERT INTO ai_notifications (user_id, feature, title, body, actions, created_at)
  VALUES
    (
      demo_user_id,
      'proof_draft',
      'AI собрал черновик пруфа',
      'Из сегодняшних заметок получается сильный proof: screenshot pack + 5 выбранных кадров + вывод, какой кадр лучше объясняет accountability loop.',
      '[{"label":"Открыть дашборд","action":"open_dashboard","url":"/dashboard"},{"label":"Скрыть","action":"dismiss"}]'::jsonb,
      NOW() - INTERVAL '12 minutes'
    ),
    (
      demo_user_id,
      'streak_milestone',
      'Серия держится 9 пруфов',
      'Самый убедительный паттерн сейчас — ежедневный внешний артефакт. Не расширяй scope, просто продолжай публиковать доказательства движения.',
      '[{"label":"Сдать следующий пруф","action":"open_dashboard","url":"/dashboard"}]'::jsonb,
      NOW() - INTERVAL '35 minutes'
    ),
    (
      demo_user_id,
      'buddy_stalled',
      'В ленте есть пруф на проверку',
      'Илья сдал тренировку 18 минут назад. Быстрое одобрение усилит его comeback streak.',
      '[{"label":"Открыть ленту","action":"open_feed","url":"/feed"}]'::jsonb,
      NOW() - INTERVAL '50 minutes'
    );

  INSERT INTO ai_proof_drafts (user_id, team_id, goal_id, note_ids, rationale, confidence, created_at)
  VALUES
    (
      demo_user_id,
      team_id,
      goal_pitch_id,
      '{}',
      'AI предлагает собрать один proof из screenshot pack, списка кадров и короткого вывода про то, какой экран лучше объясняет продукт.',
      'high',
      NOW() - INTERVAL '14 minutes'
    ),
    (
      demo_user_id,
      team_id,
      goal_launch_id,
      '{}',
      'AI предлагает оформить pricing block как отдельный proof: что изменилось, какой оффер стал яснее, какой вопрос остался для buddy review.',
      'medium',
      NOW() - INTERVAL '26 minutes'
    );
END $$;

COMMIT;

SELECT
  'demo@proof-forge.local' AS login_email,
  (SELECT COUNT(*) FROM goals g JOIN users u ON u.id = g.owner_user_id WHERE u.email = 'demo@proof-forge.local') AS owned_goals,
  (
    SELECT COUNT(*)
    FROM check_ins ci
    JOIN goals g ON g.id = ci.goal_id
    WHERE g.circle_id IN (SELECT id FROM circles WHERE invite_code IN ('PFDEMO', 'PFDEMO2'))
  ) AS circle_proofs,
  (SELECT COUNT(*) FROM ai_notifications n JOIN users u ON u.id = n.user_id WHERE u.email = 'demo@proof-forge.local' AND n.dismissed_at IS NULL) AS active_ai_notifications;
