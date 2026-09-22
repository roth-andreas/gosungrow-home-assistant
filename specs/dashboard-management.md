# Managed dashboard lifecycle

Status: Normative  
Scope: Target discovery, template rendering, Home Assistant websocket operations, resources, ownership, localization, pruning, and diagnostics

## Defaults and prerequisites

- **REQ-DASH-001** — Defaults are asset directory `/opt/gosungrow/assets`, Home Assistant config `/homeassistant`, websocket `ws://supervisor/core/websocket`, URL path `gosungrow-flow`, title `GoSungrow Flow`, icon `mdi:solar-power`, visible in sidebar, non-admin-readable, language `auto`, and no force update.
- **REQ-DASH-002** — Installation requires `SUPERVISOR_TOKEN` and a forced valid iSolarCloud login. Failure to authenticate, discover any valid target, load assets, authenticate Home Assistant, or save required managed state MUST fail the dashboard command.

## Target discovery

- **REQ-DASH-003** — Plants MUST be processed by sorted opaque `ps_id`. Devices without `ps_key` are ignored.
- **REQ-DASH-004** — Every type-14 device in a plant is a dashboard target. If none exists, select exactly one fallback ranked `11`, `7`, `1`, then other; ties preserve deterministic discovery order.
- **REQ-DASH-005** — Target selection source MUST identify the chosen device type. All usable plant devices and which one is selected MUST be retained for diagnostics and semantic matching.
- **REQ-DASH-006** — A target title is the normalized plant name; use `Plant <ps_id>` if absent. With multiple targets in a plant, append distinct device name or key. If titles still collide, append `ps_id`.
- **REQ-DASH-007** — View paths MUST lowercase, convert `_` to `-`, replace non-alphanumerics with `-`, trim separators, and fall back to `gosungrow-flow`.

## Template and layout

- **REQ-DASH-008** — The bundled YAML MUST contain prototype views `Overview`, `Aggregates`, `Trends`, and panel view `Data Sources`, in that order, containing the metric set from [domain-model.md](domain-model.md).
- **REQ-DASH-009** — Every prototype MUST be cloned for every target and every `YOUR_ESS_PS_KEY` occurrence recursively replaced. With multiple targets, titles and paths combine target and prototype; with one target, prototype titles/paths remain.
- **REQ-DASH-010** — Overview MUST show live flow, today tiles, 24-hour power balance, battery state, and daily energy. Trends MUST show power balance, solar allocation, load sources, grid exchange, battery flow/SOC, and daily energy. Aggregates MUST show day/month/year summaries for production, consumption, grid directions, and battery directions.
- **REQ-DASH-011** — Unsupported target-specific metrics MUST be pruned from entity maps/lists and cards; empty non-heading cards and sections containing only headings MUST be removed. Pruning MUST be per target and MUST not run when Home Assistant states are unavailable.
- **REQ-DASH-012** — Battery capability requires a usable numeric, target-affine SOC entity with unit `%`/empty or battery-power entity with power unit/empty. Battery energy alone does not establish capability.
- **REQ-DASH-013** — Battery metrics are `battery_power`, `p13141`, `p13174`, `p13029`, `pv_to_battery_power`, and `battery_to_load_power`. `p13116` is unsupported when no canonical candidate and no pinned default exists. Any unresolved metric is unsupported.

## Localization

- **REQ-DASH-014** — Supported locales are `en`, `de`, `sv`, and `es`; locale files MUST have identical key sets for dashboard and app option translations.
- **REQ-DASH-015** — `auto` MUST query Home Assistant user frontend data first and core config second. Language normalization lowercases and changes `_` to `-`; lookup order is full locale, base language, then English.
- **REQ-DASH-016** — Missing locale/key or blank translation MUST fall back to English. Exact known template strings are localized recursively, and flow-card labels are injected into every flow card.

## Asset and websocket contract

- **REQ-DASH-017** — The installer MUST calculate the bundled card's full SHA-256 hash and atomically stage identical bytes under every writable unique candidate config root (configured directory, `/homeassistant`, `/config`) at `www/gosungrow/gosungrow-dashboard-cards.<sha12>.js`, where `<sha12>` is the first 12 lowercase hexadecimal hash characters. Every successful write MUST be re-read and match the full hash, and at least one canonical write MUST succeed.
- **REQ-DASH-018** — The registered JavaScript resource MUST have type `module` and canonical URL `/local/gosungrow/gosungrow-dashboard-cards.<sha12>.js`. Before activation, a request to Home Assistant's separately resolved static HTTP origin for that exact path MUST return status 200, a JavaScript MIME type, and bytes matching the full expected hash. Newly registered data URLs, mutable local URLs, and CDN URLs are prohibited.
- **REQ-DASH-019** — Legacy GoSungrow data URLs, CDN URLs, unversioned local URLs, and content-addressed local URLs MUST all be recognized as managed resources and migrated to exactly one canonical resource without duplicating it or changing unrelated resources. After successful activation, each candidate root MUST retain the active and immediately previous verified content-addressed bundles and remove only older unreferenced managed bundles.
- **REQ-DASH-020** — Websocket connection uses bearer token, 15-second handshake/write timeout, 30-second reads, monotonically increasing request IDs, and ignores unrelated response messages until the matching result arrives.
- **REQ-DASH-021** — Supported Home Assistant operations are dashboard list/create/update/delete, Lovelace config read/save, resource list/create/update/delete/reload, states, entity registry, and preferred-language calls. Structured websocket error code/message MUST be preserved.
- **REQ-DASH-022** — State list MUST discard empty IDs and deduplicate case-insensitively. Registry metadata MUST enrich matching states by case-insensitive entity ID; registry failure degrades to conservative fallback instead of aborting install.

## Ownership and persistence

