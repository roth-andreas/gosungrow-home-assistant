# Dashboard source recommendations and overrides

Status: Normative  
Scope: Mapping-card schema, candidates, validation, persistence, binding, and browser save transaction

## Generated schema

| Metric | Group | Icon |
|---|---|---|
| `pv_power` | `live_power` | `mdi:solar-power` |
| `load_power` | `live_power` | `mdi:home-lightning-bolt-outline` |
| `grid_power` | `live_power` | `mdi:transmission-tower` |
| `battery_power` | `battery` | `mdi:battery-charging` |
| `p13141` | `battery` | `mdi:battery-medium` |
| `pv_to_load_power` | `live_power` | `mdi:home-import-outline` |
| `pv_to_battery_power` | `live_power` | `mdi:battery-arrow-up` |
| `pv_to_grid_power` | `live_power` | `mdi:transmission-tower-export` |
| `grid_to_load_power` | `live_power` | `mdi:transmission-tower-import` |
| `battery_to_load_power` | `live_power` | `mdi:battery-arrow-down` |
| `p13112` | `today_energy` | `mdi:white-balance-sunny` |
| `p13116` | `today_energy` | `mdi:home-lightning-bolt-outline` |
| `p13174` | `today_energy` | `mdi:battery-charging-medium` |
| `p13173` | `today_energy` | `mdi:upload-network-outline` |
| `p13147` | `today_energy` | `mdi:download-network-outline` |
| `p13199` | `energy_summary` | `mdi:home-lightning-bolt` |
| `p13029` | `energy_summary` | `mdi:battery-arrow-down-outline` |

- **REQ-SRC-001** — Every target Data Sources card MUST use type `custom:gosungrow-source-mapping-card-v1`, schema 1, validation schema 1, matcher version 3, mapping ID equal to normalized target `ps_key`, and the managed dashboard URL path. Metric group and icon metadata MUST match the table above; group display order is live power, today's energy, battery, energy summary.
- **REQ-SRC-002** — It MUST contain per-metric defaults, pinned defaults, recommendations, accepted overrides, localized metric configuration, candidates, and JSON Pointer bindings. Candidate snapshots MUST NOT contain live state values; the browser reads current Home Assistant state.
- **REQ-SRC-003** — Bindings MUST identify every matching entity string outside mapping cards. JSON Pointer escapes `~` as `~0` and `/` as `~1`. Metrics that resolve to the same entity MUST retain distinct placeholder-derived binding paths.
- **REQ-SRC-004** — Flow cards MUST receive `automatic_entities` only for fields currently equal to generated defaults so manual node sources remain distinguishable.

## Defaults and recommendations

- **REQ-SRC-005** — Existing pinned defaults take precedence across reconciliation. Otherwise a confident semantic recommendation becomes the default.
- **REQ-SRC-006** — For `p13116`, if no canonical candidate exists, use the generated entity only when it is a native canonical point and not an unsupported calculation; otherwise leave it unavailable and mark review required.
- **REQ-SRC-007** — A confident safer entity differing from a pinned default MUST be offered as a recommendation, not silently adopted. Unconfident/missing canonical verification MUST mark review when the current choice cannot be verified.
- **REQ-SRC-008** — Candidate ordering is current default first, then score descending/entity ascending. Maximum is 20 per metric: first five recommended and 15 additional. Total candidates per target MUST not exceed 340.
- **REQ-SRC-009** — Candidate confidence is high at score ≥200, medium at ≥100, low below 100; explicit manual candidates report manual confidence.
- **REQ-SRC-010** — Non-default stale candidates MUST be omitted. The default remains visible with low confidence and stale reason. Freshness is 30 minutes for non-energy and 36 hours for energy; missing timestamps count as recent, unparsable/future beyond five minutes count as stale.
- **REQ-SRC-011** — Unsupported calculated direct-solar candidates are visible only when already selected/default, labeled `calculated_legacy`, low confidence, and non-selectable.

## Override validation and persistence

- **REQ-SRC-012** — Requested overrides with unknown metric, empty entity, wrong target, wrong kind/unit, semantic conflict, or staleness MUST be rejected and logged without state values.
- **REQ-SRC-013** — A previously accepted override MUST persist when its entity disappears, or remains structurally compatible but is temporarily stale. It MUST NOT silently revert to automatic.
- **REQ-SRC-014** — A previously accepted legacy calculated `p13116` MUST persist with a blocking warning for compatibility, while remaining unsupported for new selection.
- **REQ-SRC-015** — Overrides are isolated by mapping ID even if different targets share the same generated default entity.
- **REQ-SRC-016** — Reset removes only the selected metric override and rebinds its generated/pinned default. A reset in current dashboard config MUST win over an older persisted diagnostic copy.

## Browser transaction

- **REQ-SRC-017** — Non-administrators MAY inspect sources but MUST NOT open a mutating workflow or call save APIs.
- **REQ-SRC-018** — Selecting a candidate is a preview. Mutation occurs only after `Use this source`; a warned choice requires explicit confirmation.
- **REQ-SRC-019** — Before mutation the card MUST fetch fresh Lovelace config and verify schema, mapping ID, defaults, overrides, recommendations, bindings, allowed/selectable candidate membership, and every bound path's current entity. Any mismatch is stale and MUST abort.
- **REQ-SRC-020** — Save MUST clone the dashboard, update all metric bindings and the override. Selecting the current recommendation as a first-time choice adopts it into defaults/pinned defaults and clears recommendation/review flags instead of creating a manual override.
- **REQ-SRC-021** — After save, the card MUST re-fetch and verify effective source plus every binding before changing local UI state. Read, save, or verification failure leaves local mapping unchanged and displays an error.
- **REQ-SRC-022** — Success MUST issue a Home Assistant notification and bubbling/composed `config-refresh` event. Reset follows the same transaction.

## Live warnings and accessibility

- **REQ-SRC-023** — Status priority is unavailable, needs review, manual, automatic. Warnings use live state for unavailable/non-numeric, freshness, unverified/recommended match, unsupported calculation, and physical direct-solar consistency.
- **REQ-SRC-024** — Candidate search is case-insensitive across friendly name, entity ID, device, point, source, and reason. Shared friendly-name prefixes MAY be visually shortened, but title/ARIA identity MUST preserve full name and entity ID.
- **REQ-SRC-025** — Dialog MUST support Escape close, keyboard focus trapping/restoration, minimum touch targets, scroll/focus preservation during live rerender, and mobile bottom-sheet layout.
- **REQ-SRC-026** — A fresh or unpinned dashboard MUST adopt canonical plant `pv_power` automatically. An existing manual override MUST remain unchanged. An existing pinned automatic device source MUST remain selected, be marked for review, and receive the canonical plant aggregate as its preferred recommendation; accepting it uses the verified transaction in `REQ-SRC-019`–`REQ-SRC-022` and adopts it as the new pinned default.

## Prohibited behavior

- Saving an entity merely because it was present in stale client configuration.
- Storing candidate current values in Lovelace configuration.
- Allowing one target's override to rewrite another target.
