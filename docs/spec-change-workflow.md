# Specification change workflow

`specs/` is the product-logic source of truth. The workflow separates product decisions from implementation while keeping the accepted contract and code atomic.

## Example

Start a planning turn:

```text
$plan-spec-change Change MQTT polling so a transient plant failure does not prevent successful plants from publishing. Retry each failed plant once after 10 seconds.
```

The planning skill reads only the specification system. It returns an exact, read-only approval packet with requirement IDs, wording changes, acceptance scenarios, non-goals, base commit, and fingerprints. It changes no files and never inspects source.

Review the packet. If it is correct, reply in the same conversation:

```text
do this
```

That invokes `$plan-spec-change-implementation`. It verifies that the approval is current, creates `spec-change/<plan-slug>` when starting from `main` or `master`, checks source for unplanned product decisions, applies the approved spec changes, implements only that behavior, updates tests and traceability, and runs repository validation.

## End state

When the implementation turn finishes successfully:

- normative specs, acceptance scenarios, source, tests, traceability, and required docs are together on one feature branch;
- the branch is ready for one pull request containing the complete behavioral change;
- the completion report lists requirement IDs, files, and validation results;
- no push, commit, pull request, or merge occurs unless explicitly requested.

If the repository drifted or implementation exposes a missing decision, work stops. The assistant reports the exact conflict or proposes exact spec wording and asks for a fresh planning/approval cycle; it does not invent the missing logic.
