# Brand Foundation Rollout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Превратить утверждённый identity package в рабочую brand/design foundation для ProofForge без начала продуктовой реализации.

**Architecture:** Работа идёт через короткие, проверяемые документы и лёгкие репозиторные артефакты, а не через продуктовый код. Сначала бренд переводится в нормативные документы и naming/rules artifacts, затем создаётся дизайн-фундамент для будущих UI-решений, после чего добавляются проверки и execution prompts, чтобы будущие агенты не деградировали в generic SaaS или habit-tracker направление.

**Tech Stack:** Markdown, repository docs, AGENTS.md, optional lightweight design tokens docs, shell verification commands

---

## File Structure

- `docs/brand/identity-package.md`
  Главный утверждённый identity package.
- `docs/brand/voice-system.md`
  Расширяет voice package в reusable writing rules, patterns, and anti-patterns.
- `docs/brand/visual-system.md`
  Переводит visual direction в UI-level rules, required components, state behavior, and style boundaries.
- `docs/brand/naming-validation.md`
  Выделяет naming checklist в отдельный operational doc для будущих naming sprints.
- `docs/product/landing-message-hierarchy.md`
  Фиксирует message hierarchy для landing, onboarding, and core product framing.
- `docs/product/primary-personas.md`
  Уточняет ICP на уровне одной primary persona и 1-2 future-facing adjacent personas.
- `docs/architecture/frontend-brand-constraints.md`
  Связывает бренд с будущим UI system: no generic SaaS, required signature components, state rules.
- `docs/superpowers/specs/2026-05-01-product-identity-package-design.md`
  Источник согласованных решений, менять только если появляются approval-level corrections.
- `docs/setup/implementation-prompts.md`
  Добавить brand-aware prompts для следующих design/system steps.
- `AGENTS.md`
  Зафиксировать brand/design enforcement rules для будущих агентов.

### Task 1: Extract The Brand Operating Documents

**Files:**
- Create: `docs/brand/voice-system.md`
- Create: `docs/brand/visual-system.md`
- Create: `docs/brand/naming-validation.md`
- Modify: `docs/brand/README.md`
- Test: `docs/brand/voice-system.md`

- [ ] **Step 1: Write the failing test**

```markdown
Expected failure conditions:
- `docs/brand/voice-system.md` does not exist
- `docs/brand/visual-system.md` does not exist
- `docs/brand/naming-validation.md` does not exist
- `docs/brand/README.md` does not link to all three files
```

- [ ] **Step 2: Run test to verify it fails**

Run: `find docs/brand -maxdepth 1 -type f | sort`
Expected: FAIL because only `README.md` and `identity-package.md` exist.

- [ ] **Step 3: Write minimal implementation**

```markdown
# docs/brand/voice-system.md
- Зафиксировать voice principles
- Добавить phrase patterns
- Добавить forbidden copy patterns
- Добавить short UI copy examples

# docs/brand/visual-system.md
- Зафиксировать visual formula
- Описать required signature components
- Описать state requirements
- Описать palette and typography constraints

# docs/brand/naming-validation.md
- Вынести naming checklist
- Добавить scoring table
- Добавить rejection criteria
- Добавить examples of acceptable and unacceptable names
```

- [ ] **Step 4: Run test to verify it passes**

Run: `find docs/brand -maxdepth 1 -type f | sort`
Expected: PASS with `README.md`, `identity-package.md`, `voice-system.md`, `visual-system.md`, and `naming-validation.md`.

- [ ] **Step 5: Commit**

```bash
git add docs/brand/README.md docs/brand/voice-system.md docs/brand/visual-system.md docs/brand/naming-validation.md
git commit -m "docs: expand ProofForge brand operating system"
```

### Task 2: Turn Positioning Into Reusable Product Messaging

**Files:**
- Create: `docs/product/landing-message-hierarchy.md`
- Create: `docs/product/primary-personas.md`
- Modify: `docs/product/README.md`
- Modify: `docs/product/positioning.md`
- Test: `docs/product/landing-message-hierarchy.md`

- [ ] **Step 1: Write the failing test**

```markdown
Expected failure conditions:
- `docs/product/landing-message-hierarchy.md` does not exist
- `docs/product/primary-personas.md` does not exist
- `docs/product/README.md` does not link to them
- positioning does not distinguish primary persona from future expansion personas
```

- [ ] **Step 2: Run test to verify it fails**

Run: `find docs/product -maxdepth 1 -type f | sort`
Expected: FAIL because only `README.md` and `positioning.md` exist.

- [ ] **Step 3: Write minimal implementation**

```markdown
# docs/product/landing-message-hierarchy.md
- Hero message
- Supporting proof-based explanation
- "Why this is not a habit tracker"
- Buddy approval and AI recap explanation
- CTA messaging rules

# docs/product/primary-personas.md
- Primary persona: high-agency side-project builder
- Future persona 1: learning/career accountability user
- Future persona 2: structured peer cohort or intern development system
- In/out of scope notes
```

- [ ] **Step 4: Run test to verify it passes**

Run: `find docs/product -maxdepth 1 -type f | sort`
Expected: PASS with `README.md`, `positioning.md`, `landing-message-hierarchy.md`, and `primary-personas.md`.

- [ ] **Step 5: Commit**

```bash
git add docs/product/README.md docs/product/positioning.md docs/product/landing-message-hierarchy.md docs/product/primary-personas.md
git commit -m "docs: operationalize product messaging and personas"
```

