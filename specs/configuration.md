# Configuration contract

Status: Normative
Scope: Configuration sources, precedence, environment variables, and operational overrides

## Precedence and persistence

- **REQ-CFG-001** — Product configuration MUST be resolvable from framework defaults, persisted configuration, `GOSUNGROW_` environment variables, and explicit CLI flags. Explicit flags take precedence; an explicitly supplied empty value remains empty when the affected requirement distinguishes empty from absent.
- **REQ-CFG-002** — `config write` MUST persist the effective CLI-framework configuration to `GOSUNGROW_CONFIG`; when unset, the framework's standard GoSungrow configuration location is used. Credentials and tokens MUST NOT be printed by normal successful commands.
- **REQ-CFG-003** — The Home Assistant app wrapper MUST translate app options into the configuration described by `REQ-ADDON-*`; Supervisor-provided service values are inputs only to that translation and MUST NOT redefine CLI precedence.

## Configuration catalog

The implementation MUST support this complete set of repository-specific environment inputs:

| Input | Meaning and exact special behavior |
|---|---|
| `GOSUNGROW_HOST`, `GOSUNGROW_USER`, `GOSUNGROW_PASSWORD`, `GOSUNGROW_APPKEY` | iSolarCloud endpoint and credentials corresponding to the global API flags. |
| `GOSUNGROW_TOKEN_EXPIRY` | Persisted last-login/token-expiry value corresponding to `--token-expiry`; successful login updates it only under `REQ-CLI-006`. |
| `GOSUNGROW_MQTT_HOST`, `GOSUNGROW_MQTT_PORT`, `GOSUNGROW_MQTT_USER`, `GOSUNGROW_MQTT_PASSWORD` | MQTT endpoint and credentials; port defaults to `1883`. |
| `GOSUNGROW_CONFIG` | Persisted CLI configuration path and runtime cache/config parent. |
| `GOSUNGROW_ASSET_DIR` | Managed dashboard asset directory; the app image default is `/opt/gosungrow/assets`. |
| `GOSUNGROW_DASHBOARD_LANGUAGE` | Dashboard language passed by app startup; absent/empty becomes `auto`. |
| `GOSUNGROW_DASHBOARD_RECONCILE_DELAYS` | Space-separated sequential delay values used by `REQ-ADDON-004`; default is `30 90 180 300`. |
| `GOSUNGROW_DEBUG` | Dashboard diagnostic mode is enabled after trim and case-fold only for `1`, `true`, `yes`, or `on`; app startup derives it from the `debug` option. |
| `GOSUNGROW_DID` | Trimmed browser device identifier; absent/empty generates a cryptographically random 16-digit value for the request. |
| `GOSUNGROW_LIMIT_OBJ` | Optional encrypted request limit. Absence uses endpoint behavior; `__EMPTY__` means an explicitly empty encrypted value; any other present string is encrypted verbatim. |
| `GOSUNGROW_TRACE_HTTP` | Any non-empty value enables encrypted request/response HTTP trace output. It MUST NOT be enabled by default. |
| `GOSUNGROW_TIMESTAMP_OFFSET_MS` | Optional signed base-10 milliseconds added to the request clock after the learned server offset. Whitespace is ignored around the value; absent, empty, or invalid values add zero. This is a diagnostic/compatibility override, not persisted server state. |
| `SUPERVISOR_TOKEN` | Default Home Assistant Supervisor WebSocket token for managed dashboard installation; an explicit dashboard flag may override it. |

- **REQ-CFG-004** — The catalog above MUST remain exhaustive for repository-owned environment lookups. Adding, renaming, or removing one requires a same-change update to this table, affected requirements, validation, and user documentation.
- **REQ-CFG-005** — Secret-bearing values are host API credentials, MQTT credentials, Supervisor token, access token, and persisted login material. Logs, validation output, and error messages MUST redact or omit them as required by `REQ-XCUT-009` and `REQ-XCUT-010`.
- **REQ-CFG-006** — Diagnostic overrides MUST be opt-in and MUST NOT weaken authentication, encryption, identifier opacity, or persisted-secret handling.

## Prohibited behavior

- Creating an undocumented environment variable, config key, or precedence exception.
- Treating an absent value and an explicitly empty value as equivalent where this specification distinguishes them.
- Enabling credential or encrypted-body traces by default.
