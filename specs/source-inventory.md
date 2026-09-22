# Source and artifact inventory

Status: Informative
Scope: Classification and normative ownership of every tracked repository surface

The patterns are repository-relative and are checked by `scripts/check_specs.py`. A file may match more than one row; at least one match is required. Generated and third-party files remain derived artifacts, never authorities.

| Tracked path pattern | Classification | Normative owner |
|---|---|---|
| `main.go` | product entry point | `REQ-PROD-*`, `REQ-CLI-*` |
| `cmd/*.go` | CLI, orchestration, dashboard behavior | `REQ-CLI-*`, `REQ-MQTT-*`, `REQ-DASH-*`, `REQ-RES-*`, `REQ-SRC-*` |
| `cmdHassio/*.go` | MQTT and Home Assistant model | `REQ-MQTT-*`, `REQ-HA-*` |
| `iSolarCloud/*.go` | client lifecycle, aggregation, recovery | `REQ-PROD-*`, `REQ-API-*`, `REQ-XCUT-*` |
| `iSolarCloud/api/**/*.go` | protocol, schemas, normalization, output | `REQ-API-*`, `REQ-DATA-*`, `REQ-XCUT-*` |
| `iSolarCloud/AppService/**/*.go` | app-service endpoints and conversion | `REQ-API-*`, `REQ-DATA-*`, `REQ-DOM-*` |
| `iSolarCloud/WebAppService/**/*.go` | web-app endpoints and conversion | `REQ-API-*`, `REQ-DATA-*` |
| `iSolarCloud/WebIscmAppService/**/*.go` | plant-tree endpoint and conversion | `REQ-API-*`, `REQ-DATA-*` |
| `iSolarCloud/Common/*.go` | shared product assets | `REQ-DATA-*`, `REQ-HA-*` |
| `iSolarCloud/NullArea/**/*.go` | null/test endpoint contract | `REQ-API-*`, `REQ-DATA-*` |
| `defaults/*.go` | version and product defaults | `REQ-PROD-*`, `REQ-REL-*` |
| `defaults/*.md` | non-normative configuration examples | `REQ-GOV-008`, `REQ-CFG-*` |
| `addon/gosungrow/run.sh` | app lifecycle and recovery | `REQ-ADDON-*`, `REQ-CFG-*`, `REQ-XCUT-*` |
| `addon/gosungrow/configuration.sh` | app option validation and translation policy | `REQ-ADDON-*`, `REQ-CFG-*` |
| `addon/gosungrow/configuration_test.sh` | app configuration verification | `REQ-ADDON-*`, `REQ-CFG-*`, `REQ-ACC-*` |
| `addon/gosungrow/recovery_policy.sh` | app process-boundary failure classification policy | `REQ-ADDON-*`, `REQ-XCUT-*` |
| `addon/gosungrow/recovery_policy_test.sh` | app recovery-policy verification | `REQ-ADDON-*`, `REQ-XCUT-*`, `REQ-ACC-*` |
| `addon/gosungrow/config.yaml` | app options and manifest | `REQ-ADDON-*`, `REQ-REL-*` |
| `addon/gosungrow/Dockerfile` | container build/runtime | `REQ-REL-*` |
| `addon/gosungrow/*.md` | user-facing app documentation | `REQ-GOV-008`, `REQ-ADDON-*` |
| `addon/gosungrow/assets/**` | managed dashboard, cards, locales | `REQ-DASH-*`, `REQ-SRC-*`, `REQ-CARD-*` |
| `addon/gosungrow/translations/**` | app option localization | `REQ-ADDON-*`, `REQ-COMP-008` |
| `examples/**` | non-normative user examples | `REQ-HA-*`, `REQ-CARD-*` |
| `tools/preview/*.js` | local preview implementation | `REQ-CARD-*`, `REQ-SRC-*` |
| `tools/preview/*.html` | local preview shell | `REQ-CARD-*`, `REQ-SRC-*` |
| `tools/preview/*.cjs` | frontend verification | `REQ-CARD-*`, `REQ-SRC-*` |
| `tools/*.mjs` | frontend verification | `REQ-CARD-*` |
| `tools/ha-e2e/**` | pinned Home Assistant browser integration fixtures and verification | `REQ-CARD-*`, `REQ-DASH-*`, `REQ-REL-*`, `REQ-ACC-*` |
| `.github/workflows/*.yml` | validation and publishing | `REQ-REL-*`, `REQ-GOV-*` |
| `.github/assets/**` | user-facing illustrative screenshots | `REQ-GOV-008`; informative only |
| `.agents/skills/**` | assistant change-control workflow | `REQ-GOV-*`, `REQ-COMP-*` |
| `scripts/*.py` | deterministic repository validation | `REQ-GOV-*`, `REQ-COMP-*`, `REQ-REL-*` |
| `scripts/*.ps1` | local preview/layout tooling | `REQ-DASH-*`, `REQ-CARD-*` |
| `specs/**` | normative contract, decisions, inventories | `REQ-GOV-*`, `REQ-COMP-*` |
| `docs/**` | non-normative engineering guidance | `REQ-GOV-008`, `REQ-REL-*` |
| `AGENTS.md` | contributor/assistant entry contract | `REQ-GOV-*` |
| `README.md` | user-facing summary | `REQ-GOV-008` |
| `*.md` | root contributor or historical documentation | `REQ-GOV-008` |
| `go.mod` | language/dependency declaration | `REQ-REL-*` |
| `go.sum` | dependency integrity | `REQ-REL-*` |
| `repository.yaml` | Home Assistant repository metadata | `REQ-REL-*` |
| `.dockerignore` | container build context policy | `REQ-REL-*` |
| `.gitignore` | local/generated artifact boundary | `REQ-COMP-*` |
| `.gitattributes` | repository text normalization | `REQ-REL-*` |
| `LICENSE` | legal metadata, no product logic | not applicable |

Ignored files, local screenshots, private Home Assistant fixtures, Git internals, and editor metadata are outside the tracked product surface.
