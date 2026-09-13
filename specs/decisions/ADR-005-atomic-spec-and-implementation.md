# ADR-005: Keep specification and implementation changes atomic

Status: Accepted
Date: 2026-09-13

## Context

Separating a normative change from its derived implementation allows either branch to describe behavior the repository does not provide. A conversational approval can also become stale when specifications or source change before implementation.

## Decision

Planning is a read-only approval step. An approved packet records the base commit and inspected-spec fingerprints. The write-enabled step rejects drift, then changes normative specifications and all derived artifacts on one feature branch for one pull request.

Non-behavioral corrections may be spec-only only when explicitly identified as such and no derived artifact needs to change.

## Consequences

- Reviewers see the intended contract and implementation together.
- The default branch remains internally consistent after merge.
- Stale conversational approvals are not applied silently.
- Larger changes may produce larger pull requests, so plans should stay behaviorally focused.
