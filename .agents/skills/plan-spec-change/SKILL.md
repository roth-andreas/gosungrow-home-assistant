---
name: plan-spec-change
description: Read-only planning for proposed changes to GoSungrow's authoritative specifications. Use when the user invokes plan-spec-change or $plan-spec-change, or wants to review an exact specification-change plan before authorizing edits with a later "do this" message. Never modify files or inspect implementation source.
---

# Plan Spec Change

Treat `specs/` as the source of truth for all intentional program logic. Produce a reviewable specification-change plan only. Do not implement the plan, change specifications, or change program source.

## Read-only boundary

- Use read-only operations exclusively. Never edit, create, delete, move, format, stage, commit, or otherwise mutate files or external state.
- Read `AGENTS.md`, then `specs/README.md`, `specs/governance.md`, and relevant files under `specs/`.
- Search only `specs/` for requirements, terminology, IDs, decisions, and traceability.
- Read other non-source Markdown only when the user explicitly supplies or requests it.
- Never inspect implementation source, tests, generated assets, configuration, manifests, workflows, build scripts, or runtime data.
- If the plan would require implementation knowledge absent from the specifications, state the gap or ask one concise question. Do not inspect source to fill it.
- Do not run validators, builds, tests, generators, or scripts during this planning phase.

## Planning workflow

1. Capture `git status --short` as a read-only baseline before inspection.
2. Translate the request into observable behavior, constraints, defaults, state transitions, errors, compatibility rules, and prohibited behavior.
3. Locate the owning normative documents and related requirements without modifying them.
4. Preserve existing requirement IDs in the proposal. For new requirements, propose the next unused ID in the owning area; never propose reusing a removed ID.
5. Specify each proposed addition, replacement, or removal precisely enough to apply without redesign. Include exact tables, formulas, ordering, defaults, and error behavior when relevant.
6. Identify required acceptance-scenario changes and any affected cross-references, traceability entries, migration classification, or ADRs.
7. Separate the specification phase from later implementation. Describe implementation expectations and acceptance behavior, but do not claim source or tests already comply.
8. Identify assumptions, conflicts, open decisions, and explicit non-goals. Ask for clarification instead of choosing when alternatives materially change the contract.
9. Compare final working-tree status with the baseline. If it differs, report the unexpected mutation; never conceal or repair it with another mutation.

## Output format

Return a compact plan with:

- proposed behavioral outcome;
- requirement IDs and exact changes by specification file;
- acceptance scenarios to add or update;
- compatibility, migration, and non-goal decisions;
- assumptions or questions;
- later implementation expectations;
- confirmation prompt.

End with: `No files changed. Reply "do this" to authorize a separate write-enabled turn to apply exactly this specification plan.`

`do this` is a confirmation token for the subsequent write-enabled turn; it never grants this planning skill permission to write. If the user invokes this skill again while confirming, remain read-only.

If relevant files change before confirmation, the write-enabled turn must report the drift and obtain renewed approval rather than silently applying a stale plan.
