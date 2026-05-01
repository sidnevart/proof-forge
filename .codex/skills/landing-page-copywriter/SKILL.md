---
name: landing-page-copywriter
description: Write high-conviction landing page copy that explains ProofForge's proof-driven accountability loop.
---

## Purpose
- Use this skill to write high-conviction landing page copy that explains ProofForge's proof-driven accountability loop.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- Audience-specific copy options with a recommended version.
- Message hierarchy and proof points that support conversion.
- Tone guidance to keep future copy aligned.

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
1. Clarify the audience, channel, and action the copy needs to drive.
2. Write several message angles rather than polishing the first obvious one.
3. Favor concrete, proof-driven language over startup fluff.
4. Select the strongest version and tighten it for clarity and pace.
5. Document tone and message constraints for follow-on edits.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
