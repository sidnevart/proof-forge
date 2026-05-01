---
name: project-orchestrator
description: Coordinate end-to-end ProofForge delivery across product, design, backend, frontend, AI, QA, and release workstreams.
---

## Purpose
- Use this skill to coordinate end-to-end ProofForge delivery across product, design, backend, frontend, AI, QA, and release workstreams.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- An execution brief with scope, dependencies, and immediate next actions.
- A dependency map with blockers, parallel tracks, and handoff points.
- Verification and release checkpoints tied to the requested milestone.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Bias toward coordination artifacts that unblock multiple workers without hiding ownership.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Load the current milestone, affected workstreams, and open decisions.
2. Clarify the narrowest useful outcome and the constraints around time, quality, and staffing.
3. Sequence the work into explicit tracks with owners, dependencies, and review gates.
4. Surface the risks, escalation points, and verification steps that must not be skipped.
5. Publish the orchestration brief in a format the next worker can execute immediately.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
