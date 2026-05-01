---
name: docs-finalizer
description: Bring specs, runbooks, changelogs, and operator notes to a release-ready state.
---

## Purpose
- Use this skill to bring specs, runbooks, changelogs, and operator notes to a release-ready state.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- A final documentation pass that is complete, consistent, and release-usable.
- A short list of docs still missing if the release is not actually ready.
- Tightened wording, links, and structure that reduce operator confusion.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Do not trade away product coherence or operational readiness for speed.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Load the release scope and the documents users or operators will need.
2. Check for missing sections, stale steps, broken links, and contradictions.
3. Rewrite for speed and clarity rather than completeness theater.
4. Align the docs with the actual shipped behavior and operations model.
5. Publish the final doc status and any remaining gaps.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
