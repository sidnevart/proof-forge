---
name: ui-kit-builder
description: Define the reusable component kit, tokens, states, and accessibility rules for the web app.
---

## Purpose
- Use this skill to define the reusable component kit, tokens, states, and accessibility rules for the web app.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- A concrete build brief with target files, interfaces, tests, and rollout notes.
- Required validation, observability, and failure-handling expectations.
- A minimal implementation path that fits the surrounding architecture.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Preserve a distinctive, high-quality visual identity while keeping states and data flow legible.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Load the affected context and identify the interfaces this work touches.
2. Define the smallest implementation slice that delivers usable value.
3. Specify the files, data flow, state transitions, and validation rules involved.
4. List the tests, smoke checks, and operator signals required for safe delivery.
5. Return an implementation-ready brief that another engineer can execute cleanly.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
