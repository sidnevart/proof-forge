---
name: smoke-test-release
description: Run post-deploy smoke coverage for the highest-risk ProofForge user and operator journeys.
---

## Purpose
- Use this skill to run post-deploy smoke coverage for the highest-risk ProofForge user and operator journeys.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- An execution checklist with results captured per step.
- A list of blockers or skipped steps requiring follow-up.
- A final pass or fail summary tied to evidence.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Do not trade away product coherence or operational readiness for speed.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Load the checklist, required environment, and success thresholds.
2. Run steps in the intended order and capture the result of each one.
3. Stop on hard blockers instead of papering over them.
4. Summarize failures with exact next actions.
5. Publish a final status that another operator can audit.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
