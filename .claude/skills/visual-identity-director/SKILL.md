---
name: visual-identity-director
description: Set art direction, typography, color, motion, and visual principles for a distinctive ProofForge identity.
---

## Purpose
- Use this skill to set art direction, typography, color, motion, and visual principles for a distinctive ProofForge identity.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- An art direction brief covering color, typography, layout, motion, and imagery.
- Concrete do and do-not guidance for future design work.
- A shortlist of references or motifs that reinforce ProofForge identity.

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
1. Start from the brand promise and the emotional tone the interface should create.
2. Choose a clear visual direction rather than a safe average SaaS look.
3. Define the primary ingredients that make the direction repeatable.
4. Pressure-test the direction across landing, product, and shareable surfaces.
5. Document guardrails so later builders preserve coherence.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
