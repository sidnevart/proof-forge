---
name: name-brand-sprint
description: Explore, score, and refine product and campaign naming options that fit ProofForge's accountability premise.
---

## Purpose
- Use this skill to explore, score, and refine product and campaign naming options that fit ProofForge's accountability premise.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- A strategy brief with positioning choices, target audience assumptions, and differentiators.
- A small set of evaluated options with a recommended direction.
- Messaging or brand constraints the rest of the system should honor.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Keep the brand sharp, specific, and distinct from generic wellness or habit language.
- Decisions must strengthen trust, proof, and accountability rather than gamified fluff.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Clarify the target audience, competing alternatives, and the outcome this strategy must unlock.
2. Generate a small set of options that are meaningfully different from one another.
3. Score the options against distinctiveness, clarity, and ProofForge fit.
4. Recommend one direction with reasoning, trade-offs, and constraints.
5. Capture the resulting guidance so design and product work can reuse it consistently.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
