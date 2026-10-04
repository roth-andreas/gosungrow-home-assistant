# Requirement traceability

Status: Informative  
Scope: Current implementation and verification ownership

The specification is authoritative; this map helps locate derived artifacts.

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

## Reliable plant PV aggregation

The approved contract and producer-coverage amendment are derived into normalization, independent discovery, topology validation, plant-scoped aggregation, MQTT publication diagnostics, and dashboard resolution. Specifications remain the behavioral authority. Synthetic API fixtures provide regression evidence; they do not claim verification against the reporter's live plant.

The artifacts below cover `REQ-DOM-005`, `017`; `REQ-DATA-011`, `022`–`026`; `REQ-API-031`–`032`; `REQ-MQTT-021`, `023`–`024`; `REQ-RES-007`, `019`–`020`; and `REQ-ACC-032`, `039`–`045`. User-facing behavior and diagnostic guidance are documented in `README.md` and `addon/gosungrow/DOCS.md`.

| Verification | Evidence |
|---|---|
| API decoding and normalization integration | `iSolarCloud/plant_pv_power_integration_test.go`: own device values, raw/copy provenance, native precedence, alias/conflict safety, null/invalid/zero handling, complete AC/DC coverage, and row-count independence |
| Discovery and topology integration | `iSolarCloud/plant_inventory_test.go`, `iSolarCloud/highlevel_ps_test.go`, and the decoded aggregation tests: independent metadata, intermediate nodes, missing children, cycles, and plant-local UUIDs |
| MQTT publication integration | `cmd/cmd_mqtt_plant_pv_test.go`: actual retained discovery/state serialization with transport completion/failure, filters, retry, retained-state preservation, independent rediscovery, and sorted/bounded safe diagnostics |
| Dashboard composition and compatibility | Decoded MQTT-to-dashboard template/mapping test, type-55 context and nine-producer counts, grid/phase rejection, and the existing pinned/manual upgrade and compatibility tests |

## Behavioral tests

Artifact ownership identifies executable evidence; validation results must be reported separately for each working tree.

| Test artifact | Governing requirements |
|---|---|
| `cmd/cmd_api_test.go` | `REQ-API-002`, `007`–`009`; `REQ-XCUT-002`–`004` |
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
| `cmdHassio/struct_entity_test.go` | `REQ-HA-008`–`014`, `019`–`020`; `REQ-ACC-011`, `030`, `042` |
| `cmd/cmd_mqtt_test.go` | `REQ-MQTT-005`–`024`; `REQ-DATA-018`–`026`; `REQ-API-032`; `REQ-ACC-009`–`010`, `032`, `039`–`044` |
| `cmd/cmd_ha_install_dashboard_test.go` | `REQ-DASH-001`–`037`; `REQ-RES-019`; `REQ-REL-004`, `012`–`013`; `REQ-XCUT-021`; `REQ-ACC-012`, `018`–`019`, `033`–`035`, `037`–`038`, `042` |
| `cmd/dashboard_test_helpers_test.go` | shared fixtures for `REQ-DASH-*`, `REQ-RES-*`, and `REQ-SRC-*` tests |
| `cmd/dashboard_entity_resolver_test.go` | `REQ-RES-001`–`009`, `016`–`020`; `REQ-DOM-005`, `017`; `REQ-ACC-013`, `031`, `042` |
| `cmd/dashboard_semantic_matcher_test.go` | `REQ-RES-007`, `010`–`015`, `019`–`020`; `REQ-DOM-005`; `REQ-ACC-013`, `042` |
| `cmd/dashboard_capability_pruner_test.go` | `REQ-DASH-011`–`013`; `REQ-ACC-014` |
| `cmd/dashboard_source_mapping_test.go` | `REQ-SRC-001`–`016`, `026`, `REQ-DASH-027`; `REQ-ACC-015`–`017`, `019`, `031`, `042` |
| `cmd/dashboard_i18n_test.go` | `REQ-DASH-014`–`016`; `REQ-ACC-018` |
| `tools/preview/gosungrow-source-mapping-card.test.cjs` | `REQ-SRC-017`–`025`; `REQ-CARD-001`–`002` |
| `tools/test_energy_summary_card.mjs` | `REQ-CARD-008`–`018`; `REQ-ACC-020`–`022` |
| `tools/ha-e2e/dashboard_resource_smoke.mjs`, `tools/ha-e2e/local_route_bootstrap.mjs`, `tools/ha-e2e/local_route_bootstrap.sh` | `REQ-CARD-018`; `REQ-DASH-037`; `REQ-REL-011`, `013`; `REQ-ACC-033`, `036`, `038` |
| `addon/gosungrow/recovery_policy_test.sh` | `REQ-ADDON-008`–`012`; `REQ-XCUT-019`, `021`; `REQ-ACC-028`, `038` |
| `addon/gosungrow/configuration_test.sh` | `REQ-ADDON-002`, `013`; `REQ-CFG-003`; `REQ-ACC-023` |

## Non-test verification

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