### Task 3: Bind Brand Constraints To Future UI Work

**Files:**
- Create: `docs/architecture/frontend-brand-constraints.md`
- Modify: `AGENTS.md`
- Modify: `docs/architecture/README.md`
- Test: `docs/architecture/frontend-brand-constraints.md`

- [ ] **Step 1: Write the failing test**

```markdown
Expected failure conditions:
- `docs/architecture/frontend-brand-constraints.md` does not exist
- `AGENTS.md` does not tell future agents to preserve the approved brand direction in UI work
- `docs/architecture/README.md` does not reference the constraint doc
```

- [ ] **Step 2: Run test to verify it fails**

Run: `find docs/architecture -maxdepth 1 -type f | sort`
Expected: FAIL because only `README.md` exists.

- [ ] **Step 3: Write minimal implementation**

```markdown
# docs/architecture/frontend-brand-constraints.md
- No generic SaaS purple-gradient dashboard direction
- Required surfaces: Proof Wall, Pact Card, Progress Health, Weekly Poster, Buddy Status
- Every key surface needs loading/empty/error/pending/approved/rejected/success states where relevant
- Product must read as mission control, scoreboard, and proof archive
- Checklist for future dashboard and landing-page work
```

- [ ] **Step 4: Run test to verify it passes**

Run: `find docs/architecture -maxdepth 1 -type f | sort`
Expected: PASS with `README.md` and `frontend-brand-constraints.md`.

- [ ] **Step 5: Commit**

```bash
git add AGENTS.md docs/architecture/README.md docs/architecture/frontend-brand-constraints.md
git commit -m "docs: bind approved brand direction to future UI architecture"
```

### Task 4: Add Brand-Aware Prompts For Future Agents

**Files:**
- Modify: `docs/setup/implementation-prompts.md`
- Create: `docs/setup/brand-foundation-prompts.md`
- Test: `docs/setup/brand-foundation-prompts.md`

- [ ] **Step 1: Write the failing test**

```markdown
Expected failure conditions:
- `docs/setup/brand-foundation-prompts.md` does not exist
- `docs/setup/implementation-prompts.md` does not include brand-aware next steps
```

- [ ] **Step 2: Run test to verify it fails**

Run: `sed -n '1,240p' docs/setup/implementation-prompts.md`
Expected: FAIL because current prompts do not give explicit follow-up brand foundation execution prompts.

- [ ] **Step 3: Write minimal implementation**

```markdown
# docs/setup/brand-foundation-prompts.md
1. Expand voice system into product copy rules
2. Expand visual system into signature components
3. Build naming evaluation rubric
4. Turn landing framing into hero/subhead/CTA system
5. Turn approved direction into dashboard design constraints
```

- [ ] **Step 4: Run test to verify it passes**

Run: `sed -n '1,240p' docs/setup/implementation-prompts.md && sed -n '1,240p' docs/setup/brand-foundation-prompts.md`
Expected: PASS with explicit brand-aware follow-up prompts in both files.

- [ ] **Step 5: Commit**

```bash
git add docs/setup/implementation-prompts.md docs/setup/brand-foundation-prompts.md
git commit -m "docs: add brand foundation execution prompts"
```

### Task 5: Run Documentation Verification And Finalize

**Files:**
- Modify: `docs/brand/README.md`
- Modify: `docs/product/README.md`
- Modify: `docs/architecture/README.md`
- Modify: `docs/setup/README.md`
- Test: `docs/setup/README.md`

- [ ] **Step 1: Write the failing test**

```markdown
Expected failure conditions:
- Section READMEs do not reflect the expanded brand and product document set
- There is no single command sequence for verifying the new docs exist
```

- [ ] **Step 2: Run test to verify it fails**

Run: `find docs/brand docs/product docs/architecture docs/setup -maxdepth 1 -type f | sort`
Expected: FAIL because the READMEs do not yet reflect the full expanded document set.

- [ ] **Step 3: Write minimal implementation**

```markdown
README updates must:
- enumerate the new docs
- explain which doc is canonical for which decision
- include a short verification section in `docs/setup/README.md`
```

- [ ] **Step 4: Run test to verify it passes**

Run: `find docs/brand docs/product docs/architecture docs/setup -maxdepth 1 -type f | sort`
Expected: PASS with the expanded document set present and section READMEs updated.

- [ ] **Step 5: Commit**

```bash
git add docs/brand/README.md docs/product/README.md docs/architecture/README.md docs/setup/README.md
git commit -m "docs: finalize brand foundation documentation map"
```

## Verification Checklist

- Run: `find docs/brand -maxdepth 1 -type f | sort`
  Expected: brand docs include identity package, voice system, visual system, and naming validation.
- Run: `find docs/product -maxdepth 1 -type f | sort`
  Expected: product docs include positioning, message hierarchy, and personas.
- Run: `find docs/architecture -maxdepth 1 -type f | sort`
  Expected: architecture docs include frontend brand constraints.
- Run: `sed -n '1,260p' AGENTS.md`
  Expected: AGENTS enforces Russian docs and future brand preservation.
- Run: `sed -n '1,260p' docs/setup/implementation-prompts.md`
  Expected: follow-up prompts include brand-aware next steps.

## Self-Review

- Spec coverage: the plan covers positioning, ICP operationalization, naming validation, voice system, visual direction, and future UI constraints.
- Placeholder scan: no `TODO`, `TBD`, or “similar to previous task” shortcuts remain.
- Type consistency: document names and required signature components stay consistent with the approved identity package.
