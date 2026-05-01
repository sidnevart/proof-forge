---
name: brand-voice-system
description: Build a voice and messaging system for ProofForge that feels sharp, credible, and not self-help generic.
---

## Purpose
- Use this skill to build a voice and messaging system for ProofForge that feels sharp, credible, and not self-help generic.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- A reusable system definition with principles, examples, and boundaries.
- Canonical rules other teams can apply without reinterpretation.
- A shortlist of exceptions or edge cases that need deliberate handling.

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
1. Define the few principles that must stay stable across the product.
2. Translate principles into reusable patterns, examples, and anti-patterns.
3. Test the system against core ProofForge scenarios for coherence.
4. Trim anything decorative that does not improve decision quality.
5. Publish the system in a format that can guide future contributors.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
