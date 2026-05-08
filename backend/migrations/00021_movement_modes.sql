-- +goose Up
-- +goose StatementBegin

ALTER TABLE goals
  ADD COLUMN movement_mode TEXT NOT NULL DEFAULT 'regular_rhythm'
    CHECK (movement_mode IN (
      'single_proof',
      'regular_rhythm',
      'challenge',
      'work_initiative',
      'free_goal'
    )),
  ADD COLUMN rhythm_cadence TEXT
    CHECK (rhythm_cadence IN ('daily', 'weekly', 'biweekly', 'custom')),
  ADD COLUMN challenge_duration_days INTEGER
    CHECK (challenge_duration_days IN (7, 14, 28, 42)),
  ADD COLUMN challenge_starts_at TIMESTAMPTZ,
  ADD COLUMN challenge_ends_at TIMESTAMPTZ;

ALTER TABLE goals
  ADD CONSTRAINT goals_rhythm_cadence_required
    CHECK (movement_mode != 'regular_rhythm' OR rhythm_cadence IS NOT NULL),
  ADD CONSTRAINT goals_challenge_duration_required
    CHECK (movement_mode != 'challenge' OR challenge_duration_days IS NOT NULL);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE goals
  DROP CONSTRAINT IF EXISTS goals_rhythm_cadence_required,
  DROP CONSTRAINT IF EXISTS goals_challenge_duration_required,
  DROP COLUMN IF EXISTS movement_mode,
  DROP COLUMN IF EXISTS rhythm_cadence,
  DROP COLUMN IF EXISTS challenge_duration_days,
  DROP COLUMN IF EXISTS challenge_starts_at,
  DROP COLUMN IF EXISTS challenge_ends_at;

-- +goose StatementEnd
