# ProofForge Teams Phases 1-4 — Index and Handoff

**Дата:** 2026-05-07
**Статус:** handoff/index для фазовых design specs

---

## Purpose

Этот документ связывает четыре отдельные спеки после Phase 0. Он не заменяет их и не является implementation plan. Его задача — сохранить фазовые границы из большой Teams spec и показать, где какой worker должен читать подробности.

## Source of Truth

- Общая продуктовая спека: `docs/superpowers/specs/2026-05-07-proofforge-teams-habit-loop-design.md`
- Phase 0 plan: `docs/superpowers/plans/2026-05-07-proofforge-teams-phase-0-foundation.md`
- Phase 1: `docs/superpowers/specs/2026-05-07-proofforge-teams-phase-1-daily-log-streak-design.md`
- Phase 2: `docs/superpowers/specs/2026-05-07-proofforge-teams-phase-2-weekly-proof-team-review-design.md`
- Phase 3: `docs/superpowers/specs/2026-05-07-proofforge-teams-phase-3-ai-personalization-design.md`
- Phase 4: `docs/superpowers/specs/2026-05-07-proofforge-teams-phase-4-analytics-observability-privacy-design.md`

## Phase Map

| Phase | Scope | Must preserve | Exit signal |
|---|---|---|---|
| 1 | Daily learning-log + per-team streak | Daily-log raw content private to author | Alpha users log >=4 working days/week |
| 2 | Weekly proof assembly + team feed + lead/trusted review | Proof requires artifact; daily-log only becomes visible when author includes it | >=70% proofs reviewed within 48h |
| 3 | AI evening ping, assemble suggestion, lead briefing | LLM never critical path; metadata-only/default privacy | Assemble acceptance >=60%, no privacy leak |
| 4 | Analytics events, Grafana, retention, quality gates | No raw text in analytics/Grafana/logs | Dashboards live, quality/privacy gates green |

## Sequencing Rules

1. Phase 1 starts only after Phase 0 teams/memberships/authz foundation is merged or stable enough for the worker.
2. Phase 2 starts only after Phase 1 daily-log and streak are working in alpha or there is an explicit product waiver.
3. Phase 3 starts only after Phase 2 proof lifecycle works without AI.
4. Phase 4 can start preparatory dashboard SQL and event registry work earlier, but release gate ownership belongs after Phase 3 because it validates all prior flows.

## Cross-Phase Invariants

- Team mode is not a generic habit tracker. Daily-log exists to feed weekly proof, not to collect streak taps.
- `team` remains separate from `circle`; roles are `lead`, `trusted_approver`, `member`.
- Streak is per `(user_id, team_id)` and based on working-day learning-log behavior.
- Weekly proof requires structured artifact.
- Lead/trusted approval is mandatory for proof status; AI never approves.
- Raw daily-log content is private to author unless the author intentionally includes it in a submitted proof.
- Analytics and Grafana never store or display raw daily-log/proof/comment text.
- AI default is `metadata-only`; `full` requires team mode plus individual consent.

## Handoff Notes

- Do not modify `backend/*`, `web/*`, or `migrations/*` while writing or reviewing these specs unless the implementation task explicitly starts.
- Do not edit the existing Phase 0 plan as part of Phase 1-4 spec work.
- Implementation plans should be one per phase, not one mega-plan.
- Each phase implementation plan should repeat the relevant privacy gates and smoke gates from its spec.
