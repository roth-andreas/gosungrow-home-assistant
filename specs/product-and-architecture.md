# Product and architecture

Status: Normative  
Scope: Product boundary, actors, lifecycle, and subsystem contracts

## Purpose

GoSungrow is primarily a Home Assistant app. It authenticates to Sungrow iSolarCloud, converts plant/device points into a stable internal model, publishes MQTT discovery and retained states, and creates a managed Home Assistant dashboard. A CLI exposes authentication, MQTT operation, and dashboard installation for the app and local diagnostics.

## Requirements

- **REQ-PROD-001** — The supported deployment MUST be the Home Assistant app on `aarch64` and `amd64`; the standalone CLI remains supported for local operation and diagnostics.
- **REQ-PROD-002** — The main runtime pipeline MUST be `app configuration → iSolarCloud session → plant/device/point discovery → normalized data → MQTT discovery/state → managed dashboard`.
- **REQ-PROD-003** — MQTT MUST be a prerequisite. The product is not a native Home Assistant integration and MUST NOT claim to operate without a broker and Home Assistant MQTT integration.
- **REQ-PROD-004** — Dashboard installation failure MUST NOT prevent MQTT startup. Missing credentials or MQTT host MUST prevent startup.
- **REQ-PROD-005** — The app MUST keep MQTT connected during recoverable iSolarCloud outages after MQTT initialization and MUST resume normal synchronization automatically.
- **REQ-PROD-006** — Managed dashboard overrides MUST affect only the managed GoSungrow dashboard; they MUST NOT alter MQTT entities, automations, Home Assistant Energy configuration, or unrelated dashboards.
- **REQ-PROD-007** — Multiple plants and multiple selected targets MUST remain isolated by stable Sungrow identifiers.
- **REQ-PROD-008** — The system SHOULD prefer conservative absence or an explicit review state over silently assigning semantically ambiguous energy data.
- **REQ-PROD-009** — All externally visible output MUST be deterministic for identical inputs, except timestamps, random transport material, network ordering normalized by the specification, and explicitly live Home Assistant values.
- **REQ-PROD-010** — The managed dashboard MUST retain a resource-independent usable native mode when browser enhancements cannot be activated. Custom-card delivery failure MUST affect only enhanced presentation and dashboard source editing; it MUST NOT interrupt MQTT publication, native live metrics, dashboard reconciliation, or require a Home Assistant restart.

## Component responsibilities

| Component | Responsibility | Must not own |
|---|---|---|
| App wrapper | Resolve options/services; orchestrate startup and restart | Energy semantics |
| iSolarCloud client | Transport, auth, endpoint schema, caching, recovery | Home Assistant presentation |
| Normalizer | Typed values, point metadata, virtual formulas | Broker lifecycle |
| MQTT publisher | Discovery, device hierarchy, schedules, state publication | Dashboard source persistence |
| Dashboard manager | Target discovery, rendering, matching, ownership | Modification of MQTT identities |
| Custom cards | Render live state/statistics; safe admin source edits | Server-side candidate generation |

## Lifecycle

1. Validate app inputs and resolve MQTT credentials.
2. Write the CLI configuration and discard empty JSON cache files.
3. Optionally attempt dashboard installation and schedule reconciliations.
4. Authenticate, discover devices and metadata, connect MQTT, and publish discovery.
5. Perform an immediate synchronization, then repeat on schedule.
6. Recover transient auth/network failures according to [cross-cutting.md](cross-cutting.md).

## Prohibited behavior and non-goals

- The app MUST NOT require a Home Assistant restart after dashboard installation.
- The app MUST NOT configure a fixed IP for iSolarCloud gateways.
- The app MUST NOT overwrite an unmanaged or externally modified dashboard unless force update is enabled.
- The app MUST NOT expose authentication tokens or passwords in diagnostics.
- Personal files under `notgit/` and preview output are outside the product contract.
