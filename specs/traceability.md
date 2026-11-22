# Requirement traceability

Status: Informative  
Scope: Current implementation and verification ownership

The specification is authoritative; this map helps locate derived artifacts.

## Source verification and late-availability contract

Status: implemented, with Go, Node, shell, and pinned Home Assistant Core 2026.9.3 browser evidence. Full race validation remains subject to a working CGO compiler environment; no verification against the reporter's live hardware is claimed.

| Requirements | Required follow-up evidence and ownership |
|---|---|
| `REQ-SRC-001`, `008`, `019`, `028`–`030`; `REQ-ACC-060`–`063`, `069` | Source-mapping composition and browser transaction tests: separate selection origin, verification, and availability; honest recommendations; retained unsupported identities; automatic withholding, warned manual values, incompatibility, migration, and stale-save rejection. Owners: `cmd/dashboard_source_mapping.go`, shared daily resolution, bundled source card, corresponding Go/Node tests. |
| `REQ-RES-001`–`002`, `010`, `016`, `022`; `REQ-HA-020`; `REQ-ACC-064`–`065` | MQTT discovery identity through actual HA registry ID, complete template resolution, pruning, recommendations, and bindings; renamed/suffixed canonical PV, wrong-plant and registry conflicts, ambiguity, and preserved pinned/manual choices. Owners: dashboard resolver/composition, MQTT/HA model, and existing integration tests. |
| `REQ-DASH-002`, `011`, `013`, `021`, `027`–`028`, `032`, `039`–`040`; `REQ-ADDON-004`–`005`, `014`; `REQ-ACC-065`–`066` | Deterministic lifecycle and app integration tests: entity arrival after startup retries, fixed coalescing window, serialization, retry/reconnect bounds, missed-transition recovery, unchanged-config no-op, shutdown, disabled installation, no redundant API login, ownership and MQTT isolation. Owners: dashboard lifecycle, HA observation adapter, and app orchestration. |
| `REQ-CARD-004`–`005`, `008`–`012`, `021`; `REQ-ACC-061`–`063`, `067` | Browser/card and native-fallback tests: complete versus partial flow totals, verified direct fallback, real zero, sign compatibility, manual precedence, daily presentation across current/historical periods, warnings, and no recorder mutation. Existing battery status/route cases remain regression guards. |
| `REQ-MQTT-023`, `026`; `REQ-DASH-038`, `041`; `REQ-XCUT-023`; `REQ-ACC-068`, `070` | Correlated sync/reconciliation diagnostic tests: operation-stage evidence, discovery reuse, missing/failed observation, original point metadata, reasons, stable identity/time correlation, deterministic limits/omitted counts, and redaction. No claim of HA discovery or browser display from publication alone. |

Implementation owners are `cmd/dashboard_source_policy.go`, `cmd/dashboard_native_projection.go`, `cmd/dashboard_observer.go`, `cmd/dashboard_runtime.go`, `cmd/dashboard_source_capture.go`, the existing resolver/composition and bundled cards, and `cmdHassio/publication_receipt.go`. Their Go tests, the two Node suites, app startup tests, and `tools/ha-e2e/dashboard_resource_smoke.mjs` cover the scenarios below. User guidance is updated in `README.md` and `addon/gosungrow/DOCS.md`. Existing source-inventory patterns cover these surfaces. The change does not alter aggregation arithmetic, add energy formulas or diagnostic services, delete entities, repair recorder history, or assert verification against the reporter's live hardware.

| Behavioral test | Requirements and evidence |
|---|---|
| `cmd/dashboard_source_policy_test.go` | `REQ-SRC-028`–`031`; `REQ-CARD-005`, `021`; `REQ-ACC-060`–`063`, `072`–`073`: source verification, presentation decisions, and shared serialized policy |
| `cmd/dashboard_native_projection_test.go` | `REQ-SRC-032`; `REQ-DASH-027`, `042`; `REQ-CARD-021`; `REQ-ACC-074`–`075`: restorable native projections, ownership validation, preserved layout, and regenerated bindings |
| `cmd/dashboard_observer_test.go` | `REQ-DASH-039`–`040`; `REQ-ACC-065`–`066`: relevant Home Assistant events, dedicated observation, bounded reconnect, and cancellation |
| `cmd/dashboard_runtime_test.go` | `REQ-DASH-039`–`040`; `REQ-ADDON-014`; `REQ-ACC-065`–`066`, `071`: serialized reconciliation, coalescing, retries, shutdown, and disabled installation |

