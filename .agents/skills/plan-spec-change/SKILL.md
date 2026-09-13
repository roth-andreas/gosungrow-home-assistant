---
name: plan-spec-change
description: Produce a drift-safe, read-only plan for changing GoSungrow's authoritative specifications. Use when the user invokes plan-spec-change or $plan-spec-change, or asks to review a specification change before replying "do this". Never modify files or inspect implementation source.
---

# Plan Spec Change

Treat `specs/` as the exclusive source of intentional product logic. Produce an exact plan for review; do not edit specifications or implementation.

## Read-only boundary

- Use read-only operations exclusively. Never edit, create, delete, move, format, stage, commit, switch branches, or mutate external state.
- Read `AGENTS.md`, `specs/README.md`, `specs/governance.md`, `specs/completeness.md`, and relevant files under `specs/`.
- Search only `specs/` for product requirements. Read other non-source Markdown only when explicitly requested.
- Never inspect implementation, tests, configuration, manifests, workflows, scripts, generated assets, or runtime data.
- Do not run validators, builds, tests, generators, or scripts.
- Read-only Git commands are permitted only to capture the repository root, current branch, `HEAD`, status, and hashes of inspected specification files.

## Planning workflow

1. Capture the current branch, full `HEAD`, and `git status --short`.
2. Translate the request into observable inputs, outputs, decisions, defaults, ordering, state transitions, errors, compatibility, security, migration, and prohibited behavior.
3. Locate the owning normative requirements and acceptance scenarios. If the specifications lack context needed to choose behavior, ask one concise question; never inspect source to fill the gap.
4. Preserve requirement IDs. Propose the next unused ID for additions; never reuse removed IDs.
5. State exact additions, replacements, and removals, including precise tables, formulas, strings, precedence, boundaries, and failure behavior when relevant.
6. Include required acceptance, traceability, inventory, ADR, documentation, and compatibility updates.
7. Define explicit non-goals and an implementation envelope: what derived artifacts must change and what behavior must remain unchanged.
8. Record a SHA-256 fingerprint for every inspected specification file so approval cannot be applied to silently changed inputs.
9. Recheck branch, `HEAD`, and status. Report any drift; do not repair it.

## Required output

Return a compact approval packet containing:

- plan title and stable kebab-case slug;
- base branch and full base commit;
- inspected spec files with SHA-256 fingerprints;
- exact changes by file and requirement ID;
- acceptance scenarios and prohibited outcomes;
- compatibility, migration, non-goal, and implementation-envelope decisions;
- assumptions or questions;
- approval sentence.

End with: `No files changed. Reply "do this" to authorize $plan-spec-change-implementation to apply exactly this plan and its derived implementation on one feature branch for one pull request.`

`do this` never grants this planning skill write permission. If the user invokes this skill again while confirming, remain read-only. The implementation skill must stop and request renewed approval if the recorded commit, relevant spec fingerprints, requested scope, or material working-tree state has drifted.
