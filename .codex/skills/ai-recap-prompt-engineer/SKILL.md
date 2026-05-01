---
name: ai-recap-prompt-engineer
description: Create the recap prompt strategy, guardrails, and evaluation criteria for weekly AI summaries.
---

## Purpose
- Use this skill to create the recap prompt strategy, guardrails, and evaluation criteria for weekly AI summaries.
- Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.
- Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.

## Inputs
- The current request, milestone, or problem statement.
- Relevant repository context: files, docs, decisions, and active constraints.
- Quality, release, and operational expectations that affect the work.

## Outputs
- A prompt or model interaction strategy with structured inputs, outputs, and guardrails.
- Evaluation criteria and failure cases for the AI behavior.
- A change log of assumptions or model-specific constraints.

## Rules
- Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.
- Use `superpowers:writing-plans` before implementation starts, even for small slices.
- Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.
- Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.
- Model safety, abuse resistance, and graceful degradation before adding convenience.
- Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.
- Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.

## Workflow
1. Clarify the task the model must perform and what evidence it receives.
2. Design the prompt structure, output shape, and refusal boundaries.
3. Define evaluation cases, bad outputs, and fallback behavior.
4. Keep the prompt short enough to be maintainable but specific enough to be reliable.
5. Record the assumptions so future tuning is deliberate, not accidental.

## Definition of Done
- The output is specific enough for the next worker to act without re-discovering the basics.
- ProofForge-specific constraints, risks, and quality gates are explicit.
- Open questions and assumptions are named instead of hidden.

## Forbidden
- Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.
- Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.
- Ignoring verification, release, operational, or security concerns that materially affect the result.