- **REQ-DASH-023** — State file is `dashboard_state.json` beside `GOSUNGROW_CONFIG`, or the OS temporary directory when that variable is absent; it stores URL path, full config hash, structure hash, target keys, accepted overrides, asset mode (`enhanced` or `native-fallback`), active asset URL and full hash, previous asset URL and full hash, and UTC update timestamp with mode `0600`. Legacy state without asset fields MUST remain readable and MUST be migrated on the next successful reconciliation.
- **REQ-DASH-024** — Canonical JSON hashing MUST preserve array order, sort object keys, encode primitives as JSON, and use full SHA-256 hex.
- **REQ-DASH-025** — A dashboard is managed only when persisted state names the same URL path. A non-storage dashboard at the path MUST be rejected.
- **REQ-DASH-026** — An unmanaged existing dashboard or an externally modified managed dashboard MUST be rejected unless force update is true.
- **REQ-DASH-027** — Source-only changes are not external structural modification. Structure hashing MUST replace valid bound source paths with stable metric placeholders and remove source defaults, overrides, bindings, candidates, recommendations, matcher version, and live/source status fields.
- **REQ-DASH-028** — Legacy state without a structure hash MAY be accepted only when normalized current and desired structures match. Config is saved only when new, changed, or forced; metadata is updated/created independently.

## Diagnostics

- **REQ-DASH-029** — Installation MUST report context, state-load result, GoSungrow-state count, reference/remap/unresolved counts, battery detection, save decision/reason, targets and selection, relevant warnings, and unresolved references.
- **REQ-DASH-030** — Debug mode additionally reports bounded candidate traces. Diagnostics MAY include entity IDs/current values but MUST exclude credentials and tokens.
- **REQ-DASH-031** — Enhanced dashboard delivery MUST be one transaction: build the complete desired dashboard and verify ownership before mutation; stage and hash the asset; verify its Home Assistant HTTP response; create or update the managed resource; re-read and verify its exact URL and type; save and re-read the dashboard; then persist local state. If any step after resource mutation fails, the installer MUST restore the prior managed resource registration and dashboard configuration. A failed upgrade of an existing working installation MUST leave its prior resource, dashboard, and state usable.
- **REQ-DASH-032** — On a fresh installation where custom-card activation fails but Home Assistant dashboard storage remains available, the installer MUST save a native fallback rather than custom-card references. The fallback MUST preserve targets and view paths, present the ten resolved live metrics in canonical domain order through native cards, present available current-period summary entities through native cards, replace source mapping with a localized informational card that editing may be unavailable, and contain no `custom:gosungrow-*` type. Later reconciliation MUST automatically promote the dashboard to enhanced mode without losing target or source-selection decisions.
- **REQ-DASH-033** — After the first managed-resource registration or a managed-resource URL replacement, the installer MUST use Home Assistant's storage-resource mutation and re-read path so new browser sessions receive the new URL. If the active Home Assistant resource mode exposes a resource-reload service, the installer MUST request it; it MUST NOT call the YAML-only reload service when storage mode does not expose it. A browser opened before registration MAY require one ordinary reload; content-addressed URLs MUST make a hard refresh unnecessary. The installer MUST NOT forcibly navigate or reload connected browsers.
- **REQ-DASH-034** — Asset lifecycle diagnostics MUST use bounded structured fields covering phase, expected hash, canonical URL, HTTP verification route (`supervisor-core-info` or `websocket-origin`), Supervisor metadata HTTP status and outcome, discovered Core port and TLS setting, asset HTTP status and MIME type, resource action, dashboard mode, rollback result, and cleanup result. They MUST NOT print HTTP origin URLs, IP addresses, Supervisor metadata response bodies, JavaScript bodies, base64-encoded assets, credentials, or tokens.
- **REQ-DASH-035** — Static asset verification MUST resolve transport independently from the Home Assistant API websocket. When the normalized websocket hostname is `supervisor` and its path ends in `/core/websocket`, before every installation or reconciliation verification it MUST request `http://supervisor/core/info` with `Accept: application/json`, the `SUPERVISOR_TOKEN` as a bearer credential, a 15-second request bound, and redirects disabled. The response MUST have HTTP status 200 and a JSON envelope with `result` exactly `ok`, `data.port` as an integer from 1 through 65535, and `data.ssl` as a boolean; unknown fields MUST be ignored. The verifier MUST use scheme `https` when `data.ssl` is true and `http` otherwise, hostname `homeassistant`, and the exact advertised port. It MUST NOT assume port 80 or 8123, probe ports, silently fall back when discovery is unavailable or invalid, persist or reuse discovered topology across reconciliations, or disable standard TLS certificate verification. Otherwise it MUST map `ws` to `http` and `wss` to `https`, remove a terminal `/api/websocket` or `/websocket` while preserving any preceding base path, and use the resulting origin. The verifier MUST append the canonical resource path without changing it, MUST NOT request `/core/local/...` through Supervisor, MUST attach the Supervisor credential only to `/core/info` and no credential to the static request, and MUST NOT follow redirects.
- **REQ-DASH-036** — Supervisor endpoint-discovery failure and static asset verification failure MUST remain custom-card activation failures governed by `REQ-DASH-031` and `REQ-DASH-032`: an existing working dashboard remains usable, a fresh installation receives the native fallback, and MQTT startup continues. Diagnostics and returned error chains MUST distinguish Supervisor metadata discovery from the subsequent static asset fetch while preserving a safe causal error.

## Prohibited behavior

- Cross-plant source assignment on multi-target dashboards.
- Treating a missing manual entity as permission to choose a replacement.
- Overwriting user structure without force permission.
- Saving `custom:gosungrow-*` references before their exact resource has been verified and activated.
- Deleting the last verified working managed bundle or changing unrelated Lovelace resources.