### Runtime handoff and serialized-policy completion

Status: implemented. CLI framework, startup-wrapper, composition, source transaction, and native fallback/promotion tests provide coverage for the approved settings and metadata boundaries.

| Requirements | Required follow-up evidence and ownership |
|---|---|
| `REQ-CLI-007`; `REQ-CFG-003`; `REQ-ADDON-002`, `014`; `REQ-ACC-071` | CLI/configuration and app integration tests for false defaults, flag/environment/persisted precedence, explicit true/false app handoff, malformed Boolean handling, disabled-plus-force behavior, independent manual installation, and MQTT tolerance of missing dashboard prerequisites. Owners: CLI/MQTT configuration, app startup, and corresponding tests/documentation. |
| `REQ-SRC-031`; `REQ-CARD-005`, `021`; `REQ-ACC-072`–`073` | Go composition and browser tests for the shared mapping/display policy shape, target and effective-entity agreement, live availability separation, absent/invalid metadata, direct-fallback verification, and atomic policy/binding updates on selection, reset, and recommendation adoption. Owners: source mapping/composition, bundled cards, and existing Go/Node/browser tests. |
| `REQ-DASH-027`, `032`, `042`; `REQ-ACC-074` | Native-fallback and promotion tests for non-rendered mapping retention, preserved defaults/pins/overrides, fresh re-verification, rebuilt candidate/binding paths, source-only structure normalization, unrelated layout protection, and no recorder changes or cross-target leakage. Owners: dashboard lifecycle/native conversion, source extraction, and existing lifecycle/browser tests. |

### Restorable native-card projection completion

Status: implemented. Go projection/ownership tests, Node source-transaction tests, and the pinned Home Assistant browser fixture cover withholding/restoration, preserved layout and series, regenerated bindings, and rejection of invalid ownership metadata.

| Requirements | Required follow-up evidence and ownership |
|---|---|
| `REQ-SRC-032`; `REQ-CARD-021`; `REQ-DASH-027`, `042`; `REQ-ACC-075` | Go projection/ownership tests and browser source-transaction tests for paired native template and relative-path metadata; tile and multi-series graph withholding/restoration; preserved unrelated series and layout; regenerated absolute bindings; atomic verified replacement; shared-entity metric isolation; invalid/stale/cross-target/multiply owned paths; failed-save preservation; template redaction and no hidden-card/helper-entity storage or recorder mutation. Owners: source-policy composition, native rendering, structure normalization, bundled source-save transaction, and existing Go/Node/pinned-browser fixtures. |

## Runtime ownership

| Requirements | Implementation |
|---|---|
| `REQ-PROD-*`, `REQ-CLI-*` | `main.go`, `cmd/commands.go`, `cmd/cmd_api.go`, `cmd/cmd_mqtt.go`, `cmd/cmd_ha.go` |
| `REQ-DOM-*` | `iSolarCloud/api/const.go`, `iSolarCloud/api/GoStruct/valueTypes/`, `iSolarCloud/plant_pv_power.go`, `cmd/dashboard_*` |
| `REQ-API-*` | `iSolarCloud/api/web.go`, `iSolarCloud/api/crypto.go`, `iSolarCloud/recovery.go`, `iSolarCloud/struct.go`, `iSolarCloud/data.go`, `iSolarCloud/highlevel_ps.go`, endpoint packages |
| `REQ-DATA-*` | `iSolarCloud/api/GoStruct/`, `iSolarCloud/api/struct_data.go`, endpoint `data.go` files, `iSolarCloud/AppService/queryDeviceList/data.go`, `iSolarCloud/plant_pv_power.go` |
| `REQ-MQTT-*` | `cmd/cmd_mqtt.go`, `cmd/sungrow_device_preferences.go`, `cmdHassio/` |
| `REQ-HA-*` | `cmdHassio/config.go`, `struct.go`, `struct_entity.go`, `sensors.go`, `binary_sensor.go`, `select.go`, `options.go`, `funcs.go` |
| `REQ-DASH-*` | `cmd/cmd_ha_install_dashboard.go`, `cmd/dashboard_capability_pruner.go`, `cmd/dashboard_i18n.go`, `cmd/dashboard_ha_locale.go`, dashboard YAML/locales |
| `REQ-RES-*` | `cmd/dashboard_entity_resolver.go`, `cmd/dashboard_semantic_matcher.go`, `cmd/sungrow_device_preferences.go` |
| `REQ-SRC-*` | `cmd/dashboard_source_mapping.go`, bundled card JavaScript |
| `REQ-CARD-*` | `addon/gosungrow/assets/gosungrow-energy-flow-card-v2.js`, dashboard YAML |
| `REQ-ADDON-*` | `addon/gosungrow/config.yaml`, `addon/gosungrow/run.sh` |
| `REQ-XCUT-*` | recovery/client/file code across `iSolarCloud/`, `cmd/`, `cmdHassio/`, and `run.sh` |
| `REQ-REL-*` | `go.mod`, `defaults/const.go`, `repository.yaml`, Dockerfile, app manifest, `.github/workflows/homeassistant-app.yml` |
| `REQ-GOV-*` | `AGENTS.md`, `specs/`, `scripts/check_specs.py`, CI |
| `REQ-COMP-*` | `specs/completeness.md`, `specs/source-inventory.md`, `scripts/check_specs.py`, contributor review |
| `REQ-CFG-*` | CLI framework bindings, `iSolarCloud/api/web.go`, `cmd/cmd_ha_install_dashboard.go`, `addon/gosungrow/run.sh`, Dockerfile |

