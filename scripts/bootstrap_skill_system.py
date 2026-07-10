#!/usr/bin/env python3
from __future__ import annotations

from collections import OrderedDict
from pathlib import Path
from textwrap import dedent


ROOT = Path(__file__).resolve().parent.parent
PRODUCT_CONTEXT = (
    "ProofForge is a proof-based social accountability platform where users set goals, "
    "invite a buddy, submit proof artifacts, get approval, track progress, and receive AI recaps."
)
DEFAULT_WORKFLOW_RULES = [
    "Use `superpowers:brainstorming` before acting when product direction, UX, or scope is unclear.",
    "Use `superpowers:writing-plans` before implementation starts, even for small slices.",
    "Use `superpowers:test-driven-development` for core logic and `superpowers:systematic-debugging` for defects.",
    "Use `superpowers:requesting-code-review` before claiming completion and `superpowers:finishing-a-development-branch` before release.",
]
SKILL_TREES = [ROOT / ".claude" / "skills", ROOT / ".codex" / "skills"]


SKILL_GROUPS = [
    {
        "group": "Meta",
        "skills": [
            ("project-orchestrator", "Coordinate end-to-end ProofForge delivery across product, design, backend, frontend, AI, QA, and release workstreams.", "orchestrator"),
            ("task-decomposer", "Split large ProofForge initiatives into independently deliverable, testable slices with clear sequencing and ownership.", "decomposer"),
            ("context-loader", "Assemble the smallest complete context package needed for a ProofForge task before execution starts.", "loader"),
            ("definition-of-done-enforcer", "Turn vague completion claims into explicit ProofForge acceptance criteria, verification checks, and release gates.", "enforcer"),
            ("decision-log-writer", "Capture architectural, product, and operational decisions in durable records with rationale and consequences.", "writer"),
            ("agent-handoff-writer", "Prepare precise handoff notes so the next agent or engineer can continue ProofForge work without re-discovery.", "writer"),
        ],
    },
    {
        "group": "Product / Brand / Design",
        "skills": [
            ("startup-positioning", "Define ProofForge market position, ideal users, and differentiation from generic habit trackers.", "strategist"),
            ("name-brand-sprint", "Explore, score, and refine product and campaign naming options that fit ProofForge's accountability premise.", "strategist"),
            ("brand-voice-system", "Build a voice and messaging system for ProofForge that feels sharp, credible, and not self-help generic.", "system"),
            ("visual-identity-director", "Set art direction, typography, color, motion, and visual principles for a distinctive ProofForge identity.", "director"),
            ("design-system-builder", "Define the reusable UI tokens, primitives, patterns, and accessibility rules for ProofForge surfaces.", "builder"),
            ("ux-flow-designer", "Design end-to-end user flows for accountability setup, proof submission, buddy approval, and recap consumption.", "designer"),
            ("landing-page-copywriter", "Write high-conviction landing page copy that explains ProofForge's proof-driven accountability loop.", "copywriter"),
            ("growth-loop-designer", "Design invitation, sharing, and retention loops that fit ProofForge without turning it into spamware.", "designer"),
        ],
    },
    {
        "group": "Architecture",
        "skills": [
            ("system-architect", "Shape the overall ProofForge system architecture across web, API, storage, jobs, AI, and operations.", "architect"),
            ("go-architecture-guardian", "Protect backend boundaries, package discipline, and maintainable Go architecture as the codebase grows.", "guardian"),
            ("domain-modeler", "Model the core entities, invariants, and lifecycle rules behind goals, pacts, evidence, approval, and reporting.", "modeler"),
            ("api-contract-designer", "Design versionable HTTP and webhook contracts with explicit payloads, errors, and auth boundaries.", "designer"),
            ("database-architect", "Design durable relational data structures, indexing strategy, and migration rules for ProofForge workloads.", "architect"),
            ("security-architect", "Design security controls for auth, authorization, secrets, file access, auditability, and abuse resistance.", "architect"),
            ("observability-architect", "Define logs, metrics, traces, alerts, dashboards, and operational signals for ProofForge services.", "architect"),
            ("async-jobs-architect", "Design background job boundaries, retries, idempotency, and failure handling for recap and notification work.", "architect"),
        ],
    },
    {
        "group": "Go Backend Foundation",
        "skills": [
            ("go-project-bootstrapper", "Set up the baseline Go project structure, modules, entrypoints, and conventions for ProofForge services.", "bootstrapper"),
            ("go-config-builder", "Design the environment-driven configuration layer, validation rules, and config loading patterns.", "builder"),
            ("go-logger-builder", "Create a structured logging strategy with request correlation, operational fields, and redaction rules.", "builder"),
            ("go-http-server-builder", "Define the HTTP server skeleton, routing, middleware, timeouts, and transport concerns.", "builder"),
            ("go-db-builder", "Set up database access patterns, migrations, repositories, and transaction boundaries in Go.", "builder"),
            ("go-redis-builder", "Design Redis usage for caching, locks, queues, or ephemeral coordination without hidden coupling.", "builder"),
            ("go-storage-builder", "Design file and object storage handling for proof artifacts with safe uploads and retrieval boundaries.", "builder"),
            ("go-error-handling-builder", "Standardize typed errors, wrapping, user-safe messages, and operational diagnostics.", "builder"),
            ("go-validation-builder", "Create request and domain validation patterns that fail clearly and protect business invariants.", "builder"),
            ("go-test-harness-builder", "Define test helpers, fixtures, mocks, integration harnesses, and repeatable local backend verification.", "builder"),
        ],
    },
    {
        "group": "Backend Feature Slices",
        "skills": [
            ("users-feature-builder", "Design the user account and profile slice with identity boundaries and onboarding needs.", "builder"),
            ("goals-feature-builder", "Design the goal creation, editing, status, and progress slice around proof-backed accountability.", "builder"),
            ("pacts-feature-builder", "Design the buddy pact model, roles, expectations, and lifecycle transitions.", "builder"),
            ("invites-feature-builder", "Design invitation issuance, acceptance, expiration, and abuse-resistant join flows.", "builder"),
            ("checkins-feature-builder", "Design periodic check-in flows that anchor proof collection and progress continuity.", "builder"),
            ("evidence-feature-builder", "Design evidence submission, storage, metadata, and reviewability for proof artifacts.", "builder"),
            ("approval-feature-builder", "Design buddy approval and rejection flows with auditable state changes and notification hooks.", "builder"),
            ("streak-feature-builder", "Design streak logic that reflects verified proof behavior instead of shallow daily taps.", "builder"),
            ("reports-feature-builder", "Design recap and reporting outputs spanning goals, compliance, approvals, and AI summaries.", "builder"),
            ("notifications-feature-builder", "Design event-driven notifications across app, email, or Telegram touchpoints.", "builder"),
            ("admin-feature-builder", "Design the minimal operational admin slice for support, moderation, and system control.", "builder"),
        ],
    },
    {
        "group": "Frontend",
        "skills": [
            ("web-app-bootstrapper", "Set up the frontend application structure, tooling, shell layout, and cross-cutting UX conventions.", "bootstrapper"),
            ("frontend-architecture-guardian", "Protect clear frontend boundaries, state discipline, data flow, and maintainable component layering.", "guardian"),
            ("ui-kit-builder", "Define the reusable component kit, tokens, states, and accessibility rules for the web app.", "builder"),
            ("landing-page-builder", "Design and build the ProofForge marketing landing surface with a distinctive branded experience.", "builder"),
            ("auth-ui-builder", "Design the authentication and invite entry experience with clear trust and accountability cues.", "builder"),
            ("dashboard-builder", "Design the main dashboard showing goals, proof cadence, buddy state, and recap visibility.", "builder"),
            ("goal-ui-builder", "Design the goal management surface for setup, editing, status, and proof expectations.", "builder"),
            ("pact-invite-ui-builder", "Design the pact invitation and acceptance UI flow across web and buddy-facing touchpoints.", "builder"),
            ("checkin-ui-builder", "Design the proof submission and check-in experience for fast, high-confidence evidence capture.", "builder"),
            ("approval-ui-builder", "Design buddy approval and rejection UI with context-rich evidence review and safe actions.", "builder"),
            ("reports-ui-builder", "Design weekly recap, progress reporting, and history views with clear signal and shareability.", "builder"),
            ("share-card-ui-builder", "Design branded progress share cards that reinforce ProofForge identity without feeling cheesy.", "builder"),
            ("admin-ui-builder", "Design the admin interface for support, oversight, moderation, and operational triage.", "builder"),
        ],
    },
    {
        "group": "Telegram / AI",
        "skills": [
            ("telegram-bot-architect", "Design the Telegram bot's role, security model, capabilities, and integration boundaries.", "architect"),
            ("telegram-webhook-builder", "Design webhook handling, verification, retries, and message routing for Telegram events.", "builder"),
            ("telegram-checkin-flow-builder", "Design a Telegram-native check-in flow for submitting proof and maintaining accountability momentum.", "builder"),
            ("telegram-approval-flow-builder", "Design a Telegram-native approval flow for buddies reviewing and responding to evidence.", "builder"),
            ("ai-recap-prompt-engineer", "Create the recap prompt strategy, guardrails, and evaluation criteria for weekly AI summaries.", "engineer"),
            ("ai-summary-service-builder", "Design the service that assembles context, calls models, stores outputs, and handles failures.", "builder"),
            ("ai-content-card-builder", "Design reusable AI-generated content card formats for web, Telegram, and shareable recap surfaces.", "builder"),
            ("notification-scheduler-builder", "Design the timing, batching, and policy rules behind reminders and recap dispatches.", "builder"),
            ("weekly-report-worker-builder", "Design the background worker responsible for compiling and delivering weekly recap outputs.", "builder"),
        ],
    },
    {
        "group": "QA / Security",
        "skills": [
            ("go-unit-test-builder", "Design precise unit tests for Go domain logic, services, and invariants.", "builder"),
            ("go-integration-test-builder", "Design Go integration tests for persistence, HTTP contracts, jobs, and external boundaries.", "builder"),
            ("frontend-test-builder", "Design component, interaction, and regression tests for the frontend surface.", "builder"),
            ("e2e-test-builder", "Design realistic end-to-end tests covering the primary accountability loop across the stack.", "builder"),
            ("api-contract-test-builder", "Design contract verification for HTTP APIs, callbacks, and integration-facing payloads.", "builder"),
            ("security-reviewer", "Review changes for auth, authorization, data exposure, file safety, abuse vectors, and secret handling.", "reviewer"),
            ("privacy-reviewer", "Review data collection, retention, consent, deletion, and AI usage against a privacy-first MVP bar.", "reviewer"),
            ("performance-smoke-tester", "Run fast performance smoke checks for key flows, queries, and background processing paths.", "tester"),
            ("bug-reproducer", "Reduce reported issues to deterministic steps, observable evidence, and likely fault boundaries.", "reproducer"),
            ("quality-gate-runner", "Execute the final automated and manual quality gates before merge or release.", "runner"),
            ("integration-quality-gate", "Run a full ProofForge quality gate for a feature slice: scenario matrix, backend integration coverage, frontend e2e coverage, deploy smoke validation, and a final release verdict with blockers and gaps called out explicitly.", "runner"),
        ],
    },
    {
        "group": "DevOps",
        "skills": [
            ("docker-compose-builder", "Design local multi-service composition for the ProofForge stack and developer workflows.", "builder"),
            ("nginx-deploy-builder", "Design the reverse proxy, TLS, routing, headers, and static asset concerns for deployment.", "builder"),
            ("vps-deploy-runbook-builder", "Write the VPS deployment runbook covering provisioning, rollout, rollback, and smoke validation.", "builder"),
            ("ci-cd-builder", "Design continuous integration and delivery pipelines with tests, artifacts, and release controls.", "builder"),
            ("migration-runner-builder", "Design migration execution, locking, rollback policy, and operator safety checks.", "builder"),
            ("telegram-webhook-deploy-builder", "Design deployment and runtime handling for Telegram webhook exposure and rotation.", "builder"),
            ("observability-builder", "Build the practical observability stack wiring for logs, metrics, traces, and alerts.", "builder"),
            ("grafana-dashboard-builder", "Design dashboards that make product health, worker throughput, and failures obvious.", "builder"),
            ("backup-restore-builder", "Design backup and restore procedures for databases, files, and critical configuration.", "builder"),
            ("rollback-builder", "Design fast rollback procedures for app, database, infra, and webhook changes.", "builder"),
        ],
    },
    {
        "group": "Release",
        "skills": [
            ("release-manager", "Coordinate release scope, sequencing, approvals, comms, and go/no-go decisions.", "manager"),
            ("smoke-test-release", "Run post-deploy smoke coverage for the highest-risk ProofForge user and operator journeys.", "runner"),
            ("product-readiness-reviewer", "Review whether the release is coherent, safe, branded, and worthy of real-user exposure.", "reviewer"),
            ("docs-finalizer", "Bring specs, runbooks, changelogs, and operator notes to a release-ready state.", "finalizer"),
            ("analytics-instrumentation-builder", "Design the event instrumentation plan needed to measure adoption and accountability behavior.", "builder"),
            ("launch-checklist-runner", "Execute the final launch checklist covering product, operations, analytics, and support readiness.", "runner"),
        ],
    },
]


