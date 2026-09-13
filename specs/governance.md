# Specification governance

Status: Normative  
Scope: All product and delivery behavior

## Requirements

- **REQ-GOV-001** — Documents marked `Status: Normative` MUST be the authority for intended behavior.
- **REQ-GOV-002** — `MUST` and `MUST NOT` denote unconditional requirements; `SHOULD` denotes a default that requires a documented exception; `MAY` denotes permitted optional behavior.
- **REQ-GOV-003** — Every normative rule MUST have a stable, unique `REQ-<AREA>-<NUMBER>` identifier. Renamed rules retain their ID; removed IDs are never reused.
- **REQ-GOV-004** — Behavior changes MUST update specifications and acceptance criteria before or in the same change as implementation and tests.
- **REQ-GOV-005** — Tests MUST verify externally meaningful behavior rather than private implementation structure where practical.
- **REQ-GOV-006** — Traceability is informative about file ownership but MUST cover every normative area and all behavioral test files.
- **REQ-GOV-007** — Unknown or disputed current behavior MUST be resolved explicitly; it MUST NOT be silently promoted into the specification.
- **REQ-GOV-008** — User documentation MAY summarize requirements but MUST NOT redefine them inconsistently.
- **REQ-GOV-009** — Tables and fenced examples in normative documents are normative when introduced with `MUST`; otherwise they are explanatory.
- **REQ-GOV-010** — Normative specifications MUST contain no unresolved placeholder markers.
- **REQ-GOV-011** — A behavior change's normative specifications, acceptance criteria, derived source, tests, traceability, and required documentation MUST be developed on one feature branch and submitted in one pull request. A normative-only pull request is permitted only for an explicitly labeled non-behavioral correction that requires no derived artifact change.
- **REQ-GOV-012** — An approved specification plan MUST identify its base commit and inspected-spec fingerprints. Its implementation MUST stop for renewed approval when those inputs or the approved scope drift.
- **REQ-GOV-013** — A specification change MUST satisfy the completeness review in `REQ-COMP-*` and keep the tracked source inventory classified.

## Specification style

Requirements describe observable results, constraints, algorithms needed for compatibility, and prohibited outcomes. They avoid language-specific structure. Exact strings, identifiers, paths, schedules, formulas, protocol fields, and ordering are specified when consumers depend on them.

## Prohibited behavior

- Treating code comments, old changelog entries, screenshots, preview fixtures, or undocumented quirks as higher authority than accepted specs.
- Changing a requirement merely to make a failing implementation pass without deciding the intended behavior.
- Storing credentials, tokens, private installations, or captured personal API payloads in this directory.
- Merging an intentional behavior contract separately from the implementation that makes it true.
