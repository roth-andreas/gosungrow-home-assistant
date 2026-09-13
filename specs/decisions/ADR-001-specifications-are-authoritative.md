# ADR-001: Specifications are authoritative

Status: Accepted  
Date: 2026-09-13

## Decision

Normative Markdown under `specs/` defines intended behavior. Code, tests, user docs, and generated assets implement or summarize it.

## Consequences

Behavior changes begin with requirements and acceptance criteria. Stable IDs permit traceability and assistant context. Implementation can be reorganized or regenerated without changing the contract. Low-level implementation mechanics remain outside the spec unless required for compatibility, security, or determinism.
