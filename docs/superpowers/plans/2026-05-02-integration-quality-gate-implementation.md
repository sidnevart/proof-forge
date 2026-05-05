# Integration Quality Gate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Создать project skill `integration-quality-gate`, который стандартизирует серьёзную сценарную проверку vertical slice через backend integration, frontend e2e, deploy smoke и финальный release verdict.

**Architecture:** Каноническая версия skill создаётся в `.codex/skills/integration-quality-gate/`, после чего целиком зеркалится в `.claude/skills/integration-quality-gate/` для согласованности skill system. Регистрация skill идёт через `scripts/bootstrap_skill_system.py` и служебные setup docs, а runtime-логика skill разбивается на один `SKILL.md`, пять reference-документов и отдельный `evals/evals.json` с начальными тестовыми промптами.

**Tech Stack:** Markdown, JSON, repository skill trees, Python verification script, shell verification commands

---

## File Structure

- `docs/superpowers/specs/2026-05-02-integration-quality-gate-design.md`
  Утверждённый источник требований; используется только как reference.
- `scripts/bootstrap_skill_system.py`
  Источник правды для реестра project skills и генерации setup-инвентарей.
- `docs/setup/skills-inventory.md`
  Человекочитаемый список skills; должен включать новый skill.
- `docs/setup/skills-paths.txt`
  Полный список `SKILL.md` путей для `.claude` и `.codex`; должен включать оба зеркала.
- `.codex/skills/integration-quality-gate/SKILL.md`
  Каноническое описание workflow skill и trigger conditions.
- `.codex/skills/integration-quality-gate/references/scenario-matrix.md`
  Шаблон матрицы сценариев и пример для `circles` slice.
- `.codex/skills/integration-quality-gate/references/backend-integration-rules.md`
  Правила backend integration coverage для ProofForge.
- `.codex/skills/integration-quality-gate/references/frontend-e2e-rules.md`
  Правила критических frontend journeys и e2e coverage.
- `.codex/skills/integration-quality-gate/references/deploy-smoke-rules.md`
  Правила обновления и применения `docs/ops/smoke-test.md`.
- `.codex/skills/integration-quality-gate/references/release-verdict-template.md`
  Формат финального quality gate verdict.
- `.codex/skills/integration-quality-gate/evals/evals.json`
  Начальные eval prompts для проверки skill.
- `.claude/skills/integration-quality-gate/`
  Полное зеркало канонической skill-папки из `.codex/skills/integration-quality-gate/`.

### Task 1: Register The Skill In The Project Skill System

**Files:**
- Modify: `scripts/bootstrap_skill_system.py`
- Modify: `docs/setup/skills-inventory.md`
- Modify: `docs/setup/skills-paths.txt`
- Test: `scripts/verify_skill_system.py`

- [ ] **Step 1: Write the failing test**

```markdown
Expected failure conditions:
- `integration-quality-gate` is absent from `SKILL_GROUPS` in `scripts/bootstrap_skill_system.py`
- `integration-quality-gate` is absent from `docs/setup/skills-inventory.md`
- `docs/setup/skills-paths.txt` has no `.claude/skills/integration-quality-gate/SKILL.md`
- `docs/setup/skills-paths.txt` has no `.codex/skills/integration-quality-gate/SKILL.md`
```

- [ ] **Step 2: Run test to verify it fails**

Run: `rg -n "integration-quality-gate" scripts/bootstrap_skill_system.py docs/setup/skills-inventory.md docs/setup/skills-paths.txt`
Expected: FAIL with no matches.

- [ ] **Step 3: Write minimal implementation**

```python
# scripts/bootstrap_skill_system.py
# Add the new skill inside the "QA / Security" group immediately after
# `quality-gate-runner` or рядом с ним:
("integration-quality-gate", "Run a full ProofForge quality gate for a feature slice: scenario matrix, backend integration coverage, frontend e2e coverage, deploy smoke validation, and a final release verdict with blockers and gaps called out explicitly.", "runner"),
```

```markdown
# docs/setup/skills-inventory.md
## QA / Security
- `go-unit-test-builder`
- `go-integration-test-builder`
- `frontend-test-builder`
- `e2e-test-builder`
- `api-contract-test-builder`
- `security-reviewer`
- `privacy-reviewer`
- `performance-smoke-tester`
- `bug-reproducer`
- `quality-gate-runner`
- `integration-quality-gate`
```

