---
name: change-specs
description: Change GoSungrow's authoritative behavioral specifications without inspecting or modifying implementation source. Use when the user invokes change-specs or $change-specs, or requests a specs-only change to intended behavior, requirements, acceptance criteria, compatibility rules, or architecture decisions before a separate implementation pass.
---

# Change Specs

Treat `specs/` as the source of truth for all intentional program logic. Perform requirements design only; a separate implementation skill will later make source and test changes from the resulting specification diff.

## Boundaries

- Read `AGENTS.md`, then `specs/README.md`, `specs/governance.md`, and the relevant files under `specs/`.
- Search only `specs/` when discovering requirements, terminology, IDs, or cross-references.
- Write only under `specs/` by default.
- Modify Markdown documentation outside `specs/` only when the user explicitly requests that documentation change. Keep such edits non-source.
- Do not inspect or modify implementation source, tests, generated assets, configuration, manifests, workflows, or build/release scripts.
- Do not use current code behavior to weaken or redefine a requested requirement.
- Do not commit unless the user explicitly asks.

## Workflow

1. Translate the request into observable behavior, constraints, defaults, state transitions, errors, compatibility rules, and prohibited behavior.
2. Ask a concise question only when an unresolved choice would materially change the contract. Otherwise make the smallest explicit assumption and record it in the specification.
3. Locate the owning normative document and related requirements using searches scoped to `specs/`.
4. Preserve existing requirement IDs. Add new IDs using the next unused number in the owning area; never reuse removed IDs.
5. Update every affected normative rule, table, formula, ordering rule, default, and prohibited behavior. Avoid implementation-language details unless they are compatibility requirements.
6. Update `specs/acceptance.md` with externally verifiable scenarios. Update cross-references, traceability, migration classification, or an ADR when the change materially affects them.
7. Keep the implementation boundary honest: do not claim that source or tests already implement a newly changed requirement. Express expected verification as acceptance behavior for the later implementation pass.
8. Run `python scripts/check_specs.py`. Do not run source builds or implementation tests during this specs-only phase.
9. Review `git diff -- specs` and confirm that no disallowed file was changed.

## Handoff

Return a compact implementation handoff containing:

- changed requirement IDs and specification files;
- the behavioral delta and important non-goals;
- acceptance scenarios the implementation must satisfy;
- any explicit assumptions or unresolved decisions;
- specification-validation result;
- the statement `Implementation intentionally not changed.`

The handoff must be sufficient for a later implementation skill to update code and tests without rediscovering the requested product behavior.