GROUP_RULES = {
    "Meta": [
        "Bias toward coordination artifacts that unblock multiple workers without hiding ownership.",
    ],
    "Product / Brand / Design": [
        "Keep the brand sharp, specific, and distinct from generic wellness or habit language.",
        "Decisions must strengthen trust, proof, and accountability rather than gamified fluff.",
    ],
    "Architecture": [
        "Prefer explicit boundaries, boring reliability, and maintainable Go services over clever coupling.",
    ],
    "Go Backend Foundation": [
        "Design for clear interfaces, testability, and operational visibility from the start.",
    ],
    "Backend Feature Slices": [
        "Every slice must preserve domain invariants around proof, buddy approval, and progress history.",
    ],
    "Frontend": [
        "Preserve a distinctive, high-quality visual identity while keeping states and data flow legible.",
    ],
    "Telegram / AI": [
        "Model safety, abuse resistance, and graceful degradation before adding convenience.",
    ],
    "QA / Security": [
        "Favor evidence, reproducibility, and explicit exit criteria over vague quality claims.",
    ],
    "DevOps": [
        "Optimize for repeatable operations, rollback safety, and observable failure modes.",
    ],
    "Release": [
        "Do not trade away product coherence or operational readiness for speed.",
    ],
}


ARCHETYPE_HINTS = {
    "orchestrator": {
        "outputs": [
            "An execution brief with scope, dependencies, and immediate next actions.",
            "A dependency map with blockers, parallel tracks, and handoff points.",
            "Verification and release checkpoints tied to the requested milestone.",
        ],
        "workflow": [
            "Load the current milestone, affected workstreams, and open decisions.",
            "Clarify the narrowest useful outcome and the constraints around time, quality, and staffing.",
            "Sequence the work into explicit tracks with owners, dependencies, and review gates.",
            "Surface the risks, escalation points, and verification steps that must not be skipped.",
            "Publish the orchestration brief in a format the next worker can execute immediately.",
        ],
    },
    "decomposer": {
        "outputs": [
            "A breakdown of the request into small, testable, independently reviewable slices.",
            "Recommended execution order and a map of what can happen in parallel.",
            "Clear acceptance criteria for each slice.",
        ],
        "workflow": [
            "Read the request and identify the real unit of value being delivered.",
            "Separate foundation, feature, QA, and release concerns instead of mixing them.",
            "Break each concern into slices with one owner and one clear exit condition.",
            "Mark the blockers, prerequisites, and safe parallelization boundaries.",
            "Return a sequence that minimizes risk and rework.",
        ],
    },
    "loader": {
        "outputs": [
            "A minimal context pack with the exact files, docs, decisions, and unknowns needed next.",
            "A short summary of what matters, what does not, and what still needs confirmation.",
            "A handoff-ready list of references to load before implementation or review.",
        ],
        "workflow": [
            "Identify the task, the affected subsystem, and the likely adjacent concerns.",
            "Pull only the files, docs, and prior decisions that materially affect execution.",
            "Summarize the relevant context with links or paths and explicit unknowns.",
            "Flag stale assumptions, missing documents, or conflicting guidance.",
            "Deliver a context pack sized for action, not for archival completeness.",
        ],
    },
    "enforcer": {
        "outputs": [
            "A definition of done checklist with objective pass conditions.",
            "Required verification commands, reviews, and operator checks.",
            "A list of missing evidence when a task is not actually complete.",
        ],
        "workflow": [
            "Translate the request into observable outcomes instead of vague success language.",
            "Define the tests, reviews, docs, and smoke checks needed to trust the result.",
            "Compare the current state against the checklist and mark any gap explicitly.",
            "Reject ambiguous completion claims until evidence exists.",
            "Publish the final gate list with no hidden assumptions.",
        ],
    },
    "writer": {
        "outputs": [
            "A concise written artifact with the decision, rationale, impact, and next actions.",
            "Explicit links to affected files, plans, or operational procedures.",
            "A record another engineer can trust without re-interviewing the original author.",
        ],
        "workflow": [
            "Gather the facts, the decision point, and the relevant constraints.",
            "Write the essential context, the choice made, and the trade-offs accepted.",
            "Call out the downstream consequences and what must happen next.",
            "Remove filler so the artifact can be consumed quickly under pressure.",
            "Store the note where future work will find it.",
        ],
    },
    "strategist": {
        "outputs": [
            "A strategy brief with positioning choices, target audience assumptions, and differentiators.",
            "A small set of evaluated options with a recommended direction.",
            "Messaging or brand constraints the rest of the system should honor.",
        ],
        "workflow": [
            "Clarify the target audience, competing alternatives, and the outcome this strategy must unlock.",
            "Generate a small set of options that are meaningfully different from one another.",
            "Score the options against distinctiveness, clarity, and ProofForge fit.",
            "Recommend one direction with reasoning, trade-offs, and constraints.",
            "Capture the resulting guidance so design and product work can reuse it consistently.",
        ],
    },
    "system": {
        "outputs": [
            "A reusable system definition with principles, examples, and boundaries.",
            "Canonical rules other teams can apply without reinterpretation.",
            "A shortlist of exceptions or edge cases that need deliberate handling.",
        ],
        "workflow": [
            "Define the few principles that must stay stable across the product.",
            "Translate principles into reusable patterns, examples, and anti-patterns.",
            "Test the system against core ProofForge scenarios for coherence.",
            "Trim anything decorative that does not improve decision quality.",
            "Publish the system in a format that can guide future contributors.",
        ],
    },
    "director": {
        "outputs": [
            "An art direction brief covering color, typography, layout, motion, and imagery.",
            "Concrete do and do-not guidance for future design work.",
            "A shortlist of references or motifs that reinforce ProofForge identity.",
        ],
        "workflow": [
            "Start from the brand promise and the emotional tone the interface should create.",
            "Choose a clear visual direction rather than a safe average SaaS look.",
            "Define the primary ingredients that make the direction repeatable.",
            "Pressure-test the direction across landing, product, and shareable surfaces.",
            "Document guardrails so later builders preserve coherence.",
        ],
    },
    "builder": {
        "outputs": [
            "A concrete build brief with target files, interfaces, tests, and rollout notes.",
            "Required validation, observability, and failure-handling expectations.",
            "A minimal implementation path that fits the surrounding architecture.",
        ],
        "workflow": [
            "Load the affected context and identify the interfaces this work touches.",
            "Define the smallest implementation slice that delivers usable value.",
            "Specify the files, data flow, state transitions, and validation rules involved.",
            "List the tests, smoke checks, and operator signals required for safe delivery.",
            "Return an implementation-ready brief that another engineer can execute cleanly.",
        ],
    },
    "designer": {
        "outputs": [
            "A flow or interaction design with states, transitions, and edge cases.",
            "A recommendation grounded in clarity, trust, and ProofForge-specific behavior.",
            "A concise set of UI copy or structure notes where they matter.",
        ],
        "workflow": [
            "Identify the user intent, the key decision points, and the failure cases.",
            "Map the normal flow first, then layer in empty, loading, and error states.",
            "Compare a small number of approaches and choose the clearest one.",
            "Document the chosen interaction with enough detail to build or test it.",
            "Check that the flow reinforces proof, approval, and progress visibility.",
        ],
    },
    "copywriter": {
        "outputs": [
            "Audience-specific copy options with a recommended version.",
            "Message hierarchy and proof points that support conversion.",
            "Tone guidance to keep future copy aligned.",
        ],
        "workflow": [
            "Clarify the audience, channel, and action the copy needs to drive.",
            "Write several message angles rather than polishing the first obvious one.",
            "Favor concrete, proof-driven language over startup fluff.",
            "Select the strongest version and tighten it for clarity and pace.",
            "Document tone and message constraints for follow-on edits.",
        ],
    },
    "architect": {
        "outputs": [
            "An architecture brief with components, interfaces, data flow, and constraints.",
            "Explicit trade-offs, failure modes, and operational considerations.",
            "A list of decisions that should become ADRs, plans, or implementation tasks.",
        ],
        "workflow": [
            "Define the problem boundary and the non-negotiable constraints first.",
            "Sketch the component model, ownership lines, and interface contracts.",
            "Work through failure modes, data consistency, and operational behavior.",
            "Choose the simplest architecture that preserves future maintainability.",
            "Record trade-offs and the follow-up work required to implement safely.",
        ],
    },
    "guardian": {
        "outputs": [
            "A review of the proposed approach against architecture or layering rules.",
            "Concrete corrections for boundary leaks, coupling, naming, or state drift.",
            "A keep-or-fix recommendation with explicit rationale.",
        ],
        "workflow": [
            "Load the local conventions, existing boundaries, and the proposed change.",
            "Look for coupling, leakage, duplication, and future maintenance traps.",
            "Separate hard blockers from optional polish.",
            "State the correction in operational terms the implementer can act on.",
            "Preserve good patterns instead of forcing unnecessary rewrites.",
        ],
    },
    "modeler": {
        "outputs": [
            "A domain model covering entities, relationships, invariants, and lifecycle transitions.",
            "A glossary of terms that removes ambiguity across product and engineering.",
            "Boundary notes describing what belongs in the model and what stays outside it.",
        ],
        "workflow": [
            "List the core nouns, states, and rules implied by the feature or product area.",
            "Separate domain invariants from transport or storage concerns.",
            "Model transitions, permissions, and historical record needs explicitly.",
            "Rename fuzzy concepts until the model becomes unambiguous.",
            "Publish the resulting glossary and invariants for downstream use.",
        ],
    },
    "bootstrapper": {
        "outputs": [
            "A scaffold plan with directories, foundational files, conventions, and setup steps.",
            "The minimal local tooling and verification path needed for the scaffold.",
            "Guardrails that prevent the scaffold from turning into accidental production code.",
        ],
        "workflow": [
            "Define the scope of the scaffold and what intentionally remains unimplemented.",
            "Lay out the top-level directories, conventions, and integration points.",
            "Specify setup steps, placeholders, and developer ergonomics.",
            "Add verification steps so the scaffold stays coherent over time.",
            "Document the next implementation layers rather than pre-building them.",
        ],
    },
    "engineer": {
        "outputs": [
            "A prompt or model interaction strategy with structured inputs, outputs, and guardrails.",
            "Evaluation criteria and failure cases for the AI behavior.",
            "A change log of assumptions or model-specific constraints.",
        ],
        "workflow": [
            "Clarify the task the model must perform and what evidence it receives.",
            "Design the prompt structure, output shape, and refusal boundaries.",
            "Define evaluation cases, bad outputs, and fallback behavior.",
            "Keep the prompt short enough to be maintainable but specific enough to be reliable.",
            "Record the assumptions so future tuning is deliberate, not accidental.",
        ],
    },
    "reviewer": {
        "outputs": [
            "A findings-first review with severity, rationale, and affected areas.",
            "Residual risks, missing tests, and follow-up recommendations.",
            "A clear pass, conditional pass, or fail recommendation.",
        ],
        "workflow": [
            "Load the scope, the changed area, and the relevant expectations or policy.",
            "Look for the highest-risk failure modes first.",
            "State findings with evidence and concrete impact, not vague opinions.",
            "Call out missing proof where confidence is limited.",
            "Summarize the release implication after the findings are clear.",
        ],
    },
    "tester": {
        "outputs": [
            "A smoke test plan with fast checks, thresholds, and expected evidence.",
            "A record of failures, bottlenecks, or capacity concerns.",
            "Recommendations for deeper investigation when smoke coverage is not enough.",
        ],
        "workflow": [
            "Choose the small set of paths most likely to reveal real performance or reliability problems.",
            "Define the commands, load shape, and timing thresholds up front.",
            "Run the checks and capture evidence instead of impressions.",
            "Separate actual bottlenecks from noisy local variance.",
            "Report the result with clear next actions.",
        ],
    },
    "reproducer": {
        "outputs": [
            "A deterministic reproduction recipe with preconditions, steps, and observed output.",
            "A narrowed fault boundary describing where the issue likely lives.",
            "Any artifacts, logs, or test cases that make the bug easier to fix.",
        ],
        "workflow": [
            "Collect the report, environment details, and expected behavior.",
            "Reduce the issue to the fewest possible steps that still trigger it.",
            "Capture the exact observed result, including logs or screenshots if needed.",
            "Vary one condition at a time to narrow the fault boundary.",
            "Package the reproduction so another engineer can run it quickly.",
        ],
    },
    "runner": {
        "outputs": [
            "An execution checklist with results captured per step.",
            "A list of blockers or skipped steps requiring follow-up.",
            "A final pass or fail summary tied to evidence.",
        ],
        "workflow": [
            "Load the checklist, required environment, and success thresholds.",
            "Run steps in the intended order and capture the result of each one.",
            "Stop on hard blockers instead of papering over them.",
            "Summarize failures with exact next actions.",
            "Publish a final status that another operator can audit.",
        ],
    },
    "manager": {
        "outputs": [
            "A release plan with scope, sequencing, owners, approval points, and fallback paths.",
            "Go or no-go criteria and the evidence required to make that call.",
            "Communication notes for engineering, support, and stakeholders.",
        ],
        "workflow": [
            "Confirm the release scope, the constraints, and the blast radius.",
            "Sequence technical, operational, and communication tasks explicitly.",
            "Define the go or no-go criteria before execution begins.",
            "Make rollback, smoke, and monitoring steps unavoidable.",
            "Publish the release brief with owners and checkpoints.",
        ],
    },
    "finalizer": {
        "outputs": [
            "A final documentation pass that is complete, consistent, and release-usable.",
            "A short list of docs still missing if the release is not actually ready.",
            "Tightened wording, links, and structure that reduce operator confusion.",
        ],
        "workflow": [
            "Load the release scope and the documents users or operators will need.",
            "Check for missing sections, stale steps, broken links, and contradictions.",
            "Rewrite for speed and clarity rather than completeness theater.",
            "Align the docs with the actual shipped behavior and operations model.",
            "Publish the final doc status and any remaining gaps.",
        ],
    },
}


