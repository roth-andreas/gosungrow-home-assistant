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

- **REQ-SRC-001** — Every target Data Sources card MUST use type `custom:gosungrow-source-mapping-card-v1`, schema 1, validation schema 2, matcher version 5, the target's mapping ID, and the managed dashboard URL path. To derive the mapping ID, trim leading and trailing whitespace from the opaque `ps_id` and `ps_key`, preserving case and all internal characters. Join their UTF-8 bytes in that order with one `0x00` separator, compute SHA-256, and prefix `source-` to the 12 lowercase hexadecimal characters encoding the first six digest bytes. The enhanced Data Sources card, native notice under REQ-DASH-042, and `source_policy.mapping_id` under REQ-SRC-031 MUST use the same ID for the target. Metric group and icon metadata MUST match the table above. Group display order is live power, today's energy, battery, energy summary. Reconciliation MUST migrate older managed mapping configurations without discarding saved defaults, pinned defaults, or accepted overrides. Missing verification metadata MUST NOT imply verified meaning. Migration MUST preserve existing source identities and re-evaluate their presentation under REQ-SRC-028–030 without silently selecting replacements. Current configuration and reset precedence remain governed by REQ-SRC-016.
- **REQ-SRC-002** — It MUST contain per-metric defaults, pinned defaults, recommendations, accepted overrides, localized metric configuration, candidates, and JSON Pointer bindings. Candidate snapshots MUST NOT contain live state values; the browser reads current Home Assistant state.
- **REQ-SRC-003** — Bindings MUST identify every matching entity string outside mapping cards. JSON Pointer escapes `~` as `~0` and `/` as `~1`. Metrics that resolve to the same entity MUST retain distinct placeholder-derived binding paths.
- **REQ-SRC-004** — Flow cards MUST receive `automatic_entities` only for fields currently equal to generated defaults so manual node sources remain distinguishable.

## Defaults and recommendations

- **REQ-SRC-005** — Existing nonempty pinned defaults MUST take precedence across reconciliation. Otherwise a confident semantic recommendation becomes the default. For canonical daily metrics, that recommendation MUST satisfy REQ-RES-021. An unresolved automatic source MUST NOT acquire a default from legacy matching. Existing accepted overrides retain their specified precedence and preservation rules.
- **REQ-SRC-006** — A new automatic `p13116` default MUST be a confident eligible native canonical source under REQ-RES-021. Without one, the automatic metric remains unresolved; unsupported calculations MUST NOT supply its default. Existing pinned or previously accepted legacy selections remain governed by REQ-SRC-007, 011, 013, and 014.
- **REQ-SRC-007** — A confident safer entity differing from a pinned default MUST be offered as a recommendation, not silently adopted. Unconfident/missing canonical verification MUST mark review when the current choice cannot be verified. A preserved canonical-daily selection that cannot be verified MUST retain its identity and be marked for review, even when no generated automatic source exists.
- **REQ-SRC-008** — Candidate ordering is current default first, then score descending/entity ascending. A recommendation MUST be eligible, selectable, and confident under the applicable resolution contract; list position and preservation alone MUST NOT establish recommendation status. At most five eligible candidates MAY be recommended. A preserved unverified or incompatible selection MUST appear separately as the current selection needing review, never as a recommendation. With no eligible confident source, the recommendation group MUST state that none is available. Maximum is 20 candidates per metric, including the preserved selection and recommended candidates; total candidates per target MUST not exceed 340.
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
- **REQ-SRC-019** — Before mutation the card MUST fetch fresh Lovelace config and verify schema, validation schema, mapping ID, defaults, overrides, recommendations, bindings, allowed/selectable candidate membership, selection origin, semantic verification, display decision, and every bound path's current entity. Any mismatch is stale and MUST abort. Before mutation, the card MUST also verify that the fresh dashboard matcher version equals its loaded configuration's matcher version. A client lacking validation schema 2 or the current verification metadata MUST NOT save a migrated mapping. Live availability MUST be rechecked rather than trusted from a saved snapshot.
- **REQ-SRC-020** — Save MUST clone the dashboard, update all metric bindings and the override. Selecting the current recommendation as a first-time choice adopts it into defaults/pinned defaults and clears recommendation/review flags instead of creating a manual override.
- **REQ-SRC-021** — After save, the card MUST re-fetch and verify effective source plus every binding before changing local UI state. Read, save, or verification failure leaves local mapping unchanged and displays an error.
- **REQ-SRC-022** — Success MUST issue a Home Assistant notification and bubbling/composed `config-refresh` event. Reset follows the same transaction.

