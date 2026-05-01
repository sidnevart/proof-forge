---
name: product-readiness-reviewer
description: Review whether the release is coherent, safe, branded, and worthy of real-user exposure.
---

## Purpose
- Use this skill to review whether the release is coherent, safe, branded, and worthy of real-user exposure.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- A findings-first review with severity, rationale, and affected areas.
- Residual risks, missing tests, and follow-up recommendations.
- A clear pass, conditional pass, or fail recommendation.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Do not trade away product coherence or operational readiness for speed.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Load the scope, the changed area, and the relevant expectations or policy.
2. Look for the highest-risk failure modes first.
3. State findings with evidence and concrete impact, not vague opinions.
4. Call out missing proof where confidence is limited.
5. Summarize the release implication after the findings are clear.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
