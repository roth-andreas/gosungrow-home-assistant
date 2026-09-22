# Acceptance scenarios

Status: Normative  
Scope: Cross-subsystem executable examples

## Authentication and transport

- **REQ-ACC-001** — Given a missing token file, login proceeds to the network without a file error; given corrupt token JSON, the file is removed and login proceeds.
- **REQ-ACC-002** — Given configured host/key rejection, login tries unique candidates in specified order. Given `127.0.0.11:53` on the first candidate, it stops host rotation and returns for DNS backoff. Given an earlier authentication/gateway failure followed by `127.0.0.11:53` on a later candidate, it stops rotation, preserves both failures, and remains classified as a recoverable remote login sequence rather than a general Docker-DNS outage.
- **REQ-ACC-003** — Given a timed-out HTTP request, a later request can succeed because requests use a bounded private client rather than corrupting the global client.
- **REQ-ACC-004** — Given a common request with a token, debug string contains `<redacted>` and not the token.

## Data

- **REQ-ACC-005** — Numeric and composite plant IDs round-trip; `NULL`, `--`, and invalid composite placeholders do not become zero.
- **REQ-ACC-006** — Given kVAr/MVAr/mVAr variants, reactive power publishes equivalent `var` values and `reactive_power` metadata; applying normalization twice changes nothing.
- **REQ-ACC-007** — Given any missing operand in a virtual formula, dependent virtual points are absent while independent virtual points remain.
- **REQ-ACC-008** — Given a plant-local timestamp, its wall-clock components remain and correct zone offset is attached; invalid timezone retains the original.

## MQTT and Home Assistant

- **REQ-ACC-009** — Given one plant with types 14, 11, and 22, realtime selects type 14; without 14 it selects 11; each additional plant gets its own batch.
- **REQ-ACC-010** — Given Docker DNS loss after connection, MQTT remains connected and delays follow 15/30/60/120/300 seconds; success restores five minutes.
- **REQ-ACC-011** — Given textual, numeric instant, numeric daily, and reactive-power values, discovery respectively has no measurement metadata, measurement, total/last-reset, and canonical reactive metadata.
- **REQ-ACC-030** — Given a numeric `Wp` value, normalization and discovery publish the scaled value in `kWp` with frequency-derived state metadata and `mdi:lightning-bolt`, but without `device_class`; given an otherwise equivalent `kW` value, discovery continues to publish device class `power`.
- **REQ-ACC-031** — Given two producer leaves reporting `1.2 kW` and `800 W` DC with no native total or complete AC tier, the plant publishes `2.000 kW` at `sensor.gosungrow_virtual_<ps_id>_pv_power`. A valid native plant total wins without adding device values; complete AC coverage wins over DC coverage; topology excludes a parent when its children contribute. Mixed incomplete AC/DC coverage, a missing expected contributor, an invalid value, or an incompatible unit produces no new aggregate. A new dashboard selects the plant entity, an existing pinned or manual source is not silently rewritten, and without an aggregate multiple producers cannot resolve to one automatic plant total.
- **REQ-ACC-032** — Given plant-tree UUID/UpUUID relationships, producer leaves are resolved within their opaque plant ID. Duplicate UUIDs, dangling or cross-plant parent references, and unavailable topology suppress device aggregation without stopping MQTT; a native plant total still publishes. Successful token recovery refreshes both device inventory and plant topology.

## Dashboard

- **REQ-ACC-012** — Given multiple type-14 devices, each gets all prototype views with unique path/title; no valid keys yields a clear error.
- **REQ-ACC-013** — Given a renamed entity with canonical registry unique ID, matching succeeds; conflicting direction/lifetime fails; multiple inverters prevent confident single-inverter production.
- **REQ-ACC-014** — Given no battery capability, battery cards/entities are pruned only for that target. PV-to-battery energy alone does not establish a battery.
- **REQ-ACC-015** — Given a manual override, reconciliation preserves it; if its entity disappears it remains selected/unavailable; reset restores automatic; two targets sharing a default remain isolated.
- **REQ-ACC-016** — Given direct solar above production by more than tolerance, UI requires review/confirmation. Unsupported calculated direct solar cannot be newly selected.
- **REQ-ACC-017** — Given a stale dashboard between preview and save, save aborts and local state remains unchanged. Given successful save, re-read verifies every binding before local update.
- **REQ-ACC-018** — Given locale `de-DE`, lookup uses `de`; unknown locale uses English; every supported locale has parity.
- **REQ-ACC-019** — Given source-only dashboard changes, structure hash remains stable; unrelated layout change is external modification and blocks non-forced replacement.

## Frontend summaries

- **REQ-ACC-020** — Fresh install with only live daily value displays one bucket for day/month/year. Current-day recorder rows are replaced, not added.
- **REQ-ACC-021** — Month/year boundaries keep completed previous period totals and start current period from live day. Missing data yields empty chart.
- **REQ-ACC-022** — Date labels follow Home Assistant preferences, relabel cached data without refetch, and expose matching keyboard/tooltip labels.

## Add-on and release

- **REQ-ACC-023** — Dashboard failure logs warning and MQTT starts; missing credentials/host is fatal; panic output exits without login refresh.
- **REQ-ACC-024** — Binary and app versions align, specs pass consistency validation, Go tests pass, shell parses, and amd64 image builds before publication.

## Configuration and source-of-truth workflow

- **REQ-ACC-025** — Given `GOSUNGROW_TIMESTAMP_OFFSET_MS=" 1500 "`, a request timestamp advances by 1500 milliseconds after the learned server offset; given an absent, empty, or invalid value, it advances by zero diagnostic milliseconds.
- **REQ-ACC-026** — Given an approved plan packet followed by `do this`, spec and derived implementation changes are made on one feature branch for one pull request; given base-commit, fingerprint, scope, or overlapping-worktree drift, no plan-dependent write occurs until renewed approval.
- **REQ-ACC-027** — Given a new tracked product file or repository-owned environment lookup, validation fails until the source inventory or configuration catalog classifies it.

## Failure diagnostics

- **REQ-ACC-028** — Given a stable recoverable-remote classification whose human-readable diagnostic contains Docker-DNS text, the app wrapper uses normal remote recovery; given a stable Docker-DNS classification whose text does not contain resolver keywords, it uses DNS backoff. Human-readable wording alone never selects recovery.
- **REQ-ACC-029** — Given a failed multi-candidate login sequence, diagnostics list hosts in attempt order, identify the first failure and terminal stop reason, retain at most two distinct messages per host, and contain no user credentials or session tokens.

## Prohibited behavior

An acceptance test MUST NOT weaken its governing subsystem requirement. When an example and a subsystem requirement appear inconsistent, the more specific safety constraint wins and the inconsistency must be corrected.
