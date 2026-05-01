---
name: release-manager
description: Coordinate release scope, sequencing, approvals, comms, and go/no-go decisions.
---

## Purpose
- Use this skill to coordinate release scope, sequencing, approvals, comms, and go/no-go decisions.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- A release plan with scope, sequencing, owners, approval points, and fallback paths.
- Go or no-go criteria and the evidence required to make that call.
- Communication notes for engineering, support, and stakeholders.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Do not trade away product coherence or operational readiness for speed.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Confirm the release scope, the constraints, and the blast radius.
2. Sequence technical, operational, and communication tasks explicitly.
3. Define the go or no-go criteria before execution begins.
4. Make rollback, smoke, and monitoring steps unavoidable.
5. Publish the release brief with owners and checkpoints.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
