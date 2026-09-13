# Cross-cutting behavior

Status: Normative  
Scope: Error taxonomy, retry eligibility, files, secrets, determinism, and compatibility

## Error classes

- **REQ-XCUT-001** — Configuration errors (missing credentials/host/token, malformed options), incompatible local data, and programming/runtime failures are non-recoverable unless a subsystem explicitly states otherwise.
- **REQ-XCUT-002** — Recoverable remote errors are login state `-1`, login rejection, invalid app key/token, login-required, cannot-login, HTTP 5xx, internal/bad/service-unavailable/gateway-timeout, DNS resolution, network unreachable, connection refused, context deadline, and I/O timeout.
- **REQ-XCUT-003** — Docker DNS is the recoverable subset containing `127.0.0.11:53` plus no-such-host, temporary-name-resolution, or server-misbehaving. It MUST use DNS backoff and MUST NOT cause gateway rotation/login refresh by itself.
- **REQ-XCUT-004** — Recovery is layered: endpoint performs at most one replay; MQTT startup performs at most three attempts; live MQTT defers to its next-cycle policy; app wrapper restarts only classified failures.

## Files

- **REQ-XCUT-005** — JSON/config/cache writes that replace existing data MUST write a temporary sibling, set requested mode, sync, close, then rename. Temporary files MUST be cleaned on failure; Windows MAY remove the old destination immediately before rename.
- **REQ-XCUT-006** — Parent directories MUST be created with `0755`; normal cached JSON uses the implementation default mode and dashboard state uses `0600`.
- **REQ-XCUT-007** — Removing a nonexistent cache file succeeds. An `unexpected end of JSON input` or JSON syntax error is corruption eligible for cache removal/refetch.
- **REQ-XCUT-008** — Runtime state inside the app MUST live under `/data/.GoSungrow`; assets are read from `/opt/gosungrow/assets`; Home Assistant public card copies live under a config root's `www/gosungrow`.

## Security and privacy

- **REQ-XCUT-009** — User passwords, session tokens, generated symmetric keys, private keys, Supervisor tokens, and MQTT passwords MUST NOT appear in normal logs, specs, dashboard config, or diagnostics. Public protocol client identifiers and the RSA public key are intentionally specified in [isolarcloud-api.md](isolarcloud-api.md).
- **REQ-XCUT-010** — Any human-readable common request output MUST replace a non-empty token with `<redacted>` and leave an empty token empty.
- **REQ-XCUT-011** — HTTP tracing is diagnostic-only and MUST be documented as potentially sensitive. Dashboard diagnostics are limited to IDs, metadata, state values, reasons, and counts.
- **REQ-XCUT-012** — Dashboard mutations require Home Assistant administrator status and fresh server-side optimistic verification. MQTT runtime selects are not a security boundary.

## Determinism and compatibility

- **REQ-XCUT-013** — Sets exposed in generated files/logs/UI MUST use stable sorting or stable discovery order as specified. Canonical JSON MUST not depend on map iteration.
- **REQ-XCUT-014** — Compatibility aliases and legacy app keys MAY remain only when documented. New behavior MUST prefer canonical meanings and MUST surface unsafe legacy behavior for review.
- **REQ-XCUT-015** — Unknown API fields and unknown MQTT option values SHOULD be preserved when safe; unknown result codes/messages, schemas, and semantic conflicts MUST fail closed.
- **REQ-XCUT-016** — A missing optional capability removes only dependent output. It MUST NOT disable unrelated plants, entities, cards, or synchronization.

## Logging

- **REQ-XCUT-017** — Logs MUST identify lifecycle step, retry attempt/delay, sync cycle and endpoint, target selection, and actionable failure category. Repeated Docker-DNS remediation guidance SHOULD be emitted once per outage.
- **REQ-XCUT-018** — Successful recovery MUST log outage duration and resumption of the normal schedule.

## Prohibited behavior

- Retrying all errors indiscriminately.
- Mutating persisted state before an external save has been verified.
- Depending on nondeterministic map order for selection or hashes.
