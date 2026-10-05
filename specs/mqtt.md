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
- **REQ-MQTT-014** — A token-invalid collection failure MUST establish an authentication-recovery obligation. MQTT MUST initiate at most one forced-login sequence per sync cycle for token-invalid or pending authentication recovery. If that sequence succeeds, MQTT MUST refresh device inventory and attempt plant-topology refresh before collection. A token-invalid failure discovered during collection MAY cause exactly one complete collection retry after successful recovery when the cycle's forced-login opportunity has not already been used. Otherwise collection MUST defer to the next permitted cycle. Recoverable failures during login, rediscovery, or retried collection MUST preserve the live MQTT process and connection and follow the applicable next-cycle policy. Non-recoverable failures MUST follow REQ-MQTT-016. Endpoint-level recovery retains its separate bound under REQ-API-028.
- **REQ-MQTT-015** — Docker DNS failures after MQTT initialization MUST preserve MQTT connection and last retained values and retry after `15s`, `30s`, `60s`, `120s`, then `300s` indefinitely. A successful cycle resets the outage counter and normal five-minute schedule.
- **REQ-MQTT-016** — A non-recoverable error MUST end the runtime command with an error so the app wrapper can decide whether to restart.

- **REQ-MQTT-025** — Before each scheduled sync attempt, previous recoverable operation errors MUST cease to block endpoint lookup or execution. This reset MUST preserve pending authentication or device-rediscovery obligations. Pending authentication MUST be attempted before collection, using the recovery anchor and candidate order under REQ-API-007 and REQ-API-034. A recoverable authentication failure MUST end that collection attempt and retain the obligation for the next retry permitted by REQ-MQTT-010, REQ-MQTT-011, and REQ-MQTT-015. Successful MQTT-initiated forced authentication MUST establish a device-rediscovery obligation. Recoverable device-rediscovery failure MUST retain that obligation for the next permitted attempt without requiring another login solely because rediscovery failed. Successful device rediscovery clears that obligation; topology refresh remains nonfatal under REQ-MQTT-022. These rules apply to both `mqtt run` and `mqtt sync`.

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
- **REQ-MQTT-021** — Each synchronization attempt MUST evaluate plant PV aggregation at most once per plant, after normalizing its plant-scoped query-device snapshot and before publication filtering. Device-sum completeness is governed by `REQ-DATA-023`–`024`. A selected-device realtime response MUST NOT substitute for that snapshot. Successful eligible aggregates MUST publish state through the normal retained pipeline, with discovery governed by `REQ-MQTT-013`; at most one canonical discovery/state publication per plant is permitted per attempt. A permitted collection replay constitutes another attempt. Suppression MUST publish no fabricated replacement or retained-state deletion. A previously retained state may remain and becomes stale under the 30-minute live-source rule.
- **REQ-MQTT-022** — Plant topology acquisition failure or invalid topology MUST be nonfatal, suppress only device-derived PV aggregation for the affected plant, and emit at most one warning per plant per topology refresh attempt. Other plants, entities, and synchronization MUST continue. A usable native plant total remains publishable without topology. Token-triggered device rediscovery MUST also refresh topology.
- **REQ-MQTT-023** — Each aggregation attempt MUST emit one info-level final summary beginning `Plant PV aggregation:`. Required fields are `ps_id`, synchronization cycle, outcome, source, expected producer count, received producer count, valid AC count, valid DC count, and reason. An unavailable expected count MUST be reported as unknown, not zero. Outcomes are `published`, `suppressed`, `filtered`, or `publication_failed`. Sources are `native_plant`, `summed_device_ac`, `summed_device_dc`, or `none`. Published success requires the applicable publication operations to complete.
- **REQ-MQTT-024** — Aggregation diagnostic reasons, counts, and debug detail MUST follow these rules:

  - Successful publication has reason `none`. Device-derived suppression reports the first applicable blocker in this order: `inventory_unavailable`, `inventory_conflict`, `topology_unavailable`, `topology_invalid`, `no_producers`, `conflicting_points`, `missing_contributors`, `incompatible_units`, `invalid_values`, `incomplete_basis`. A valid native plant total bypasses inventory and topology blockers.
  - A filtered result has reason `endpoint_filter` or `unknown_parent`; a publication failure has reason `publication_failed`.
  - The expected producer count is the number of expected producer leaves. The received count is the number of those leaves with recognized measurements in the current snapshot, including invalid measurements. Valid AC and DC counts include only leaves with a valid, compatible measurement of the respective basis.
  - Debug detail MUST expose sorted device keys, device types, relevant original point IDs, units, validity, and rejection reasons. Detail MUST be bounded to at most 100 power-point records per plant per attempt and include the omitted-record count.
  - These diagnostics MUST NOT include credentials, tokens, raw requests or responses, serial numbers, or numeric measurement values. HTTP tracing MUST NOT be required. Existing topology-refresh warning limits remain unchanged.

## Prohibited behavior

- Publishing a realtime request containing multiple plants.
- Dropping retained Home Assistant state merely because iSolarCloud is temporarily unavailable.
- Starting authentication recovery or rotating gateways solely because of a directly classified Docker-DNS endpoint failure. Resuming authentication that was already pending remains governed by REQ-XCUT-003 and REQ-MQTT-025.
