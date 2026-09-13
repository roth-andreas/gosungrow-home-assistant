# Specification completeness contract

Status: Normative
Scope: Boundary, evidence, and maintenance of the source-of-truth specification

## Requirements

- **REQ-COMP-001** — `specs/` MUST determine every intentional externally observable input, output, decision, formula, ordering rule, default, validation boundary, state transition, side effect, failure outcome, retry/recovery rule, compatibility behavior, migration, security constraint, persistence rule, user-visible presentation behavior, and delivery rule.
- **REQ-COMP-002** — An implementation task that requires a product decision not determined by `specs/` MUST stop at a specification gap. The gap MUST be resolved through a reviewed normative change and acceptance scenario before dependent implementation proceeds.
- **REQ-COMP-003** — Replaceable mechanics MAY be selected with programming knowledge when they do not affect observable behavior. Package layout, private names, internal data structures, helper decomposition, formatting, CSS declarations, and SVG coordinates are replaceable unless another requirement makes them a compatibility contract.
- **REQ-COMP-004** — Exact external names, protocol fields, serialized keys, topics, paths, schedules, precedence, constants, formulas, and stable ordering MUST be specified when a consumer, persisted artifact, or compatibility boundary depends on them.
- **REQ-COMP-005** — Every tracked product, verification, packaging, and automation surface MUST be classified in [source-inventory.md](source-inventory.md) and owned by at least one normative area. New tracked surfaces MUST be classified in the same change that adds them.
- **REQ-COMP-006** — Every normative requirement MUST have verification evidence in [traceability.md](traceability.md): an automated test, a deterministic repository validator, or an explicitly named review/build check. Requirement ranges are permitted only when every ID in the range exists.
- **REQ-COMP-007** — Endpoint response structs and third-party payload fields that are merely transported and never interpreted MAY remain implementation schema. Every consumed or emitted field that changes a decision or output MUST be specified.
- **REQ-COMP-008** — Locale copy and visual styling MAY be regenerated with equivalent meaning, but locale key parity, selection/fallback rules, accessibility semantics, data meaning, interaction, and compatibility identifiers MUST be normative.
- **REQ-COMP-009** — Completeness claims MUST be checked mechanically where possible and reviewed against the affected source inventory on every behavior change. Passing a validator does not authorize inventing omitted logic.
- **REQ-COMP-010** — Informative documents, source code, tests, screenshots, examples, and historical behavior MUST NOT silently create product logic. They may provide evidence for a proposed normative decision.

## Completeness review questions

A behavior change is incomplete until reviewers can answer yes to all applicable questions:

1. Are all accepted inputs, precedence rules, defaults, boundaries, and rejected inputs determined?
2. Are outputs, identifiers, ordering, precision, units, and side effects determined?
3. Are success, partial success, failure, retry, cancellation, recovery, and persistence determined?
4. Are compatibility, security, privacy, migration, localization, and accessibility outcomes determined?
5. Can each new or changed requirement be verified without relying on a private implementation detail?
6. Can the derived implementation be replaced without changing the contract?

## Prohibited behavior

- Declaring specifications complete because they mirror current code line by line.
- Treating a generated artifact, fixture, test assertion, or library default as authority for an absent decision.
- Hiding observable behavior behind the label “implementation detail.”
- Requiring exact internal structure when multiple implementations satisfy the same contract.
