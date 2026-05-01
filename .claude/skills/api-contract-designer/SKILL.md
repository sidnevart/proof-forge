---
name: api-contract-designer
description: Design versionable HTTP and webhook contracts with explicit payloads, errors, and auth boundaries.
---

## Purpose
- Use this skill to design versionable HTTP and webhook contracts with explicit payloads, errors, and auth boundaries.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- A flow or interaction design with states, transitions, and edge cases.
- A recommendation grounded in clarity, trust, and ProofForge-specific behavior.
- A concise set of UI copy or structure notes where they matter.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Prefer explicit boundaries, boring reliability, and maintainable Go services over clever coupling.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Identify the user intent, the key decision points, and the failure cases.
2. Map the normal flow first, then layer in empty, loading, and error states.
3. Compare a small number of approaches and choose the clearest one.
4. Document the chosen interaction with enough detail to build or test it.
5. Check that the flow reinforces proof, approval, and progress visibility.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
