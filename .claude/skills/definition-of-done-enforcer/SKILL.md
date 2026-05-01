---
name: definition-of-done-enforcer
description: Turn vague completion claims into explicit ProofForge acceptance criteria, verification checks, and release gates.
---

## Purpose
- Use this skill to turn vague completion claims into explicit ProofForge acceptance criteria, verification checks, and release gates.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- A definition of done checklist with objective pass conditions.
- Required verification commands, reviews, and operator checks.
- A list of missing evidence when a task is not actually complete.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Bias toward coordination artifacts that unblock multiple workers without hiding ownership.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Translate the request into observable outcomes instead of vague success language.
2. Define the tests, reviews, docs, and smoke checks needed to trust the result.
3. Compare the current state against the checklist and mark any gap explicitly.
4. Reject ambiguous completion claims until evidence exists.
5. Publish the final gate list with no hidden assumptions.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