## Canonical daily dashboard resolution

`cmd/dashboard_daily_resolution.go` carries one canonical daily result through template composition, saved-source preservation, capability pruning, source mapping, and final diagnostics. `cmd/dashboard_entity_resolver.go`, `cmd/dashboard_semantic_matcher.go`, `cmd/dashboard_capability_pruner.go`, `cmd/dashboard_source_mapping.go`, and `cmd/cmd_ha_install_dashboard.go` derive `REQ-RES-001`, `004`, `010`, `015`, `021`–`022`; `REQ-SRC-001`, `005`–`007`, `019`, `027`; and `REQ-DASH-011`, `013`, `038`. The bundled source card verifies matcher-version equality before saving.

| Verification | Requirements and evidence |
|---|---|
| `cmd/dashboard_daily_resolution_test.go` | `REQ-ACC-052`–`057`: real API decoding and MQTT discovery/state identities through the complete dashboard template; renamed/native daily sources, zero, unsafe lifetime/export fallback, partial/absent registry, invalid/stale/ambiguous sources, saved missing/stale selections, separate bindings, and bounded final diagnostics |
| `cmd/dashboard_entity_resolver_test.go`, `cmd/dashboard_semantic_matcher_test.go`, `cmd/dashboard_capability_pruner_test.go`, `cmd/dashboard_source_mapping_test.go` | Canonical eligibility, trusted identity, daily period, scope/confidence, pruning, source recommendations, and compatibility under the updated requirements |
| `tools/preview/gosungrow-source-mapping-card.test.cjs` | `REQ-SRC-019`, `REQ-ACC-056`: matching-version recommendation adoption and selection/reset rejection when the fresh matcher version differs |

User guidance is in `README.md` and `addon/gosungrow/DOCS.md`; release changes are recorded under 3.3.7 in `CHANGELOG.md`. These synthetic cases do not claim verification of a reporter's live hardware. Existing source-inventory patterns classify both new Go artifacts.

## Reliable plant PV aggregation

The approved contract and producer-coverage amendment are derived into normalization, independent discovery, topology validation, plant-scoped aggregation, MQTT publication diagnostics, and dashboard resolution. Specifications remain the behavioral authority. Synthetic API fixtures provide regression evidence; they do not claim verification against the reporter's live plant.

The artifacts below cover `REQ-DOM-005`, `017`; `REQ-DATA-011`, `022`–`026`; `REQ-API-031`–`032`; `REQ-MQTT-021`, `023`–`024`; `REQ-RES-007`, `019`–`020`; and `REQ-ACC-032`, `039`–`045`. User-facing behavior and diagnostic guidance are documented in `README.md` and `addon/gosungrow/DOCS.md`.

