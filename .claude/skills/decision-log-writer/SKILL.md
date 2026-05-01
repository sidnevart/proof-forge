---
name: decision-log-writer
description: Capture architectural, product, and operational decisions in durable records with rationale and consequences.
---

## Purpose
- Use this skill to capture architectural, product, and operational decisions in durable records with rationale and consequences.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- A concise written artifact with the decision, rationale, impact, and next actions.
- Explicit links to affected files, plans, or operational procedures.
- A record another engineer can trust without re-interviewing the original author.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Bias toward coordination artifacts that unblock multiple workers without hiding ownership.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Gather the facts, the decision point, and the relevant constraints.
2. Write the essential context, the choice made, and the trade-offs accepted.
3. Call out the downstream consequences and what must happen next.
4. Remove filler so the artifact can be consumed quickly under pressure.
5. Store the note where future work will find it.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
