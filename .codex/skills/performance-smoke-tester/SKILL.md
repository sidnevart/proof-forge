---
name: performance-smoke-tester
description: Run fast performance smoke checks for key flows, queries, and background processing paths.
---

## Purpose
- Use this skill to run fast performance smoke checks for key flows, queries, and background processing paths.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- A smoke test plan with fast checks, thresholds, and expected evidence.
- A record of failures, bottlenecks, or capacity concerns.
- Recommendations for deeper investigation when smoke coverage is not enough.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Favor evidence, reproducibility, and explicit exit criteria over vague quality claims.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Choose the small set of paths most likely to reveal real performance or reliability problems.
2. Define the commands, load shape, and timing thresholds up front.
3. Run the checks and capture evidence instead of impressions.
4. Separate actual bottlenecks from noisy local variance.
5. Report the result with clear next actions.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
