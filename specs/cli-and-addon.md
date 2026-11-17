# CLI and Home Assistant app

Status: Normative  
Scope: Commands, flags, app options, configuration precedence, startup, and restart policy

## CLI

- **REQ-CLI-001** — Binary name is `GoSungrow`; uncaught command errors print `ERROR: <message>` to stderr and exit nonzero.
- **REQ-CLI-002** — Supported product commands are `api login`, `mqtt run [arg]`, `mqtt sync [cron]`, `ha install-dashboard [ps_id ...]` (alias `dashboard-install`), and the framework-provided `config write`. Group commands without a valid action display help. `config write` MUST serialize the effective flag/environment configuration because app startup depends on it.
- **REQ-CLI-003** — Global API flags are `--host`, `--user/-u`, `--password/-p`, `--appkey`, and `--token-expiry`; MQTT flags are `--mqtt-host`, `--mqtt-port`, `--mqtt-user`, `--mqtt-password`.
- **REQ-CLI-004** — Dashboard flags are `--asset-dir`, `--ha-config-dir`, `--ha-ws-url`, `--supervisor-token`, `--language`, `--diagnostic-context`, `--url-path`, `--title`, `--icon`, `--show-in-sidebar`, `--require-admin`, and `--force-update`. Supervisor token, HA config directory, and diagnostic context SHOULD be hidden from normal help.
- **REQ-CLI-005** — Configuration MUST support `GOSUNGROW`-prefixed environment settings, config-file values, and flags through the CLI framework; explicit flags take precedence over persisted/default values.
- **REQ-CLI-006** — Successful login MUST update persisted token-expiry/token configuration only when the token changed.

- **REQ-CLI-007** — Boolean runtime settings `--dashboard-enabled` and `--dashboard-force-update` MUST be available to `mqtt run`, `mqtt sync`, and `config write`, with framework environment equivalents `GOSUNGROW_DASHBOARD_ENABLED` and `GOSUNGROW_DASHBOARD_FORCE_UPDATE`. Each defaults to false; precedence for these settings MUST be explicit flag, environment, persisted configuration, then default. Invalid Boolean values are configuration errors. False enablement MUST prevent automatic dashboard installation, observation, scheduled reconciliation, and availability-driven reconciliation; replacement permission alone MUST NOT enable any of them. Explicit `ha install-dashboard` remains independently available with its existing options and MUST NOT be disabled by the runtime setting. When MQTT runtime dashboard work is enabled but dashboard prerequisites are missing, the runtime MUST report an actionable dashboard failure without stopping MQTT. Existing token, asset, language, and reconciliation-delay settings MUST be reused; these runtime settings MUST NOT add app options or bypass ownership and verified-save protections.

## App options

| Option | Type/default | Rule |
|---|---|---|
| `gosungrow_user` | required string | fatal when empty |
| `gosungrow_password` | required password | fatal when empty |
| `gosungrow_host` | string, empty | empty selects the existing default gateway; otherwise selects one canonical supported gateway |
| `mqtt_host` | string, empty | selects custom MQTT when non-empty |
| `mqtt_port` | port, 1883 | custom/service fallback |
| `mqtt_username` | string, empty | custom credentials |
| `mqtt_password` | password, empty | custom credentials |
| `install_dashboard` | bool, true | enables install/reconcile |
| `dashboard_force_update` | bool, true | passes force update |
| `dashboard_language` | string, `auto` | explicit or HA locale |
| `debug` | bool, false | verbose diagnostics |