def sentence_case(text: str) -> str:
    return text[0].lower() + text[1:] if text else text


def markdown_list(items: list[str]) -> str:
    return "\n".join(f"- {item}" for item in items)


def render_skill(group: str, slug: str, description: str, archetype: str) -> str:
    hints = ARCHETYPE_HINTS[archetype]
    purpose = [
        f"Use this skill to {sentence_case(description)}",
        "Keep the output aligned with ProofForge's proof-driven accountability loop instead of generic habit-tracking patterns.",
        "Favor maintainable architecture, explicit trade-offs, and concrete next actions over hand-wavy advice.",
    ]
    inputs = [
        "The current request, milestone, or problem statement.",
        "Relevant repository context: files, docs, decisions, and active constraints.",
        "Quality, release, and operational expectations that affect the work.",
    ]
    outputs = hints["outputs"]
    rules = DEFAULT_WORKFLOW_RULES + GROUP_RULES[group] + [
        "Prefer explicit file paths, interfaces, and verification commands whenever implementation guidance is part of the answer.",
        "Do not drift into adjacent workstreams unless the handoff or dependency is part of the output.",
    ]
    workflow = hints["workflow"]
    dod = [
        "The output is specific enough for the next worker to act without re-discovering the basics.",
        "ProofForge-specific constraints, risks, and quality gates are explicit.",
        "Open questions and assumptions are named instead of hidden.",
    ]
    forbidden = [
        "Treating ProofForge like a generic habit tracker, social feed, or gamified streak toy.",
        "Using vague language such as 'handle later', 'should be fine', or 'add tests' without specifics.",
        "Ignoring verification, release, operational, or security concerns that materially affect the result.",
    ]
    lines = [
        "---",
        f"name: {slug}",
        f"description: {description}",
        "---",
        "",
        "## Purpose",
        markdown_list(purpose),
        "",
        "## Inputs",
        markdown_list(inputs),
        "",
        "## Outputs",
        markdown_list(outputs),
        "",
        "## Rules",
        markdown_list(rules),
        "",
        "## Workflow",
        "\n".join(f"{idx}. {step}" for idx, step in enumerate(workflow, start=1)),
        "",
        "## Definition of Done",
        markdown_list(dod),
        "",
        "## Forbidden",
        markdown_list(forbidden),
        "",
    ]
    return "\n".join(lines)


