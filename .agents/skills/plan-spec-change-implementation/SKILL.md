---
name: plan-spec-change-implementation
description: Implement an approved GoSungrow specification change from the current conversation, after the specs have been updated and confirmed. Use when the user invokes plan-spec-change-implementation or $plan-spec-change-implementation to make source, test, configuration, asset, and documentation changes strictly derived from specs/. Follow docs/go-conventions.md and stop with a proposed spec amendment if required behavior is missing or ambiguous.
---

# Implement Spec Change

Treat `specs/` as the exclusive source of product logic. Translate an already approved and applied specification change into implementation and tests. Use programming judgment only for behavior-neutral mechanics.

## Preconditions

- Read `AGENTS.md` and the complete `docs/go-conventions.md` before inspecting or changing implementation.
- Read `specs/README.md`, `specs/governance.md`, affected normative specifications, `specs/acceptance.md`, and relevant `specs/traceability.md` entries.
- Identify the approved spec delta from the current conversation and the working tree or named commit. Prefer an uncommitted `specs/` diff; otherwise inspect the explicitly identified spec commit.
- Require a concrete, already-applied specification delta. If the target delta is absent or ambiguous, ask the user to identify it or use `$plan-spec-change`; do not infer a feature from a general request.
- Capture working-tree status and preserve every unrelated user change.

## No invented logic

- Map every externally observable branch, default, formula, validation, ordering rule, retry, side effect, error, compatibility behavior, and prohibited outcome to an affected normative requirement.
- Do not add helpful fallbacks, new configuration, broader acceptance, stricter rejection, migrations, logging semantics, limits, or UI behavior unless specifications require them.
- Do not copy undocumented behavior from existing source into the new implementation merely because it already exists nearby.
- Allow ordinary implementation mechanics—types, helpers, data structures, interfaces, cleanup, and refactoring—only when they do not create product behavior and comply with `docs/go-conventions.md`.
- Leave unrelated unspecified behavior unchanged.
- Do not modify normative specifications during this skill. `specs/traceability.md` may be updated mechanically for new tests or moved implementation ownership, without adding product logic.

## Specification-gap gate

Stop before choosing behavior when implementation exposes a missing or contradictory decision, including an unspecified default, precedence rule, input boundary, failure outcome, state transition, ordering rule, security constraint, compatibility rule, migration, or external side effect.

Report the gap with:

1. affected requirement IDs and specification file;
2. the exact missing decision and why code cannot remain neutral;
3. source or test evidence that exposed the gap, without treating it as authority;
4. a precise proposed normative addition or replacement and acceptance scenario;
5. alternatives when more than one product decision is valid.

Do not edit the specification or implement the unresolved behavior. Ask the user to review the proposal with `$plan-spec-change`, apply the approved spec update, then invoke this skill again. Continue only independent work that cannot prejudice the unresolved decision.

## Implementation workflow

1. Validate the starting specifications with `python scripts/check_specs.py`.
2. List affected requirement IDs, acceptance scenarios, prohibited behaviors, and implementation owners from traceability.
3. Inspect the complete affected call paths, callers, tests, fixtures, schemas, persistence, logging, cancellation, and compatibility boundaries.
4. Design the smallest complete change. Avoid unrelated cleanup and preserve public contracts unless the specs explicitly change them.
5. Add or update tests that prove the happy path, boundaries, specified failures, regression case, determinism, and security behavior relevant to the requirements.
6. Implement only the accepted behavior. Follow all applicable rules in `docs/go-conventions.md`; use the repository's declared Go version and established language-specific conventions for non-Go files.
7. Update user-facing non-normative documentation, examples, configuration, assets, version metadata, and `specs/traceability.md` only when required by the accepted specifications or repository governance.
8. Format changed files and run focused tests during development.
9. Run all mandatory validation from `docs/go-conventions.md` and `specs/README.md`, plus risk-specific race, vet, vulnerability, fuzz, build, frontend, or container checks when applicable.
10. Review the final diff against every affected requirement and prohibited behavior. Confirm no behavior lacks a specification and no unrelated workspace file entered the change.

## Completion

Return a compact implementation report containing:

- implemented requirement IDs and acceptance scenarios;
- changed source, test, and supporting files;
- validation commands and results;
- specification gaps found, or `No specification gaps found`;
- environment limitations and remaining work, if any.

Do not commit unless the user explicitly asks.
