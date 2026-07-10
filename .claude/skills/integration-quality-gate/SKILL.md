---
name: integration-quality-gate
description: Validate one ProofForge feature slice for release readiness across the full integration stack: scenario matrix, backend integration coverage, frontend e2e coverage, deploy smoke validation, and a final verdict with blockers and gaps called out explicitly.
---

## Purpose
- Use this skill to run a release-grade integration quality gate for a ProofForge feature slice.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Preserve the approved ProofForge product and brand direction; do not drift into generic SaaS, wellness, or streak-first framing while defining scenarios or verdicts.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.
- Force an evidence-first sequence: scenario matrix, backend integration, frontend e2e, deploy smoke, then verdict.
- Surface blockers, missing coverage, and stale validation instead of collapsing them into a vague pass.

## Inputs
- The feature slice, change set, or branch under review.
- Relevant product constraints, acceptance criteria, and prior implementation notes.
- Current test commands, smoke commands, and any known operational caveats.

## Outputs
- A scenario matrix that names critical paths, gaps, and ownership.
- A verification record for backend integration, frontend e2e, and deploy smoke checks.
- A final quality-gate verdict with blockers, concerns, and exact follow-up actions.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Build the scenario matrix first; do not start test execution before the matrix exists.
- Run backend integration verification before frontend e2e so contract and persistence failures are caught earlier.
- Do not count component tests, isolated UI interaction tests, or story-level checks as e2e coverage.
- Treat a stale smoke checklist or stale smoke commands as an explicit gap, not as evidence of current readiness.
- Run fresh verification commands before claiming completion; prior logs or old CI results are not enough.
- Do not treat this skill as a generic final checklist or a post-deploy-only smoke pass; it is for one feature slice that must be validated across scenario design, backend integration, frontend e2e, and deploy smoke as a single release decision.
- Do not trade away product coherence, deployment confidence, or observability gaps for a superficial green status.
- Keep the verdict tied to evidence, exact commands, and observed outcomes.
- Stop and call out hard blockers instead of silently narrowing scope.

## Workflow
1. Identify the feature slice, acceptance criteria, and relevant risks.
2. Write the scenario matrix covering primary flows, edge cases, integration boundaries, and smoke expectations.
3. Execute backend integration checks and record failures, skips, or missing coverage.
4. Execute frontend e2e checks only after backend integration status is known.
5. Review deploy smoke validation and mark any stale checklist, stale command set, or missing fresh run as a gap.
6. Summarize the evidence into a final verdict with blockers, concerns, and next actions.

## Definition of Done
- The scenario matrix exists and is specific enough for another operator to audit.
- Backend integration status is explicit before any frontend e2e conclusion.
- E2E claims are backed by actual end-to-end coverage rather than component-level tests.
- Smoke validation status is fresh, or any staleness is called out as a gap.
- The final verdict is supported by fresh commands and concrete evidence.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Starting with ad hoc test runs before the scenario matrix is defined.
- Presenting frontend e2e status while backend integration verification is still missing.
- Counting component tests as e2e or implying that they cover full-stack behavior.
- Substituting a repo-wide generic quality pass or a post-deploy-only smoke run for slice-level integration validation.
- Treating stale smoke documentation as equivalent to a current smoke run.
- Claiming completion from memory, old output, or inferred success without fresh commands.