```text
# docs/setup/skills-paths.txt
.claude/skills/integration-quality-gate/SKILL.md
.codex/skills/integration-quality-gate/SKILL.md
```

- [ ] **Step 4: Run test to verify it passes**

Run: `python3 scripts/verify_skill_system.py --list | rg "integration-quality-gate" && rg -n "integration-quality-gate" docs/setup/skills-inventory.md docs/setup/skills-paths.txt`
Expected: PASS with one skill-list match and references in both setup docs.

- [ ] **Step 5: Commit**

```bash
git add scripts/bootstrap_skill_system.py docs/setup/skills-inventory.md docs/setup/skills-paths.txt
git commit -m "docs: register integration quality gate skill"
```

### Task 2: Write The Core Skill Workflow

**Files:**
- Create: `.codex/skills/integration-quality-gate/SKILL.md`
- Create: `.claude/skills/integration-quality-gate/SKILL.md`
- Test: `.codex/skills/integration-quality-gate/SKILL.md`

- [ ] **Step 1: Write the failing test**

```markdown
Expected failure conditions:
- `.codex/skills/integration-quality-gate/SKILL.md` does not exist
- `.claude/skills/integration-quality-gate/SKILL.md` does not exist
- the skill frontmatter and required headings are missing
- there is no rule forcing scenario matrix -> backend integration -> frontend e2e -> deploy smoke -> release verdict order
```

- [ ] **Step 2: Run test to verify it fails**

Run: `test -f .codex/skills/integration-quality-gate/SKILL.md && test -f .claude/skills/integration-quality-gate/SKILL.md`
Expected: FAIL because neither file exists.

- [ ] **Step 3: Write minimal implementation**

```markdown
---
name: integration-quality-gate
description: Use this whenever the user asks to seriously verify a feature, add integration coverage, close end-to-end gaps, prepare a vertical slice for release, or says things like "все проверь", "серьезно подойдем к проверке", "добавь интеграционные тесты", "закрой e2e", "проверь сценарии", "готово ли это к пушу/проду". This skill is mandatory when the task involves backend integration tests, frontend end-to-end coverage, release smoke checks, or a ProofForge quality gate for a feature slice. It builds a scenario matrix first, then drives backend integration coverage, frontend e2e coverage, deploy smoke validation, and a final release verdict with blockers and uncovered scenarios called out explicitly.
---

## Purpose
- Use this skill to run a ProofForge quality gate for a feature slice instead of settling for a happy-path test.
- Keep the work centered on proof, buddy approval, social pressure, and release trustworthiness.

## Inputs
- The active feature slice, bugfix, or release candidate.
- Relevant backend routes, frontend journeys, and deploy surfaces.
- Current tests, smoke docs, and known risks.

## Outputs
- A scenario matrix with explicit coverage status.
- A concrete test plan or implementation brief for backend integration, frontend e2e, and deploy smoke.
- A final quality gate verdict with blockers, gaps, commands, and release status.

## Rules
- Always build the scenario matrix before adding or changing tests.
- Backend integration comes before frontend e2e.
- Frontend component tests do not count as e2e coverage.
- A local `go test` pass is not enough to claim release readiness.
- If `docs/ops/smoke-test.md` is stale for the slice, call that out as a gap.
- Never say "всё проверено" without fresh verification commands and explicit uncovered scenarios.

## Workflow
1. Load the slice context, affected routes, docs, and current tests.
2. Build a scenario matrix covering happy path, permissions, invalid transitions, regressions, and deploy smoke.
3. Audit existing coverage and classify each scenario as covered, partially covered, missing, or stale.
4. Close or plan backend integration gaps first.
5. Close or plan frontend e2e gaps second.
6. Update or validate deploy smoke coverage third.
7. Run verification commands and publish a final release verdict.

## Definition of Done
- The scenario matrix exists and all critical scenarios are classified.
- Backend integration, frontend e2e, and deploy smoke coverage are either implemented or explicitly called out as gaps.
- The final verdict names blockers and says `ready`, `ready with known gaps`, or `not ready`.

## Forbidden
- Declaring success from happy-path coverage alone.
- Treating unit tests as integration tests.
- Treating component tests as e2e.
- Hiding missing scenarios behind vague wording.
```

