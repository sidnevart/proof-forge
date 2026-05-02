# Scenario Matrix

Use this matrix before running any integration-quality gate commands. The matrix is the canonical map of what must be proven, which evidence exists, and where the slice is still exposed.

## Coverage States

- `covered`: fresh evidence exists for this scenario in the current change window.
- `partially covered`: some evidence exists, but an important boundary, role, or failure mode is still unproven.
- `missing`: no meaningful evidence exists yet.
- `stale`: evidence exists, but it is old, from a different slice, or no longer matches current behavior.

## Canonical Template

| Scenario | User/System Path | Risk if Broken | Backend Integration | Frontend E2E | Deploy Smoke | Status | Notes |
|---|---|---|---|---|---|---|---|
| Primary happy path | | | command, file, or gap | command, spec, or gap | smoke step, command, or gap | covered / partially covered / missing / stale | |
| Authorization boundary | | | | | | covered / partially covered / missing / stale | |
| Invalid transition or rejected input | | | | | | covered / partially covered / missing / stale | |
| Derived read model or downstream visibility | | | | | | covered / partially covered / missing / stale | |
| Recovery or operator-facing check | | | | | | covered / partially covered / missing / stale | |

## Example: owner registration to proof submission slice

| Scenario | User/System Path | Risk if Broken | Backend Integration | Frontend E2E | Deploy Smoke | Status | Notes |
|---|---|---|---|---|---|---|---|
| Registration -> goal creation | Owner registers, creates a goal, and receives `pending_buddy_acceptance` with invite data | New owners cannot enter the accountability loop | `TestRegistrationGoalCreationAndDashboardFlow` in `backend/internal/platform/app/api_integration_test.go` covers registration and goal creation | missing | `docs/ops/smoke-test.md` steps 2 and 3 cover this after deploy | covered | This is the only slice in the current app integration suite with fresh HTTP-level proof |
| Dashboard visibility after goal creation | Newly created goal appears in `/v1/dashboard` with the correct summary counts | Writes succeed but the owner cannot see or trust the current state | `TestRegistrationGoalCreationAndDashboardFlow` verifies `total_goals`, `pending_buddy_acceptance`, and dashboard goal count | missing | `docs/ops/smoke-test.md` step 4 checks dashboard visibility | covered | This is a required derived read-model row, not optional reporting |
| Buddy invite acceptance | Invited buddy loads `/v1/invites/{token}` and accepts the invite as the invited email | Owner is stuck in pending state and the buddy cannot join the goal | missing | missing | missing | missing | Routes exist in `backend/internal/goals/http_handler.go` and `web/app/invites/[token]/page.tsx`, but this worktree has no committed integration, e2e, or smoke proof for the flow |
| Check-in submission | Goal owner creates a draft check-in, adds evidence, and submits it | Proof artifacts cannot move into buddy review | missing | missing | `docs/ops/smoke-test.md` steps 5a, 5b, and 5c cover the deployed path | partially covered | Production smoke exists, but there is no committed backend integration or browser e2e coverage in this worktree |
| Approval -> recap visibility | Buddy approves a submitted check-in and the recap becomes visible to owner and buddy | Proof can be submitted but the accountability loop never closes | missing | missing | `docs/ops/smoke-test.md` step 6 only proves recaps are empty before approval; no approval or recap generation smoke step exists | missing | Current routes support review actions and recap reads, but this worktree does not contain end-to-end proof for approval-driven recap visibility |

## Rules

- Build the matrix first. Do not start ad hoc command execution without it.
- Name the user role or system actor for each scenario. "API works" is too vague.
- Call out missing and stale evidence directly. Do not hide them inside optimistic notes.
- Treat backend integration, frontend e2e, and deploy smoke as separate evidence columns even when one command touches multiple layers.
- If a scenario matters for release readiness but no test or smoke step exists yet, keep the row and mark the gap.
