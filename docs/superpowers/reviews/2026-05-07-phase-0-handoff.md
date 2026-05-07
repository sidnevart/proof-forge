# ProofForge Teams Phase 0 — Handoff

**Дата:** 2026-05-07
**Статус:** implementation готов к ревью/коммиту с одним условным quality-gate риском по coverage.

## Что сделано

- Добавлен backend package `backend/internal/teams/` для `teams`, `team_memberships`, ролей `lead`, `trusted_approver`, `member`, invite-flow, lead-only management и AI consent flag.
- Добавлены миграции `backend/migrations/00011_teams.sql` и `backend/migrations/00012_team_extensions.sql`.
- `00012` делает `goals.circle_id` nullable, добавляет `goals.team_id` и XOR `circle_id IS NULL OR team_id IS NULL`, чтобы Phase 2 могла создавать team-bound goals.
- Teams API смонтирован под текущим контрактом `/v1/teams`.
- Web skeleton добавлен в `web/app/(product)/teams/`, `web/components/product/teams-list.tsx`, `web/lib/api.ts`, `web/lib/types.ts`.
- Invite-code теперь возвращается и показывается только lead. Обычные members видят team detail без кода.
- Removed membership не может rejoin по старому invite-code; left membership может rejoin.

## Спеки Phase 1-4

- `docs/superpowers/specs/2026-05-07-proofforge-teams-phase-1-daily-log-streak-design.md`
- `docs/superpowers/specs/2026-05-07-proofforge-teams-phase-2-weekly-proof-team-review-design.md`
- `docs/superpowers/specs/2026-05-07-proofforge-teams-phase-3-ai-personalization-design.md`
- `docs/superpowers/specs/2026-05-07-proofforge-teams-phase-4-analytics-observability-privacy-design.md`
- `docs/superpowers/specs/2026-05-07-proofforge-teams-phases-1-4-index.md`

Все новые контракты используют текущий backend-префикс `/v1`.

## Проверки

- `backend: go test ./...` — pass.
- `backend: go build ./...` — pass.
- `backend: TEST_DATABASE_URL=postgres://.../proofforge_test_codex_20260507_2011 go test ./internal/teams/...` — pass.
- `backend: MIGRATION_DATABASE_URL=postgres://.../proofforge_migrations_codex_20260507_2012 go test ./migrations` — pass, включает destructive `up/down/up` на отдельной базе.
- `backend: TEST_DATABASE_URL=postgres://.../proofforge_test_codex_20260507_2011 go test -cover ./internal/teams/...` — pass, coverage 64.8%.
- `web: ./node_modules/.bin/tsc --noEmit` — pass.
- `web: npm test` — pass, 17 files / 83 tests.
- `web: npm run lint` — pass.
- `web: npm run build` — pass.

## Остаточные риски

- План Phase 0 требует `teams/` coverage >=85%, фактическое значение после DB-backed repository tests — 64.8%. Основная причина: production `postgres_repository.go` остаётся крупным denominator. Для 85% нужен отдельный repository coverage pass на `ListMyTeams`, `GetTeamForUser`, `RemoveMember`, `SetAIConsent`, `RegenerateInviteCode`, error branches и `scanDetail`.
- Полный `TEST_DATABASE_URL=... go test -p 1 -tags=integration ./...` ранее падал на существующих `internal/platform/app` E2E: отсутствующие legacy routes для stakes/milestones и колонка `invites.acceptance_token`. `internal/teams` на отдельной test DB проходит.
- Manual browser smoke lead-create/member-join не проводился в этой сессии. Docker dev stack уже был поднят, но финальная проверка ограничена automated gates.

## Следующее действие

1. Перед merge решить, принимаем ли Phase 0 coverage 64.8% как conditional pass или добиваем repository coverage до 85%.
2. Если нужен manual smoke: через web `/teams` создать команду lead user, скопировать invite-code, join вторым user через `/teams/join?code=...`, проверить что member detail не показывает invite-code.
3. Phase 1 implementation plan писать отдельно от Phase 2-4; не объединять всё в один mega-plan.