| Verification | Evidence |
|---|---|
| API decoding and normalization integration | `iSolarCloud/plant_pv_power_integration_test.go`: own device values, raw/copy provenance, native precedence, alias/conflict safety, null/invalid/zero handling, complete AC/DC coverage, and row-count independence |
| Discovery and topology integration | `iSolarCloud/plant_inventory_test.go`, `iSolarCloud/highlevel_ps_test.go`, and the decoded aggregation tests: independent metadata, intermediate nodes, missing children, cycles, and plant-local UUIDs |
| MQTT publication integration | `cmd/cmd_mqtt_plant_pv_test.go`: actual retained discovery/state serialization with transport completion/failure, filters, retry, retained-state preservation, independent rediscovery, and sorted/bounded safe diagnostics |
| Dashboard composition and compatibility | Decoded MQTT-to-dashboard template/mapping test, type-55 context and nine-producer counts, grid/phase rejection, and the existing pinned/manual upgrade and compatibility tests |

## Runtime session recovery

The issue #26 requirements are derived into isolated authentication candidates, persistence before session promotion, a retained successful gateway/app-key anchor, operation-error retirement, and separate authentication and device-rediscovery obligations. Both MQTT modes use the same bounded sync cycle; cron mode serializes normal triggers and Docker-DNS deadlines and propagates fatal cycle errors. Synthetic HTTP and MQTT fixtures provide regression evidence without using a live account or the user's token store. User guidance is in `README.md` and `addon/gosungrow/DOCS.md`.

| Test artifact | Governing requirements | Evidence |
|---|---|---|
| `cmd/cmd_api_test.go` | `REQ-API-004`, `033`–`034`; `REQ-CLI-006`; `REQ-ACC-047`, `050` | Configuration persistence before promotion, restoration on write failure, unchanged-token write suppression, and reuse of the winning gateway |
| `iSolarCloud/recovery_test.go` | `REQ-API-007`, `033`–`034`; `REQ-XCUT-003`, `022`; `REQ-ACC-046`–`047`, `050`–`051` | Repeated fresh authentication, retained session/token on failed candidates, cache bypass, persistence failure, pending-auth collection guard, runtime ordering, and first-candidate DNS stop |
| `iSolarCloud/AppService/login/auth_test.go` | `REQ-API-004`–`006`, `033`; `REQ-ACC-047`, `050` | Isolated token paths, missing/corrupt token recovery, forced network authentication, deferred token/login-cache writes, and token-file replacement failure |
| `cmd/cmd_mqtt_test.go` | `REQ-API-028`, `031`–`034`; `REQ-MQTT-014`–`016`, `022`, `025`; `REQ-XCUT-003`, `022`; `REQ-ACC-046`–`051` | Actual HTTP and retained MQTT publication across failed/healthy cycles, DNS request counts and delays, bounded login/collection/endpoint replay, device/metadata retry without redundant login, nonfatal topology refresh, fatal phase propagation, and cron/DNS serialization |

Client ownership is `iSolarCloud/struct.go`, `recovery.go`, `data.go`, `api/web.go`, and `AppService/login/{auth,struct}.go`; orchestration ownership is `cmd/cmd_api.go` and `cmd/cmd_mqtt.go`. No new configuration or persisted recovery-state format is introduced. Bootstrap ordering, wire protocol, entity identity, opaque IDs, dashboard behavior, startup retry limits, and release metadata retain their existing contracts. Validation results are reported separately for the current working tree.

## Behavioral tests

Artifact ownership identifies executable evidence; validation results must be reported separately for each working tree.

