---
name: bug-reproducer
description: Reduce reported issues to deterministic steps, observable evidence, and likely fault boundaries.
---

## Purpose
- Use this skill to reduce reported issues to deterministic steps, observable evidence, and likely fault boundaries.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- A deterministic reproduction recipe with preconditions, steps, and observed output.
- A narrowed fault boundary describing where the issue likely lives.
- Any artifacts, logs, or test cases that make the bug easier to fix.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Favor evidence, reproducibility, and explicit exit criteria over vague quality claims.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Collect the report, environment details, and expected behavior.
2. Reduce the issue to the fewest possible steps that still trigger it.
3. Capture the exact observed result, including logs or screenshots if needed.
4. Vary one condition at a time to narrow the fault boundary.
5. Package the reproduction so another engineer can run it quickly.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