## Live warnings and accessibility

- **REQ-SRC-023** — Status priority is unavailable, needs review, manual, automatic. Warnings use live state for unavailable/non-numeric, freshness, unverified/recommended match, unsupported calculation, and physical direct-solar consistency. A withheld daily value MUST show its review/unavailable reason rather than a numeric metric value; an otherwise usable unverified manual selection MUST retain its persistent warning under REQ-SRC-029 even though its identity is explicitly selected.
- **REQ-SRC-024** — Candidate search is case-insensitive across friendly name, entity ID, device, point, source, and reason. Shared friendly-name prefixes MAY be visually shortened, but title/ARIA identity MUST preserve full name and entity ID.
- **REQ-SRC-025** — Dialog MUST support Escape close, keyboard focus trapping/restoration, minimum touch targets, scroll/focus preservation during live rerender, and mobile bottom-sheet layout.
- **REQ-SRC-026** — A fresh or unpinned dashboard MUST adopt canonical plant `pv_power` automatically. An existing manual override MUST remain unchanged. An existing pinned automatic device source MUST remain selected, be marked for review, and receive the canonical plant aggregate as its preferred recommendation; accepting it uses the verified transaction in `REQ-SRC-019`–`REQ-SRC-022` and adopts it as the new pinned default.

## Complete source composition

- **REQ-SRC-027** — A known target metric MUST NOT be treated as unknown solely because legacy matching or automatic resolution produced no entity. A final confident canonical source or saved selection requiring preservation MUST retain its source-mapping entry, applicable candidates, and metric-specific bindings. Pruning-induced changes to JSON Pointer positions MUST preserve the original target and metric ownership.

## Selection and presentation safety

- **REQ-SRC-028** — Per-metric configuration MUST distinguish selection origin (`automatic`, including pinned defaults, or `manual` for an accepted explicit override), semantic verification (`verified`, `unverified`, or `incompatible`), and a display decision (`normal`, `warned`, or `withheld`), with an explanatory reason. An absent selection MUST remain distinguishable from a saved selection. Live availability (`available`, `unavailable`, or `stale`) MUST be evaluated separately using current Home Assistant state and REQ-SRC-010. Ranking confidence, a saved identity, and a numeric state MUST NOT by themselves establish verified meaning. Verified meaning requires the applicable trusted identity, target, kind, period, direction, and scope contract. Incompatibility requires trusted evidence that one of those semantics or units conflicts with the metric; an HA friendly name, a large value, or the unit `MWh` alone MUST NOT establish that conflict. Missing registry evidence is unverified, not proof of incompatibility. Existing staleness rules remain applicable and MUST NOT turn temporary staleness into semantic incompatibility.
- **REQ-SRC-029** — Canonical daily presentation MUST preserve saved selection identity independently of whether its value may be displayed. A verified usable source is displayed normally. An unverified automatic selection MUST be withheld; an otherwise usable unverified explicit manual selection MUST remain displayed with a persistent warning. A proven incompatible selection MUST be withheld regardless of selection origin. A missing or invalid live value MUST be unavailable, never zero. Withholding MUST retain an actionable Data Sources review entry, selected entity identity, and reason, without promoting a replacement or changing an accepted override. Existing stricter restrictions on unsupported calculated direct-solar sources remain applicable. The same decision MUST govern enhanced cards, native fallback, summaries, charts, and tooltips under REQ-CARD-021.
- **REQ-SRC-030** — Recommendation eligibility, selection origin, semantic verification, reasons, and display decisions MUST survive generation, reconciliation, source-save verification, and native/enhanced transitions consistently. A schema migration MUST re-evaluate saved selections without altering their identities or recorder data. New source-policy metadata MUST be excluded from external-structure comparison under REQ-DASH-027. Verified working mappings and explicitly selected compatible manual sources MUST NOT be remapped by this change. Browser warnings and unavailable/review states MUST support all locales and accessible identity under REQ-CARD-016–017 and REQ-SRC-024–025.

## Serialized source policy