| Test artifact | Governing requirements |
|---|---|
| `cmd/cmd_api_test.go` | `REQ-API-002`, `007`–`009`; `REQ-XCUT-002`–`004`; `REQ-CFG-001` |
| `cmd/failure_class_test.go` | `REQ-ADDON-008`–`012`; `REQ-XCUT-019`, `021`; `REQ-ACC-028`, `038` |
| `iSolarCloud/recovery_test.go` | `REQ-API-007`–`009`, `028`–`029`; `REQ-XCUT-002`–`004`, `017`, `019`; `REQ-ACC-002`, `029` |
| `iSolarCloud/highlevel_ps_test.go` | `REQ-API-022`–`024`, `031`–`032`; `REQ-ACC-032`, `044` |
| `iSolarCloud/api/web_timeout_test.go` | `REQ-API-001`, `016`; `REQ-ACC-003` |
| `iSolarCloud/api/web_request_metadata_test.go` | `REQ-API-012`, `014`; `REQ-CFG-004`; `REQ-ACC-025` |
| `iSolarCloud/api/struct_request_test.go` | `REQ-API-006`; `REQ-XCUT-009`–`010`; `REQ-ACC-004` |
| `iSolarCloud/AppService/login/auth_test.go` | `REQ-API-004`–`006`; `REQ-ACC-001` |
| `iSolarCloud/AppService/getPsList/funcs_test.go` | `REQ-DOM-002`–`003`; `REQ-API-022` |
| `iSolarCloud/AppService/getDeviceList/data_test.go` | `REQ-DOM-002`–`003`, `005`; `REQ-API-032` |
| `iSolarCloud/AppService/queryDeviceRealTimeDataByPsKeys/data_test.go` | `REQ-API-021`; `REQ-DOM-002`–`003` |
| `iSolarCloud/AppService/queryDeviceList/data_test.go` | `REQ-DATA-011`–`017`, `025`; `REQ-ACC-007`, `039`–`041` |
| `iSolarCloud/plant_pv_power_test.go` | `REQ-DOM-013`, `016`–`017`; `REQ-DATA-021`–`024`; `REQ-ACC-031` |
| `iSolarCloud/plant_pv_power_integration_test.go` | `REQ-DATA-011`, `021`–`026`; `REQ-API-031`–`032`; `REQ-ACC-039`–`041`, `044`–`045` |
| `iSolarCloud/plant_inventory_test.go` | `REQ-DOM-005`; `REQ-API-032`; `REQ-DATA-023`–`024`; `REQ-ACC-041` |
| `cmd/cmd_mqtt_plant_pv_test.go` | `REQ-MQTT-013`, `021`–`024`; `REQ-HA-020`; `REQ-API-032`; `REQ-RES-007`, `019`–`020`; `REQ-ACC-032`, `039`, `041`–`045` |
| `iSolarCloud/AppService/getPsDetail/timezone_test.go` | `REQ-DATA-007`; `REQ-ACC-008` |
| `iSolarCloud/api/GoStruct/valueTypes/integers_test.go` | `REQ-DATA-001`–`003`; `REQ-ACC-005` |
| `iSolarCloud/api/GoStruct/valueTypes/psid_test.go` | `REQ-DOM-002`–`003`; `REQ-ACC-005` |
| `iSolarCloud/api/GoStruct/valueTypes/uuid_test.go` | `REQ-DATA-001`–`003` |
| `iSolarCloud/api/GoStruct/valueTypes/uv_test.go` | `REQ-DATA-004`–`006`, `011`, `025`; `REQ-HA-012`; `REQ-ACC-006` |
| `iSolarCloud/api/GoStruct/valueTypes/float_test.go` | `REQ-DATA-002`; `REQ-ACC-041`; JSON parser fuzzing |
| `iSolarCloud/api/GoStruct/output/file_test.go` | `REQ-XCUT-005`–`007` |
| `cmdHassio/options_test.go` | `REQ-MQTT-019`–`020` |
| `cmdHassio/options_test.go` (discovery serialization) | `REQ-MQTT-004`, `020`; `REQ-HA-017`, `022`; `REQ-ACC-077`. Five startup payloads, stable IDs and topics, retained discovery, and shared device fields |
| `cmd/cmd_mqtt_options_e2e_test.go`, `tools/ha-e2e/mqtt_options_smoke.sh` | `REQ-HA-022`; `REQ-ACC-077`–`079`. Pinned Home Assistant Core and Mosquitto verify fresh registration and command/state round trip, retained discovery replacement, and registry identity preservation |
| `cmdHassio/struct_entity_test.go` | `REQ-HA-008`–`014`, `019`–`020`; `REQ-ACC-011`, `030`, `042` |
| `cmd/cmd_mqtt_test.go` | `REQ-MQTT-005`–`024`; `REQ-DATA-018`–`026`; `REQ-API-032`; `REQ-ACC-009`–`010`, `032`, `039`–`044`, `011`, `076`; `REQ-HA-013`, `021` (temperature discovery and state publication, retained discovery refresh, and text readings with irradiance units) |
| `cmd/cmd_ha_install_dashboard_test.go` | `REQ-DASH-001`–`037`; `REQ-RES-019`; `REQ-REL-004`, `012`–`013`; `REQ-XCUT-021`; `REQ-ACC-012`, `018`–`019`, `033`–`035`, `037`–`038`, `042` |
| `cmd/dashboard_test_helpers_test.go` | shared fixtures for `REQ-DASH-*`, `REQ-RES-*`, and `REQ-SRC-*` tests |
| `cmd/dashboard_entity_resolver_test.go` | `REQ-RES-001`–`009`, `016`–`020`; `REQ-DOM-005`, `017`; `REQ-ACC-013`, `031`, `042` |
| `cmd/dashboard_semantic_matcher_test.go` | `REQ-RES-007`, `010`–`015`, `019`–`020`; `REQ-DOM-005`; `REQ-ACC-013`, `042` |
| `cmd/dashboard_capability_pruner_test.go` | `REQ-DASH-011`–`013`; `REQ-ACC-014` |
| `cmd/dashboard_source_mapping_test.go` | `REQ-SRC-001`–`016`, `026`, `REQ-DASH-027`; `REQ-ACC-015`–`017`, `019`, `031`, `042` |
| `cmd/dashboard_i18n_test.go` | `REQ-DASH-014`–`016`; `REQ-ACC-018` |
| `tools/preview/gosungrow-source-mapping-card.test.cjs` | `REQ-SRC-017`–`025`; `REQ-CARD-001`–`002` |
| `tools/test_energy_summary_card.mjs` | `REQ-CARD-003`–`020`; `REQ-ACC-020`–`022`, `058`–`059`: battery direction, route suppression, manual values, unavailable inputs, transitions, localization, and accessible identity |
| Live Flow battery layout review (wide and compact) | `REQ-CARD-019`; `REQ-ACC-058`–`059`: browser review of caption, power, direction status, SOC, and active route labels without overlap, including all supported locales |
| `tools/ha-e2e/dashboard_resource_smoke.mjs`, `tools/ha-e2e/local_route_bootstrap.mjs`, `tools/ha-e2e/local_route_bootstrap.sh` | `REQ-CARD-018`; `REQ-DASH-037`; `REQ-REL-011`, `013`; `REQ-ACC-033`, `036`, `038` |
| `addon/gosungrow/recovery_policy_test.sh` | `REQ-ADDON-008`–`012`; `REQ-XCUT-019`, `021`; `REQ-ACC-028`, `038` |
| `addon/gosungrow/configuration_test.sh` | `REQ-ADDON-001`–`002`, `013`; `REQ-CFG-003`; `REQ-ACC-023` |

