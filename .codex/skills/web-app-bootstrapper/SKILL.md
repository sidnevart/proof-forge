---
name: web-app-bootstrapper
description: Set up the frontend application structure, tooling, shell layout, and cross-cutting UX conventions.
---

## Purpose
- Use this skill to set up the frontend application structure, tooling, shell layout, and cross-cutting UX conventions.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- A scaffold plan with directories, foundational files, conventions, and setup steps.
- The minimal local tooling and verification path needed for the scaffold.
- Guardrails that prevent the scaffold from turning into accidental production code.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Preserve a distinctive, high-quality visual identity while keeping states and data flow legible.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Define the scope of the scaffold and what intentionally remains unimplemented.
2. Lay out the top-level directories, conventions, and integration points.
3. Specify setup steps, placeholders, and developer ergonomics.
4. Add verification steps so the scaffold stays coherent over time.
5. Document the next implementation layers rather than pre-building them.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
