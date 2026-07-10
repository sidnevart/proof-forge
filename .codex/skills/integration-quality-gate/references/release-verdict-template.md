# Release Verdict Template

Use this exact structure at the end of the integration quality gate.

## Covered

- Scenario:
  Evidence:
  Outcome:

## Gaps

- Gap:
  Impact:
  Next action:
- If there are no gaps, write `none`.

## Blockers

- Blocker:
  Evidence:
  Required fix:
- If there are no blockers, write `none`.

## Commands

- `command`
  Result:

## Release Verdict

- Selected verdict: `ready` | `ready with known gaps` | `not ready`
- Choose exactly one value.
- Select `not ready` if any blocker exists.
- Select `ready with known gaps` if there are no blockers but at least one gap remains.
- Select `ready` only if both `Gaps` and `Blockers` are `none`.

## Why

- One short paragraph explaining why the verdict follows from the evidence above, including why the result is `ready`, `ready with known gaps`, or `not ready`.
