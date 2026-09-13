# MQTT runtime

Status: Normative  
Scope: Broker connection, startup discovery, synchronization, selection, options, and outage handling

## Connection and startup

- **REQ-MQTT-001** — Broker host is required; port defaults to `1883`. Transport is unencrypted TCP. Username/password are optional at the CLI layer and used when supplied.
- **REQ-MQTT-002** — Client ID, entity prefix, and service device name MUST be `GoSungrow`; discovery prefix MUST be `homeassistant`; suggested area MUST be `Roof`.
- **REQ-MQTT-003** — MQTT operations use QoS 0, retained discovery/state publications, and a 5-second publish/subscribe timeout. The last will MUST retain `OFF` on `homeassistant/sensor/GoSungrow/state`.
- **REQ-MQTT-004** — Startup order MUST be: construct MQTT client; discover Sungrow devices with recoverable retry; build service, virtual, system, plant, and device registry hierarchy; connect broker; publish runtime options; load endpoint selection; cache point metadata.
- **REQ-MQTT-005** — Startup API discovery MUST attempt at most three times for recoverable non-Docker-DNS failures, force reauthentication between attempts, and sleep 1 then 2 seconds before later attempts. Non-recoverable and Docker-DNS errors return immediately to the app wrapper.

## Target selection and batching

- **REQ-MQTT-006** — Exactly one realtime target MUST be selected per plant from devices with a usable `ps_key`. Plant identity is device `ps_id`, or the key segment before `_` when `ps_id` is absent.
- **REQ-MQTT-007** — Selection rank is device type `14`, `11`, `7`, `1`, then all other types. Equal-ranked candidates use lexicographically smallest `ps_key`. Plants and resulting targets are sorted by `ps_id`.
- **REQ-MQTT-008** — A plant with no usable key MUST be omitted from realtime calls and produce a warning.
- **REQ-MQTT-009** — Non-realtime endpoints MUST be requested together. The realtime logical endpoint MUST be split into one batch per selected target with argument `PsKeyList:<ps_key>`. No realtime batch is made without a target.

## Synchronization

- **REQ-MQTT-010** — `mqtt run` MUST perform an immediate sync and then repeat every five minutes by default. Before non-first normal cycles it waits a default 40-second post-schedule delay. Docker-DNS retry cycles MUST omit that delay.
- **REQ-MQTT-011** — `mqtt sync` MUST perform an immediate sync and schedule singleton executions using `*/5 * * * *` by default; a supplied cron expression uses `.` as an alias for `*`.
- **REQ-MQTT-012** — Each sync MUST detect a local calendar-day transition, request configured batches, normalize/publish each result, and update `LastRefresh` only after complete success.
- **REQ-MQTT-013** — Discovery config MUST be republished on first observation, when a point value changed since the previous cycle, or on a new day. State MUST be published on every successful cycle for each eligible point.
- **REQ-MQTT-014** — A token-invalid sync MUST force login, refresh the device list, and retry the complete collection once. Other recoverable gateway errors MUST keep the service alive and wait for the next cycle.
- **REQ-MQTT-015** — Docker DNS failures after MQTT initialization MUST preserve MQTT connection and last retained values and retry after `15s`, `30s`, `60s`, `120s`, then `300s` indefinitely. A successful cycle resets the outage counter and normal five-minute schedule.
- **REQ-MQTT-016** — A non-recoverable error MUST end the runtime command with an error so the app wrapper can decide whether to restart.

## Endpoint selection

- **REQ-MQTT-017** — Endpoint configuration MUST live at `<config-dir>/mqtt_endpoints.json`; if absent, create it from the defaults in [data-normalization.md](data-normalization.md).
- **REQ-MQTT-018** — Endpoint names are processed deterministically for display. Include/exclude patterns use anchored glob-like matching where `.` is literal and `*` matches arbitrary text; a matching exclude rejects first, while a later matching include admits the point. An endpoint with no includes admits nothing.

## Runtime select entities

| ID | Label | Allowed values | Default |
|---|---|---|---|
| `loglevel` | Log Level | error, warning, info, debug | current logger level |
| `fetchschedule` | Fetch Schedule | 2m through 10m | 5m |
| `sleepdelay` | Sleep Delay After Schedule | 0s through 60s in 10s steps | 40s |
| `servicestate` | Service State | Run, Restart, Stop | Run |

- **REQ-MQTT-019** — Option matching is case-insensitive but stored/published values MUST use the configured canonical spelling. Unknown values remain unmodified for compatibility.
- **REQ-MQTT-020** — `loglevel`, `fetchschedule`, and `sleepdelay` messages MUST update runtime behavior after successful parsing. `servicestate` is currently informational and MUST NOT claim to restart or stop the process.

## Prohibited behavior

- Publishing a realtime request containing multiple plants.
- Dropping retained Home Assistant state merely because iSolarCloud is temporarily unavailable.
- Refreshing login repeatedly for Docker embedded-DNS failures.