## Non-test verification

The issue #15 battery layout review exercised 48 combinations: wide/compact layouts, four supported locales, and discharge, charge, idle, balanced, unavailable, and manual-source states. Browser bounding-box checks found no overlap or clipping of the battery power, caption, or direction status. Wide discharge and compact charge screenshots were visually reviewed; generated screenshots remain under ignored `tools/preview/screenshots/issue-15/`. These synthetic checks do not claim verification against the reporter's live hardware. Current working-tree validation results are reported separately.

| Verification | Requirements |
|---|---|
| `python scripts/check_specs.py` | `REQ-GOV-003`, `006`, `010`; `REQ-REL-006`, `009` |
| `go test ./...` plus the reviewed Go diff | `REQ-PROD-*`, `REQ-DOM-*`, `REQ-API-*`, `REQ-DATA-*`, `REQ-MQTT-*`, `REQ-HA-*`, `REQ-DASH-*`, `REQ-RES-*`, `REQ-SRC-*`, `REQ-CARD-*`, `REQ-CLI-*`, `REQ-XCUT-*`, and their `REQ-ACC-*` scenarios |
| Both Node.js test files above | `REQ-SRC-017`–`025`, `REQ-CARD-*`, frontend acceptance requirements |
| `bash -n addon/gosungrow/run.sh` | `REQ-ADDON-*` |
| Docker smoke build, app-manifest review, and pinned Core lifecycle test | `REQ-REL-002`–`004`, `006`, `012`–`013` |
| source/configuration inventory checks in `scripts/check_specs.py` | `REQ-COMP-005`–`009`, `REQ-CFG-004`, `REQ-ACC-027` |
| pull-request coupling check in `scripts/check_spec_change_scope.py` | `REQ-GOV-004`, `011`–`012`; `REQ-ACC-026` |
| reviewed implementation diff | requirements without a narrower automated artifact; `REQ-COMP-001`–`004`, `007`–`010` |