- **REQ-ADDON-001** — If `mqtt_host` is non-empty, use only custom MQTT host/port/user/password; otherwise use Home Assistant Supervisor MQTT service values. Resolved host is required and resolved port defaults to 1883.
- **REQ-ADDON-002** — Startup MUST export resolved API, MQTT, debug, asset, dashboard-language, and runtime dashboard settings, then write CLI configuration before runtime work. Existing app option `install_dashboard` MUST supply `GOSUNGROW_DASHBOARD_ENABLED` and an explicit `--dashboard-enabled` value; `dashboard_force_update` MUST supply `GOSUNGROW_DASHBOARD_FORCE_UPDATE` and an explicit `--dashboard-force-update` value. The app MUST pass both Boolean choices, including false, to `config write` and its MQTT runtime command so app configuration wins over stale persisted values. Initial dashboard installation retains its existing enable check and `ha install-dashboard` options; runtime replacement permission MUST NOT change those command semantics or enable automatic work on its own.
- **REQ-ADDON-003** — Startup MUST remove zero-length JSON files directly beside the runtime config, treating them as corrupt cache remnants.
- **REQ-ADDON-004** — When dashboard installation is enabled, attempt it before MQTT and continue with a warning on failure. Reconcile in a background task after sequential sleeps of 30, 90, 180, and 300 seconds (attempts therefore begin about 30, 120, 300, and 600 seconds after MQTT startup), or use the space-separated `GOSUNGROW_DASHBOARD_RECONCILE_DELAYS` override as the sequential sleep list. This startup schedule MUST remain intact and MUST NOT limit the lifetime availability observation required by REQ-ADDON-014.
- **REQ-ADDON-005** — Background reconciliation, availability observation, pending follow-up work, and retries MUST be stopped when the main process exits. Each attempt uses managed defaults, selected language, and force flag when configured, subject to ownership and unchanged-config protections in REQ-DASH-028 and REQ-DASH-039–040.
- **REQ-ADDON-014** — When runtime dashboard work is enabled under REQ-CLI-007 using the app's existing `install_dashboard` choice, relevant HA state and registry changes MUST drive reconciliation under REQ-DASH-039–040 for the lifetime of the main runtime, including after the startup schedule finishes. Scheduled and availability-driven work MUST share serialization and coalescing rather than run concurrent dashboard mutations. When disabled, automatic installation, observation, scheduled work, and availability-driven work MUST NOT run even if runtime force-update permission is true. Availability-driven work MUST reuse discovered target information, MUST NOT force iSolarCloud login solely because an HA entity changed, and MUST preserve MQTT operation during missing dashboard prerequisites or observation/reconciliation failure. Standalone MQTT commands MUST NOT infer enablement from a Supervisor token or an existing dashboard; explicit manual installation retains REQ-CLI-007's independent behavior.

## Process restart loop

- **REQ-ADDON-006** — The wrapper MUST run `mqtt run` while teeing each attempt to a temporary log and preserving the binary exit code.
- **REQ-ADDON-007** — Panic/runtime-fatal output is non-recoverable and MUST exit without refreshing login so the underlying error remains visible.
- **REQ-ADDON-008** — A recoverable-remote failure classification MUST refresh login and restart after `min(attempt×3,30)` seconds. This class covers token/login-required, HTTP 5xx, gateway, DNS/network, connection, deadline, and I/O timeout failures except a directly classified Docker-DNS failure.
- **REQ-ADDON-009** — A Docker-DNS failure classification MUST NOT refresh login and MUST retry after 15, 30, 60, 120, then 300 seconds. A nested diagnostic mentioning `127.0.0.11:53` MUST NOT select this policy unless the stable failure classification is Docker DNS. Temporary log files MUST be removed between attempts.
- **REQ-ADDON-010** — Any unclassified nonzero exit MUST be returned without indefinite retry.
- **REQ-ADDON-011** — A normal binary error exit MUST provide the wrapper a stable classification distinguishing recoverable remote, Docker DNS, and non-recoverable failure. The wrapper MUST select recovery from that classification rather than human-readable log substrings. A missing or unknown classification is unclassified under `REQ-ADDON-010`; panic/runtime-fatal detection remains governed by `REQ-ADDON-007`.
- **REQ-ADDON-012** — A dashboard command classified `operator_action_required` MUST remain nonfatal to MQTT startup, MUST retain the existing bounded dashboard-reconciliation schedule, and MUST report the Core-restart guidance from `REQ-DASH-037`. The wrapper MUST recognize the class but MUST NOT refresh iSolarCloud login, restart the MQTT process, initiate a Core restart, or treat an add-on restart as remediation.
- **REQ-ADDON-013** — The wrapper MUST trim `gosungrow_host`. Empty selects `https://augateway.isolarcloud.com`; otherwise the value MUST equal one of `https://augateway.isolarcloud.com`, `https://gateway.isolarcloud.com`, `https://gateway.isolarcloud.eu`, `https://gateway.isolarcloud.com.hk`, `https://gateway.isolarcloud.com.cn`, or `https://gateway.isolarcloud.in`. A valid value becomes the configured host exported and persisted before runtime work. An unsupported value is a fatal configuration error and MUST prevent dashboard and MQTT startup without changing API credentials or MQTT selection.

## Prohibited behavior

- Falling back to Supervisor MQTT credentials when a custom host was explicitly selected.
- Making dashboard success a prerequisite for MQTT.
- Hiding a local Go panic behind authentication retries.
