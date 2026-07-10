# Frontend E2E Rules

Frontend e2e coverage for this skill means proving the browser-visible journey of the slice across the real frontend, real backend, and the user-critical state transitions between them.

## Critical Journeys

- The primary user journey that makes the slice valuable.
- The highest-risk role boundary or actor handoff in the slice.
- The most important failure or rejection path the user can actually hit.
- Any follow-up view where the user must see the result of a prior action.

## What E2E Is And Is Not

- E2E proves multi-step behavior through the browser and networked application stack.
- Component tests, interaction tests, and story-level checks do not count as e2e coverage for this skill.
- Visual snapshots alone do not count as e2e coverage.
- API-only tests do not count as e2e coverage, even if they touch several services.

## Prioritization Rules

- Start with one journey that proves the slice can be used end to end.
- Add the actor boundary or handoff next if the slice includes permissions, buddy actions, or cross-user behavior.
- Add the highest-value failure path before lower-value cosmetic paths.
- If time is constrained, keep breadth on critical journeys before depth on edge styling or minor copy states.

## If E2E Infrastructure Is Missing

- Say so explicitly. Missing Playwright, missing fixtures, or no runnable browser harness is a gap.
- Do not relabel component tests as e2e to fill the hole.
- Fall back to documenting the missing journey, the blocked command, and the minimum harness needed to close the gap.
- Keep the slice verdict honest: missing e2e infrastructure means e2e readiness is `missing`, not `covered`.

## Evidence Expectations

- Tie every e2e claim to an exact command, spec file, or explicit gap.
- Keep scenario names aligned with the scenario matrix so backend and frontend evidence can be compared directly.
- If backend integration already proves a failure mode better than the browser can, the browser still needs at least the user-visible journey for that slice.
