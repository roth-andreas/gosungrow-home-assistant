---
name: plan-spec-change-implementation
description: Apply the approved plan-spec-change packet from the current conversation, then implement its derived GoSungrow source and tests on one branch for one pull request. Use when the user invokes plan-spec-change-implementation or $plan-spec-change-implementation, or replies "do this" to a current approved plan. Follow docs/go-conventions.md and stop with a proposed specification amendment when required logic is missing or ambiguous.
---

# Apply And Implement An Approved Spec Change

Treat `specs/` as the exclusive source of intentional product logic. Apply the approved normative delta first, then derive implementation and verification from it. Programming judgment is permitted only for behavior-neutral mechanics.

## Approval and drift gate

- Require an approval packet produced by `$plan-spec-change` in the current conversation and an unambiguous user confirmation such as `do this`.
- Read `AGENTS.md`, the complete `docs/go-conventions.md`, `specs/README.md`, `specs/governance.md`, `specs/completeness.md`, every affected normative file, relevant acceptance scenarios, and traceability.
- Compare the packet's full base commit and each relevant spec SHA-256 fingerprint with the repository. Inspect `git status --short` and preserve unrelated changes.
- Stop without writing if the approved plan is missing, ambiguous, materially changed by later instructions, based on a different commit, has fingerprint drift, or overlaps unrelated user changes. Report the drift and request a fresh `$plan-spec-change` plan.
- Before writing, inspect all affected implementation paths and identify decisions the plan did not specify. Apply the specification-gap gate below.

## One branch and one pull request

- Specifications, acceptance changes, implementation, tests, traceability, and required documentation for one behavior change MUST remain on the same feature branch and be submitted in the same pull request.
- If currently on `main` or `master`, create and switch to `spec-change/<approved-plan-slug>` before editing. Otherwise keep the current feature branch unless the user names another branch.
- Do not create a spec-only commit intended to merge independently from its implementation. Commits may be split for review only when all remain in the same branch and pull request.
- Do not push, open a pull request, merge, or commit unless the user explicitly requests that action.

## No invented logic

- Map every externally observable branch, default, formula, validation, ordering rule, retry, side effect, error, compatibility behavior, and prohibited outcome to a normative requirement.
- Do not add fallbacks, configuration, acceptance, rejection, migrations, limits, logging semantics, or UI behavior unless the specifications require them.
- Do not promote undocumented nearby source behavior into the contract.
- Allow types, helpers, data structures, interfaces, cleanup, and refactoring only when behavior-neutral and compliant with `docs/go-conventions.md`.
- Leave unrelated behavior unchanged.

## Specification-gap gate

Stop before choosing behavior when implementation exposes a missing or contradictory default, precedence rule, boundary, failure outcome, state transition, ordering rule, security constraint, compatibility rule, migration, or side effect.

Report:

1. affected requirement IDs and files;
2. the exact undecided behavior and why code cannot remain neutral;
3. source or test evidence that exposed it, without treating source as authority;
4. a precise proposed normative amendment and acceptance scenario;
5. alternatives when multiple product decisions are valid.

Do not edit the unresolved requirement or implement behavior depending on it. Ask the user to review a fresh `$plan-spec-change` packet. Preserve and report any independent work already completed.

## Implementation workflow

1. Validate the starting repository with `python scripts/check_specs.py`.
2. Create or confirm the feature branch, then apply the approved specification and acceptance edits exactly. Run the spec validator again.
3. List affected requirement IDs, acceptance scenarios, prohibited behavior, and implementation owners.
4. Inspect complete affected call paths, callers, tests, fixtures, schemas, persistence, logs, cancellation, configuration, and compatibility boundaries.
5. Add or update tests for specified happy paths, boundaries, failures, regressions, determinism, and security behavior.
6. Implement only the accepted behavior, following `docs/go-conventions.md` and the repository's declared Go version.
7. Update traceability, inventory, user documentation, examples, configuration, assets, and version metadata only when required by the accepted contract.
8. Format changed files and run focused tests while developing.
9. Run every mandatory validator from `docs/go-conventions.md` and `specs/README.md`, plus applicable race, vet, vulnerability, fuzz, frontend, build, shell, and container checks.
10. Review the complete branch diff against every affected requirement and prohibited outcome. Confirm the diff contains both the normative change and all derived artifacts, with no unrelated files.

## Completion

Return:

- branch name and the single-PR boundary;
- implemented requirement IDs and acceptance scenarios;
- changed specification, source, test, and support files;
- validation commands and results;
- `No specification gaps found`, or the gap report;
- environment limitations and remaining work.
