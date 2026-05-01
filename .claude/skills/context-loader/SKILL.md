---
name: context-loader
description: Assemble the smallest complete context package needed for a ProofForge task before execution starts.
---

## Purpose
- Use this skill to assemble the smallest complete context package needed for a ProofForge task before execution starts.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- A minimal context pack with the exact files, docs, decisions, and unknowns needed next.
- A short summary of what matters, what does not, and what still needs confirmation.
- A handoff-ready list of references to load before implementation or review.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Bias toward coordination artifacts that unblock multiple workers without hiding ownership.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Identify the task, the affected subsystem, and the likely adjacent concerns.
2. Pull only the files, docs, and prior decisions that materially affect execution.
3. Summarize the relevant context with links or paths and explicit unknowns.
4. Flag stale assumptions, missing documents, or conflicting guidance.
5. Deliver a context pack sized for action, not for archival completeness.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
