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

## App options

| Option | Type/default | Rule |
|---|---|---|
| `gosungrow_user` | required string | fatal when empty |
| `gosungrow_password` | required password | fatal when empty |
| `mqtt_host` | string, empty | selects custom MQTT when non-empty |
| `mqtt_port` | port, 1883 | custom/service fallback |
| `mqtt_username` | string, empty | custom credentials |
| `mqtt_password` | password, empty | custom credentials |
| `install_dashboard` | bool, true | enables install/reconcile |
| `dashboard_force_update` | bool, true | passes force update |
| `dashboard_language` | string, `auto` | explicit or HA locale |
| `debug` | bool, false | verbose diagnostics |

- **REQ-ADDON-001** — If `mqtt_host` is non-empty, use only custom MQTT host/port/user/password; otherwise use Home Assistant Supervisor MQTT service values. Resolved host is required and resolved port defaults to 1883.
- **REQ-ADDON-002** — Startup MUST export resolved API, MQTT, debug, asset, and dashboard-language settings, then write CLI configuration before runtime work.
- **REQ-ADDON-003** — Startup MUST remove zero-length JSON files directly beside the runtime config, treating them as corrupt cache remnants.
- **REQ-ADDON-004** — When dashboard installation is enabled, attempt it before MQTT and continue with a warning on failure. Reconcile in a background task after sequential sleeps of 30, 90, 180, and 300 seconds (attempts therefore begin about 30, 120, 300, and 600 seconds after MQTT startup), or use the space-separated `GOSUNGROW_DASHBOARD_RECONCILE_DELAYS` override as the sequential sleep list.
- **REQ-ADDON-005** — Background reconciliation MUST be stopped when the main process exits. Each attempt uses managed defaults, selected language, and force flag when configured.

## Process restart loop

- **REQ-ADDON-006** — The wrapper MUST run `mqtt run` while teeing each attempt to a temporary log and preserving the binary exit code.
- **REQ-ADDON-007** — Panic/runtime-fatal output is non-recoverable and MUST exit without refreshing login so the underlying error remains visible.
- **REQ-ADDON-008** — A recoverable-remote failure classification MUST refresh login and restart after `min(attempt×3,30)` seconds. This class covers token/login-required, HTTP 5xx, gateway, DNS/network, connection, deadline, and I/O timeout failures except a directly classified Docker-DNS failure.
- **REQ-ADDON-009** — A Docker-DNS failure classification MUST NOT refresh login and MUST retry after 15, 30, 60, 120, then 300 seconds. A nested diagnostic mentioning `127.0.0.11:53` MUST NOT select this policy unless the stable failure classification is Docker DNS. Temporary log files MUST be removed between attempts.
- **REQ-ADDON-010** — Any unclassified nonzero exit MUST be returned without indefinite retry.
- **REQ-ADDON-011** — A normal binary error exit MUST provide the wrapper a stable classification distinguishing recoverable remote, Docker DNS, and non-recoverable failure. The wrapper MUST select recovery from that classification rather than human-readable log substrings. A missing or unknown classification is unclassified under `REQ-ADDON-010`; panic/runtime-fatal detection remains governed by `REQ-ADDON-007`.

## Prohibited behavior

- Falling back to Supervisor MQTT credentials when a custom host was explicitly selected.
- Making dashboard success a prerequisite for MQTT.
- Hiding a local Go panic behind authentication retries.