```bash
mkdir -p .codex/skills/integration-quality-gate .claude/skills/integration-quality-gate
cp .codex/skills/integration-quality-gate/SKILL.md .claude/skills/integration-quality-gate/SKILL.md
```

- [ ] **Step 4: Run test to verify it passes**

Run: `python3 scripts/verify_skill_system.py && diff -u .codex/skills/integration-quality-gate/SKILL.md .claude/skills/integration-quality-gate/SKILL.md`
Expected: PASS with verification script succeeding and no diff between mirrored files.

- [ ] **Step 5: Commit**

```bash
git add .codex/skills/integration-quality-gate/SKILL.md .claude/skills/integration-quality-gate/SKILL.md
git commit -m "docs: add integration quality gate skill workflow"
```

### Task 3: Write The Reference Pack

**Files:**
- Create: `.codex/skills/integration-quality-gate/references/scenario-matrix.md`
- Create: `.codex/skills/integration-quality-gate/references/backend-integration-rules.md`
- Create: `.codex/skills/integration-quality-gate/references/frontend-e2e-rules.md`
- Create: `.codex/skills/integration-quality-gate/references/deploy-smoke-rules.md`
- Create: `.codex/skills/integration-quality-gate/references/release-verdict-template.md`
- Create: `.claude/skills/integration-quality-gate/references/scenario-matrix.md`
- Create: `.claude/skills/integration-quality-gate/references/backend-integration-rules.md`
- Create: `.claude/skills/integration-quality-gate/references/frontend-e2e-rules.md`
- Create: `.claude/skills/integration-quality-gate/references/deploy-smoke-rules.md`
- Create: `.claude/skills/integration-quality-gate/references/release-verdict-template.md`
- Test: `.codex/skills/integration-quality-gate/references/scenario-matrix.md`

- [ ] **Step 1: Write the failing test**

```markdown
Expected failure conditions:
- the `references/` directory does not exist under the new skill
- there is no canonical scenario matrix template
- there are no layer-specific rules for backend integration, frontend e2e, deploy smoke, or release verdicts
```

- [ ] **Step 2: Run test to verify it fails**

Run: `find .codex/skills/integration-quality-gate/references -maxdepth 1 -type f | sort`
Expected: FAIL because the `references/` directory does not exist yet.

- [ ] **Step 3: Write minimal implementation**

```markdown
# .codex/skills/integration-quality-gate/references/scenario-matrix.md
# Scenario Matrix
| Scenario | Layer | Current Coverage | Required Asset | Blocker |
| --- | --- | --- | --- | --- |
| register owner -> create circle -> peer join -> goal -> invite accept -> check-in -> approve -> standings | backend integration | missing | `backend/internal/platform/app/circles_integration_test.go` | yes |
| dashboard -> create/join circle -> create goal in circle | frontend e2e | missing | `web/e2e/circles-onboarding.spec.ts` | yes |
| post-deploy circle flow smoke | deploy smoke | stale | `docs/ops/smoke-test.md` | yes |
```

