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

- **REQ-DASH-017** — The bundled card file MUST be written to `www/gosungrow/` under each unique available candidate config root: configured directory, `/homeassistant`, `/config`. At least one write must succeed.
- **REQ-DASH-018** — The registered JavaScript resource MUST be a `module` whose URL is an embedded base64 JavaScript data URL with a fragment containing the first 12 hexadecimal SHA-256 characters. Filesystem copies are still written for inspection and fallback delivery.
- **REQ-DASH-019** — Existing managed card resources, including stale versions/data URLs, MUST be updated rather than duplicated. Unrelated resources MUST remain untouched.
- **REQ-DASH-020** — Websocket connection uses bearer token, 15-second handshake/write timeout, 30-second reads, monotonically increasing request IDs, and ignores unrelated response messages until the matching result arrives.
- **REQ-DASH-021** — Supported Home Assistant operations are dashboard list/create/update, Lovelace config read/save, resource list/create/update, states, entity registry, preferred-language calls. Structured websocket error code/message MUST be preserved.
- **REQ-DASH-022** — State list MUST discard empty IDs and deduplicate case-insensitively. Registry metadata MUST enrich matching states by case-insensitive entity ID; registry failure degrades to conservative fallback instead of aborting install.

## Ownership and persistence

- **REQ-DASH-023** — State file is `dashboard_state.json` beside `GOSUNGROW_CONFIG`, or the OS temporary directory when that variable is absent; it stores URL path, full config hash, structure hash, target keys, accepted overrides, and UTC update timestamp with mode `0600`.
- **REQ-DASH-024** — Canonical JSON hashing MUST preserve array order, sort object keys, encode primitives as JSON, and use full SHA-256 hex.
- **REQ-DASH-025** — A dashboard is managed only when persisted state names the same URL path. A non-storage dashboard at the path MUST be rejected.
- **REQ-DASH-026** — An unmanaged existing dashboard or an externally modified managed dashboard MUST be rejected unless force update is true.
- **REQ-DASH-027** — Source-only changes are not external structural modification. Structure hashing MUST replace valid bound source paths with stable metric placeholders and remove source defaults, overrides, bindings, candidates, recommendations, matcher version, and live/source status fields.
- **REQ-DASH-028** — Legacy state without a structure hash MAY be accepted only when normalized current and desired structures match. Config is saved only when new, changed, or forced; metadata is updated/created independently.

## Diagnostics

- **REQ-DASH-029** — Installation MUST report context, state-load result, GoSungrow-state count, reference/remap/unresolved counts, battery detection, save decision/reason, targets and selection, relevant warnings, and unresolved references.
- **REQ-DASH-030** — Debug mode additionally reports bounded candidate traces. Diagnostics MAY include entity IDs/current values but MUST exclude credentials and tokens.

## Prohibited behavior

- Cross-plant source assignment on multi-target dashboards.
- Treating a missing manual entity as permission to choose a replacement.
- Overwriting user structure without force permission.
