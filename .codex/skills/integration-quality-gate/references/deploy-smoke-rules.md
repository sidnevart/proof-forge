# Deploy Smoke Rules

Deploy smoke validation is the release-time check that the slice still works after deployment. Use `docs/ops/smoke-test.md` as the base checklist, then compare the slice against it instead of assuming the generic smoke run is automatically sufficient.

## How To Compare A Slice Against The Smoke Runbook

- Start with `docs/ops/smoke-test.md` and identify which existing steps already exercise the slice.
- Mark each mapped smoke step with its explicit expected result and blocker condition from the runbook.
- Identify slice-critical behavior that is not covered by any smoke step and record it as a smoke gap.
- If the runbook has a related step but does not prove the slice-specific expectation, mark it as only partial smoke coverage.

## Expected And Blocker Conditions

- Every smoke claim must name the exact command or runbook step used.
- Every smoke claim must also name the expected result, not just "step passed".
- Every blocker must be explicit: which condition would stop release, and why it matters for the slice.
- If a command was not run fresh in the current deploy window, the status is `stale`, not `covered`.

## Minimum Recording Format

| Smoke Step | Slice Relevance | Expected | Blocker If | Status | Notes |
|---|---|---|---|---|---|
| Runbook step or custom step | Why this step matters to the slice | Exact expected result | Exact blocker condition | covered / partially covered / missing / stale | command, deploy id, or gap |

## Gap Policy

- Missing smoke coverage is a gap even when backend integration and frontend e2e are strong.
- A stale runbook, stale command set, or stale evidence is also a gap.
- If the slice needs an additional post-deploy action beyond the current runbook, record that action and mark the current smoke coverage as partial or missing until it exists.