def ensure_text(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


def build_readme(title: str, body: str) -> str:
    return dedent(
        f"""\
        # {title}

        {body}
        """
    )


def create_top_level_files() -> None:
    files: dict[Path, str] = {
        ROOT / "README.md": build_readme(
            "ProofForge Development System",
            dedent(
                """\
                This repository is scaffolded as a development system for ProofForge, not as the product implementation itself.

                Current scope:
                - mirrored local skill systems for Claude and Codex
                - project-level operating instructions
                - docs skeleton for product, brand, architecture, QA, release, and runbooks
                - repo skeleton for backend, web, infra, scripts, and tests
                - bootstrap and verification scripts for the skill system

                Primary references:
                - `AGENTS.md`
                - `CLAUDE.md`
                - `docs/setup/recommended-order.md`
                - `docs/setup/implementation-prompts.md`
                - `docs/setup/setup-commands.md`
                - `docs/setup/verification-commands.md`
                """
            ).strip(),
        ),
        ROOT / "AGENTS.md": dedent(
            """\
            # ProofForge Agent Operating Guide

            ## Product context
            ProofForge is a proof-based social accountability platform. Users set goals, invite a buddy, submit proof artifacts, receive buddy approval, track progress, and get AI weekly recaps.

            ## Non-negotiables
            - Do not treat the product like a generic habit tracker.
            - Preserve a distinctive brand and visual identity.
            - Prefer maintainable Go backend architecture over framework sprawl.
            - Frontend quality matters: interaction design, accessibility, and visual coherence are first-class concerns.
            - Every meaningful implementation path must account for tests, deployment, observability, and smoke checks.

            ## Skill system
            - Project skills live in both `.claude/skills/` and `.codex/skills/`.
            - Use the mirrored project skills before doing domain-specific work when one clearly applies.
            - Default workflow:
              - `superpowers:brainstorming` before unclear product, UX, or architecture work
              - `superpowers:writing-plans` before implementation
              - `superpowers:test-driven-development` for core logic
              - `superpowers:systematic-debugging` for bugs
              - `superpowers:requesting-code-review` before completion
              - `superpowers:finishing-a-development-branch` before release

            ## Working style
            - Start by loading only the context needed for the current task.
            - Write explicit decisions to `docs/decisions/`.
            - Keep specs in `docs/superpowers/specs/` and plans in `docs/superpowers/plans/`.
            - Prefer small, testable vertical slices over broad partially-finished layers.
            """
        ),
        ROOT / "CLAUDE.md": dedent(
            """\
            # ProofForge Claude Instructions

            Use the local skill system in `.claude/skills/` before domain work whenever a matching skill exists. Mirror any durable workflow guidance into `.codex/skills/` so both agent surfaces stay aligned.

            Default execution policy:
            - clarify the product distinction first: proof, buddy approval, and recaps are core
            - brainstorm before unclear design work
            - write a plan before implementation
            - use TDD for domain logic
            - use systematic debugging for defects
            - request review before calling work complete
            - finish the branch deliberately before release

            Repository intent:
            - this scaffold is the development system and operating model
            - product code should be added later through planned implementation slices
            """
        ),
        ROOT / ".env.example": dedent(
            """\
            APP_ENV=development
            APP_NAME=proofforge
            APP_HOST=0.0.0.0
            APP_PORT=8080
            WEB_ORIGIN=http://localhost:3000

            DATABASE_URL=postgres://proofforge:proofforge@localhost:5432/proofforge?sslmode=disable
            REDIS_URL=redis://localhost:6379/0

            STORAGE_PROVIDER=s3
            S3_BUCKET=proofforge-dev
            S3_ENDPOINT=http://localhost:9000
            S3_REGION=us-east-1
            S3_ACCESS_KEY_ID=change-me
            S3_SECRET_ACCESS_KEY=change-me

            TELEGRAM_BOT_TOKEN=change-me
            TELEGRAM_WEBHOOK_BASE_URL=https://example.com
            TELEGRAM_WEBHOOK_SECRET=change-me

            OPENAI_API_KEY=change-me
            AI_RECAP_MODEL=gpt-5.4

            SMTP_HOST=localhost
            SMTP_PORT=1025
            SMTP_USERNAME=
            SMTP_PASSWORD=
            SMTP_FROM=noreply@example.com

            LOG_LEVEL=debug
            OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
            GRAFANA_URL=http://localhost:3001

            ANALYTICS_PROVIDER=posthog
            ANALYTICS_API_KEY=change-me
            """
        ),
        ROOT / ".gitignore": dedent(
            """\
            .DS_Store
            .env
            .env.local
            .env.*.local
            !.env.example
            node_modules/
            dist/
            build/
            coverage/
            tmp/
            .tmp/
            .cache/
            .pytest_cache/
            .mypy_cache/
            .ruff_cache/
            __pycache__/
            *.pyc
            vendor/
            bin/
            out/
            """
        ),
        ROOT / "Makefile": dedent(
            """\
            .PHONY: help bootstrap verify verify-skills list-skills docs tree

            help:
            \t@printf "Available targets:\\n"
            \t@printf "  bootstrap      Regenerate mirrored skills and scaffold inventories\\n"
            \t@printf "  verify         Verify the skill system and scaffold structure\\n"
            \t@printf "  verify-skills  Alias for verify\\n"
            \t@printf "  list-skills    Print the expected skill inventory\\n"
            \t@printf "  docs           List scaffolded docs\\n"
            \t@printf "  tree           List scaffolded repository paths\\n"

            bootstrap:
            \tpython3 scripts/bootstrap_skill_system.py

            verify: verify-skills

            verify-skills:
            \tpython3 scripts/verify_skill_system.py

            list-skills:
            \tpython3 scripts/verify_skill_system.py --list

            docs:
            \tfind docs -maxdepth 3 -type f | sort

            tree:
            \tfind . -maxdepth 3 \\( -path './.git' -o -path './__pycache__' \\) -prune -o -type f -print | sort
            """
        ),
    }
    for path, content in files.items():
        ensure_text(path, content.rstrip() + "\n")


def create_docs_and_repo_skeleton() -> None:
    doc_files = {
        ROOT / ".claude" / "skills" / "README.md": build_readme(
            "Claude Skills",
            "ProofForge-specific skills for Claude-compatible agents. Each skill folder contains a `SKILL.md` file with the shared local format.",
        ),
        ROOT / ".codex" / "skills" / "README.md": build_readme(
            "Codex Skills",
            "ProofForge-specific skills for Codex-compatible agents. This tree mirrors `.claude/skills/` one-for-one.",
        ),
        ROOT / "backend" / "README.md": build_readme(
            "Backend Skeleton",
            "Reserved for the maintainable Go backend. Add service code only through planned slices using the architecture and backend foundation skills.",
        ),
        ROOT / "backend" / "cmd" / "README.md": build_readme(
            "Backend Entrypoints",
            "Placeholder for future Go entrypoints such as API, workers, or migration commands.",
        ),
        ROOT / "backend" / "internal" / "README.md": build_readme(
            "Backend Internal",
            "Placeholder for domain, application, transport, and platform packages once implementation begins.",
        ),
        ROOT / "backend" / "pkg" / "README.md": build_readme(
            "Backend Shared Packages",
            "Placeholder for carefully-scoped shared packages that are safe to import across services.",
        ),
        ROOT / "web" / "README.md": build_readme(
            "Frontend Skeleton",
            "Reserved for the high-quality web app and marketing site. Add implementation only through the planned frontend slices.",
        ),
        ROOT / "web" / "app" / "README.md": build_readme(
            "Web App",
            "Placeholder for routes, layouts, and page-level application surfaces.",
        ),
        ROOT / "web" / "components" / "README.md": build_readme(
            "Web Components",
            "Placeholder for reusable components and primitives backed by the local design system.",
        ),
        ROOT / "web" / "lib" / "README.md": build_readme(
            "Web Libraries",
            "Placeholder for data clients, helpers, and frontend platform code.",
        ),
        ROOT / "infra" / "README.md": build_readme(
            "Infrastructure Skeleton",
            "Reserved for Docker, reverse proxy, CI, and deployment artifacts.",
        ),
        ROOT / "infra" / "docker" / "README.md": build_readme(
            "Docker",
            "Placeholder for local composition and image-level infrastructure files.",
        ),
        ROOT / "infra" / "nginx" / "README.md": build_readme(
            "Nginx",
            "Placeholder for reverse proxy, TLS, and deployment-facing web server configuration.",
        ),
        ROOT / "infra" / "ci" / "README.md": build_readme(
            "CI",
            "Placeholder for CI and CD workflow definitions and supporting scripts.",
        ),
        ROOT / "scripts" / "README.md": build_readme(
            "Scripts",
            "Operational scripts for bootstrapping and verifying the repository scaffold live here.",
        ),
        ROOT / "tests" / "README.md": build_readme(
            "Test Skeleton",
            "Reserved for future unit, integration, frontend, and end-to-end test suites.",
        ),
        ROOT / "docs" / "README.md": build_readme(
            "Docs Skeleton",
            "Documentation is organized by product, brand, architecture, API, QA, runbooks, release, decisions, and setup.",
        ),
        ROOT / "docs" / "product" / "README.md": build_readme(
            "Product Docs",
            "Place product specs, user problems, milestones, and roadmap notes here.",
        ),
        ROOT / "docs" / "brand" / "README.md": build_readme(
            "Brand Docs",
            "Place positioning, naming work, voice rules, visual direction, and copy systems here.",
        ),
        ROOT / "docs" / "architecture" / "README.md": build_readme(
            "Architecture Docs",
            "Place architecture overviews, domain models, data designs, and system diagrams here.",
        ),
        ROOT / "docs" / "api" / "README.md": build_readme(
            "API Docs",
            "Place contract references, auth rules, webhook payload docs, and integration notes here.",
        ),
        ROOT / "docs" / "qa" / "README.md": build_readme(
            "QA Docs",
            "Place test strategy, quality gates, defect reports, and performance notes here.",
        ),
        ROOT / "docs" / "runbooks" / "README.md": build_readme(
            "Runbooks",
            "Place deployment, rollback, backup, restore, incident, and operator procedures here.",
        ),
        ROOT / "docs" / "release" / "README.md": build_readme(
            "Release Docs",
            "Place release notes, launch checklists, readiness reviews, and rollout plans here.",
        ),
        ROOT / "docs" / "decisions" / "README.md": build_readme(
            "Decision Log",
            "Store ADR-style records here using a stable date-and-slug naming convention.",
        ),
        ROOT / "docs" / "setup" / "README.md": build_readme(
            "Setup Docs",
            "This folder contains the generated inventories, setup commands, prompts, and execution order for the scaffold.",
        ),
        ROOT / "docs" / "superpowers" / "README.md": build_readme(
            "Superpowers Workflow",
            "Use this area for brainstorm specs, implementation plans, and code review notes produced via the Superpowers workflow.",
        ),
        ROOT / "docs" / "superpowers" / "specs" / "README.md": build_readme(
            "Specs",
            "Store approved design specs here before implementation work starts.",
        ),
        ROOT / "docs" / "superpowers" / "plans" / "README.md": build_readme(
            "Plans",
            "Store detailed implementation plans here after specs are approved.",
        ),
        ROOT / "docs" / "superpowers" / "reviews" / "README.md": build_readme(
            "Reviews",
            "Store review notes, launch reviews, and quality gate evidence here.",
        ),
        ROOT / "docs" / "setup" / "recommended-order.md": dedent(
            """\
            # Recommended Order Of Execution

            1. `project-orchestrator`
            2. `context-loader`
            3. `startup-positioning`
            4. `brand-voice-system`
            5. `visual-identity-director`
            6. `system-architect`
            7. `domain-modeler`
            8. `api-contract-designer`
            9. `database-architect`
            10. `go-project-bootstrapper`
            11. `web-app-bootstrapper`
            12. `docker-compose-builder`
            13. `ci-cd-builder`
            14. `analytics-instrumentation-builder`
            15. `quality-gate-runner`

            Use the more specialized skills after these foundations exist.
            """
        ),
        ROOT / "docs" / "setup" / "implementation-prompts.md": dedent(
            """\
            # First 10 Implementation Prompts

            1. Use `startup-positioning` to define the ICP, core pain, and ProofForge differentiation.
            2. Use `brand-voice-system` and `visual-identity-director` to create the first brand system draft.
            3. Use `system-architect` to propose the first full-stack architecture for the MVP.
            4. Use `domain-modeler` to define goals, pacts, evidence, approvals, streaks, and reports.
            5. Use `api-contract-designer` to draft the initial HTTP API and Telegram webhook contracts.
            6. Use `database-architect` to design the relational schema and migration approach.
            7. Use `go-project-bootstrapper` to scaffold the Go backend around the approved architecture.
            8. Use `web-app-bootstrapper` and `ui-kit-builder` to scaffold the frontend shell and design primitives.
            9. Use `telegram-bot-architect` and `telegram-webhook-builder` to design the bot interaction boundary.
            10. Use `docker-compose-builder` and `ci-cd-builder` to wire local and CI execution paths.
            """
        ),
        ROOT / "docs" / "setup" / "setup-commands.md": dedent(
            """\
            # Setup Commands

            ```bash
            git init
            git branch -m main
            python3 scripts/bootstrap_skill_system.py
            python3 scripts/verify_skill_system.py
            make verify
            ```
            """
        ),
        ROOT / "docs" / "setup" / "verification-commands.md": dedent(
            """\
            # Verification Commands

            ```bash
            python3 scripts/verify_skill_system.py
            make verify
            find .claude/skills -mindepth 1 -maxdepth 1 -type d | wc -l
            find .codex/skills -mindepth 1 -maxdepth 1 -type d | wc -l
            rg --files .claude/skills .codex/skills | rg 'SKILL.md$' | wc -l
            ```
            """
        ),
    }
    for path, content in doc_files.items():
        ensure_text(path, content.rstrip() + "\n")


def write_skill_trees() -> list[str]:
    slugs: list[str] = []
    for tree in SKILL_TREES:
        tree.mkdir(parents=True, exist_ok=True)
    for group in SKILL_GROUPS:
        for slug, description, archetype in group["skills"]:
            slugs.append(slug)
            body = render_skill(group["group"], slug, description, archetype)
            for tree in SKILL_TREES:
                ensure_text(tree / slug / "SKILL.md", body)
    return slugs


def write_inventories(skill_slugs: list[str]) -> None:
    grouped_lines = ["# Skills Inventory", ""]
    for group in SKILL_GROUPS:
        grouped_lines.append(f"## {group['group']}")
        for slug, _, _ in group["skills"]:
            grouped_lines.append(f"- `{slug}`")
        grouped_lines.append("")
    ensure_text(ROOT / "docs" / "setup" / "skills-inventory.md", "\n".join(grouped_lines).rstrip() + "\n")

    all_files = sorted(
        str(path.relative_to(ROOT))
        for path in ROOT.rglob("*")
        if path.is_file() and ".git" not in path.parts and "__pycache__" not in path.parts
    )
    ensure_text(ROOT / "docs" / "setup" / "files-inventory.txt", "\n".join(all_files).rstrip() + "\n")

    mirrored = [str(path.relative_to(ROOT)) for tree in SKILL_TREES for path in sorted(tree.glob("*/SKILL.md"))]
    ensure_text(ROOT / "docs" / "setup" / "skills-paths.txt", "\n".join(mirrored).rstrip() + "\n")


def main() -> None:
    create_top_level_files()
    create_docs_and_repo_skeleton()
    skill_slugs = write_skill_trees()
    write_inventories(skill_slugs)
    print(f"Scaffolded {len(skill_slugs)} skills in {len(SKILL_TREES)} trees.")


if __name__ == "__main__":
    main()
