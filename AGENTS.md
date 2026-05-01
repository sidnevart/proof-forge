# ProofForge Agent Operating Guide

## Product context
ProofForge is a proof-based social accountability platform. Users set goals, invite a buddy, submit proof artifacts, receive buddy approval, track progress, and get AI weekly recaps.

## Non-negotiables
- Do not treat the product like a generic habit tracker.
- Preserve a distinctive brand and visual identity.
- For UI work, preserve the approved ProofForge direction from `docs/brand/identity-package.md` and `docs/architecture/frontend-brand-constraints.md`; do not drift toward generic SaaS purple-gradient dashboards or soft wellness framing.
- Prefer maintainable Go backend architecture over framework sprawl.
- Frontend quality matters: interaction design, accessibility, and visual coherence are first-class concerns.
- Every meaningful implementation path must account for tests, deployment, observability, and smoke checks.

## Skill system
- Project skills live in both `.claude/skills/` and `.codex/skills/`.
- Use the mirrored project skills before doing domain-specific work when one clearly applies.
- Default workflow:
  - `superpowers:brainstorming` before unclear product, UX, or architecture work
  - `superpowers:writing-plans` before implementation
  - `superpowers:test-driven-development` for core logic
  - `superpowers:systematic-debugging` for bugs
  - `superpowers:requesting-code-review` before completion
  - `superpowers:finishing-a-development-branch` before release

## Working style
- Start by loading only the context needed for the current task.
- Write explicit decisions to `docs/decisions/`.
- Keep specs in `docs/superpowers/specs/` and plans in `docs/superpowers/plans/`.
- Prefer small, testable vertical slices over broad partially-finished layers.
- Пиши всю проектную документацию на русском языке по умолчанию, если пользователь явно не попросил другой язык.
