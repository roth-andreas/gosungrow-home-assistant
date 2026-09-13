# Migrated behavior inventory

Status: Informative  
Scope: Classification of behavior found during the initial source-to-spec migration

Every tracked runtime surface was reviewed. Normative destinations are listed in [traceability.md](traceability.md).

| Observed area | Classification | Resolution |
|---|---|---|
| Home Assistant app as primary deployment | intended | `REQ-PROD-*`, `REQ-ADDON-*` |
| CLI API/MQTT/dashboard commands | intended support | `REQ-CLI-*` |
| Browser-compatible encrypted iSolarCloud protocol | required compatibility | `REQ-API-010`–`017` |
| Legacy hosts/app keys | required compatibility | `REQ-API-002`, `007`–`009` |
| Generic Go reflection engine | replaceable implementation | behavioral output captured by `REQ-DATA-008`–`010` |
| Full raw response structs | transport schema implementation | known logic fields specified; unknown fields preserved by `REQ-API-024`, `REQ-XCUT-015` |
| Composite plant IDs | intended compatibility | `REQ-DOM-002`, `REQ-ACC-005` |
| Missing/corrupt caches as stale | intended resilience | `REQ-API-005`, `026`; `REQ-XCUT-007` |
| Virtual ESS formulas | intended compatibility, some legacy | `REQ-DATA-011`–`017` |
| Calculated direct-solar entities | unsafe legacy compatibility | retained but blocked for new dashboard selection by `REQ-DATA-016`, `REQ-SRC-011`, `014` |
| Zero-valued battery-to-grid/grid-to-battery placeholders | legacy compatibility | explicitly specified in virtual power table; dashboard may prune/unresolve them |
| MQTT service-state select without process effect | informational legacy | `REQ-MQTT-020`; no false claim of control |
| MQTT option unknown-value preservation | compatibility | `REQ-MQTT-019` |
| Exact dashboard matcher scores | compatibility algorithm | `REQ-RES-004`–`014` |
| Managed dashboard full and structure hashes | intended ownership protection | `REQ-DASH-023`–`028` |
| Embedded data-URL card resource | intended deployment reliability | `REQ-DASH-017`–`019` |
| Four dashboard locales | intended | `REQ-DASH-014`–`016`, `REQ-CARD-016`–`017` |
| CSS, SVG coordinates, internal Go type names | replaceable implementation | deliberately not normative |
| Repository-specific environment inputs | intended configuration contract | exhaustively classified by `REQ-CFG-*` |
| Tracked implementation and automation files | derived surfaces | exhaustively classified in `source-inventory.md` |
| Preview screenshots and `notgit/` | development/personal artifact | outside product scope |
| Historical changelog behavior superseded by current code | historical | not normative unless present in specs |

No unresolved behavior was promoted with a `TODO` marker. Future disputes are handled by `REQ-GOV-007`.
