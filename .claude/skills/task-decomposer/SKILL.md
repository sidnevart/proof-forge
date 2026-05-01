---
name: task-decomposer
description: Split large ProofForge initiatives into independently deliverable, testable slices with clear sequencing and ownership.
---

## Purpose
- Use this skill to split large ProofForge initiatives into independently deliverable, testable slices with clear sequencing and ownership.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- A breakdown of the request into small, testable, independently reviewable slices.
- Recommended execution order and a map of what can happen in parallel.
- Clear acceptance criteria for each slice.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Bias toward coordination artifacts that unblock multiple workers without hiding ownership.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Read the request and identify the real unit of value being delivered.
2. Separate foundation, feature, QA, and release concerns instead of mixing them.
3. Break each concern into slices with one owner and one clear exit condition.
4. Mark the blockers, prerequisites, and safe parallelization boundaries.
5. Return a sequence that minimizes risk and rework.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
