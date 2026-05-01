---
name: system-architect
description: Shape the overall ProofForge system architecture across web, API, storage, jobs, AI, and operations.
---

## Purpose
- Use this skill to shape the overall ProofForge system architecture across web, API, storage, jobs, AI, and operations.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- An architecture brief with components, interfaces, data flow, and constraints.
- Explicit trade-offs, failure modes, and operational considerations.
- A list of decisions that should become ADRs, plans, or implementation tasks.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Prefer explicit boundaries, boring reliability, and maintainable Go services over clever coupling.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Define the problem boundary and the non-negotiable constraints first.
2. Sketch the component model, ownership lines, and interface contracts.
3. Work through failure modes, data consistency, and operational behavior.
4. Choose the simplest architecture that preserves future maintainability.
5. Record trade-offs and the follow-up work required to implement safely.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
