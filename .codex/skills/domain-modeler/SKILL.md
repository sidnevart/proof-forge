---
name: domain-modeler
description: Model the core entities, invariants, and lifecycle rules behind goals, pacts, evidence, approval, and reporting.
---

## Purpose
- Use this skill to model the core entities, invariants, and lifecycle rules behind goals, pacts, evidence, approval, and reporting.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- A domain model covering entities, relationships, invariants, and lifecycle transitions.
- A glossary of terms that removes ambiguity across product and engineering.
- Boundary notes describing what belongs in the model and what stays outside it.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Prefer explicit boundaries, boring reliability, and maintainable Go services over clever coupling.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. List the core nouns, states, and rules implied by the feature or product area.
2. Separate domain invariants from transport or storage concerns.
3. Model transitions, permissions, and historical record needs explicitly.
4. Rename fuzzy concepts until the model becomes unambiguous.
5. Publish the resulting glossary and invariants for downstream use.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