```markdown
# .codex/skills/integration-quality-gate/references/backend-integration-rules.md
# Backend Integration Rules
- Test real HTTP routes through `backend/internal/platform/app/*_integration_test.go`.
- Use the shared integration DB harness from `backend/testutil`.
- Cover happy path, authz, invalid transitions, and derived read models.
- Do not count service-unit tests as integration coverage.
```

```markdown
# .codex/skills/integration-quality-gate/references/frontend-e2e-rules.md
# Frontend E2E Rules
- Reserve e2e for redirect, session, multi-step journey, and cross-surface flows.
- Do not replace route-level behavior with component tests.
- If e2e infrastructure is missing, mark it as a release gap and define the minimum bootstrap needed.
- Prioritize one critical journey per vertical slice before optional coverage.
```

```markdown
# .codex/skills/integration-quality-gate/references/deploy-smoke-rules.md
# Deploy Smoke Rules
- Compare the slice against `docs/ops/smoke-test.md`.
- Add user-visible and operator-visible checks when the flow changes.
- Every smoke step must include an explicit expected result and blocker condition.
- If a flow is not smoke-testable yet, record that as a known gap.
```

```markdown
# .codex/skills/integration-quality-gate/references/release-verdict-template.md
# Quality Gate Verdict
## Covered
- Circle happy path is covered through HTTP-level integration tests.
- Critical onboarding journey is covered through frontend e2e.

## Gaps
- Telegram-specific flows are out of scope for this slice and must be called out separately.
- Any missing deploy smoke step must be listed here with the exact missing check.

## Blockers
- Release is blocked if a critical scenario is still marked `missing`.
- Release is blocked if fresh verification commands were not run.

## Commands
- `go test ./...`
- `npm run test:e2e`
- `make verify-web`
- `make verify-deploy`

## Release Verdict
- `ready` | `ready with known gaps` | `not ready`

## Why
- Backend integration, frontend e2e, and deploy smoke together determine whether the slice is trustworthy enough to ship.
```

```bash
mkdir -p .codex/skills/integration-quality-gate/references .claude/skills/integration-quality-gate
cp -R .codex/skills/integration-quality-gate/references .claude/skills/integration-quality-gate/
```

- [ ] **Step 4: Run test to verify it passes**

Run: `find .codex/skills/integration-quality-gate/references -maxdepth 1 -type f | sort && diff -ru .codex/skills/integration-quality-gate/references .claude/skills/integration-quality-gate/references`
Expected: PASS with five reference files listed and no diff between mirrors.

- [ ] **Step 5: Commit**

```bash
git add .codex/skills/integration-quality-gate/references .claude/skills/integration-quality-gate/references
git commit -m "docs: add integration quality gate reference pack"
```

### Task 4: Add Eval Prompts For Skill Verification

**Files:**
- Create: `.codex/skills/integration-quality-gate/evals/evals.json`
- Create: `.claude/skills/integration-quality-gate/evals/evals.json`
- Test: `.codex/skills/integration-quality-gate/evals/evals.json`

- [ ] **Step 1: Write the failing test**

```markdown
Expected failure conditions:
- there is no eval file for the new skill
- the approved prompts from the design spec are not captured in a machine-readable format
```

- [ ] **Step 2: Run test to verify it fails**

Run: `test -f .codex/skills/integration-quality-gate/evals/evals.json && test -f .claude/skills/integration-quality-gate/evals/evals.json`
Expected: FAIL because the eval files do not exist.

- [ ] **Step 3: Write minimal implementation**

```json
{
  "skill_name": "integration-quality-gate",
  "evals": [
    {
      "id": 1,
      "prompt": "Добавь серьёзное интеграционное покрытие для нового circle flow и скажи, можно ли это катить на прод.",
      "expected_output": "Skill builds a scenario matrix, prioritizes backend integration, names e2e and smoke obligations, and ends with a release verdict.",
      "files": []
    },
    {
      "id": 2,
      "prompt": "Проверь vertical slice с invite, check-in и buddy approval. Нужен полный quality gate, не только happy-path.",
      "expected_output": "Skill covers permissions, invalid transitions, regression scenarios, and deploy smoke instead of stopping at the happy path.",
      "files": []
    },
    {
      "id": 3,
      "prompt": "Перед push разложи сценарии, добей пробелы по backend и e2e, и обнови smoke checklist.",
      "expected_output": "Skill produces scenario classification, missing coverage callouts, smoke checklist updates, commands run, and a clear push verdict.",
      "files": []
    }
  ]
}
```

```bash
mkdir -p .codex/skills/integration-quality-gate/evals .claude/skills/integration-quality-gate
cp -R .codex/skills/integration-quality-gate/evals .claude/skills/integration-quality-gate/
```

- [ ] **Step 4: Run test to verify it passes**

Run: `jq -r '.skill_name, (.evals | length)' .codex/skills/integration-quality-gate/evals/evals.json && diff -u .codex/skills/integration-quality-gate/evals/evals.json .claude/skills/integration-quality-gate/evals/evals.json && python3 scripts/verify_skill_system.py && diff -ru .codex/skills/integration-quality-gate .claude/skills/integration-quality-gate`
Expected: PASS with `integration-quality-gate`, `3`, a clean skill-system verification, and no diff between mirrored skill trees.

- [ ] **Step 5: Commit**

```bash
git add .codex/skills/integration-quality-gate .claude/skills/integration-quality-gate
git commit -m "docs: add integration quality gate eval prompts and verify skill package"
```