- **REQ-SRC-031** — Generated mapping metric records MUST use the fields in the following table for the shared selection/presentation decision. Their canonical metric identity remains the existing `key`; `mapping_id` remains the target identity on the enclosing mapping card. Display-card metadata MUST use the same decision record under `source_policy.mapping_id` and `source_policy.metrics`, where `metrics` is an object keyed by canonical metric ID, as shown below. The record's `entity_id` MUST agree with the effective selected default/accepted override and applicable display bindings; the `mapping_id` MUST agree with the target. Source policy MUST NOT choose a different entity implicitly or cross target boundaries. Live availability and freshness MUST be evaluated from current HA state under REQ-SRC-010, not persisted as permission to display. Missing, malformed, or identity-mismatched policy MUST NOT imply verified meaning; invalid policy MUST block source mutation until refreshed, while saved selection identity and the presentation rules in REQ-SRC-028–030 remain applicable. Selection, reset, and recommendation-adoption transactions MUST update and verify the policy records together with all affected entity bindings under REQ-SRC-019–021. Policy metadata MUST NOT contain numeric measurement values, credentials, or raw payloads.

| Field | Required meaning |
|---|---|
| `entity_id` | Effective selected HA entity ID; empty string when no source exists. |
| `selection_origin` | `automatic` or `manual`, consistent with the saved default/accepted override under REQ-SRC-028. |
| `verification` | `verified`, `unverified`, or `incompatible` under REQ-SRC-028. |
| `display_decision` | `normal`, `warned`, or `withheld` under REQ-SRC-029. |
| `reason` | Explanation of the verification/display decision, not a ranking score or a measurement value. |

The display-card wire shape MUST follow this structure; bracketed values stand for the applicable identities and explanation:

```yaml
source_policy:
  mapping_id: <target mapping ID>
  metrics:
    p13199:
      entity_id: <selected entity ID>
      selection_origin: automatic
      verification: unverified
      display_decision: withheld
      reason: <explanation>
```

## Restorable native-card projections

- **REQ-SRC-032** — A native display card whose rendered structure changes when a metric is withheld MUST retain restorable presentation metadata in the optional paired fields `source_policy.native_template` and `source_policy.native_bindings`. Both fields MUST be present together when this projection is used and MUST satisfy the following contract:

  - `native_template` MUST be an object containing one supported generated native card's restorable configuration before source withholding, including its non-source properties and layout. It MUST NOT contain live measurement values, credentials, raw payloads, embedded `source_policy` or `gosungrow_source_mapping` objects, or recursively retained projection templates. Static configuration numbers, such as graph duration or layout dimensions, are not measurement values. The template MUST remain non-rendered metadata, not a hidden card used as storage or a measurement-producing entity.
  - `native_bindings` MUST be an object keyed by canonical metric ID. Each value MUST be an array of JSON Pointer strings relative to the root of `native_template`, using REQ-SRC-003's escaping rules. Every path MUST resolve to an entity-reference string owned by that metric in that template, consistent with `source_policy.mapping_id`, `source_policy.metrics`, and the effective saved selection. A path MUST NOT be owned by multiple metrics or address another card, target, or surrounding dashboard. Distinct owned paths MAY contain the same entity ID without merging their metric ownership.
  - Rendering MUST derive the native projection from the template, owned paths, and current source-policy decisions. Withholding a metric MUST remove only its numeric presentation and replace it with the applicable review/unavailable explanation; other permitted series, non-source card properties, and layout MUST remain intact. Later permission to display the metric MUST restore its presentation from the same template, not guess the former card structure or substitute a different source. The template's owned entity references MUST be rebound consistently when the selected source changes; its unrelated configuration MUST NOT be replaced by that operation.
  - Source selection, reset, and recommendation adoption MUST rebuild the affected native projection and its final rendered-dashboard binding paths in the same cloned configuration transaction as the selection and policy update. Template-relative paths MUST NOT be confused with absolute rendered-dashboard paths or reused as obsolete positions after filtering changes an array. REQ-SRC-019–021 verification MUST include the retained template and owned paths, resulting native projection, policy records, and regenerated bindings before reporting success. Invalid, missing-pair, stale, mismatched, or unresolved template/path metadata MUST block mutation until refreshed rather than authorize an unsafe reconstruction.
  - This metadata MUST NOT grant source eligibility, alter recorder data, or add helper entities, a custom native card, a state file, or a service. It MUST remain scoped to the one native card and MUST NOT be dumped in diagnostics. Missing projection metadata in older configurations MUST be rebuilt through fresh managed reconciliation before a source transaction depends on it. Ownership comparison MUST retain evidence of the template's non-source structure and unrelated rendered layout edits under REQ-DASH-027.

## Prohibited behavior

- Saving an entity merely because it was present in stale client configuration.
- Storing candidate current values in Lovelace configuration.
- Allowing one target's override to rewrite another target.
