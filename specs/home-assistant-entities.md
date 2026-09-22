# Home Assistant entity contract

Status: Normative  
Scope: MQTT discovery payloads, IDs, device hierarchy, metadata, and state payloads

## Identity and topics

- **REQ-HA-001** — Topic segments are joined with `/`; spaces and `:` inside segments become `_`. IDs join non-empty trimmed components with `-`, replacing runs of `/`, space, `:`, or `.` with `-`.
- **REQ-HA-002** — Sensor discovery/state topics MUST be `homeassistant/sensor/GoSungrow/<id>/config|state`; binary sensors use `binary_sensor`; select options use `select` and add `cmd`.
- **REQ-HA-003** — Entity object ID and unique ID MUST be the same stable ID derived from `GoSungrow` plus the complete normalized endpoint path. Entity identity MUST NOT depend on localized display names.
- **REQ-HA-004** — Discovery payloads and state payloads MUST be retained at QoS 0. Sensors and binary sensors MUST set force-update true, enabled-by-default true, UTF-8 encoding, and no expiry.

## Device hierarchy

- **REQ-HA-005** — The service device identifier is `GoSungrow`; software version text MUST identify the binary and source repository; manufacturer is the project maintainer for the service and endpoint devices.
- **REQ-HA-006** — Each plant and device MUST have stable identifiers derived from its opaque Sungrow ID. Device registry connections MUST express service → plant → device topology; a device whose plant ID equals its key attaches to the service.
- **REQ-HA-007** — An entity with an unknown parent device MUST be ignored and logged rather than published under an invented device.

## Entity classification and templates

- **REQ-HA-008** — Boolean point/value, or unit marker `binary_sensor`, `binary`, `Bool`, or `valueTypes.Bool`, MUST produce a binary sensor and not a regular sensor. Unit `select` produces a select; all other valid values produce a sensor.
- **REQ-HA-009** — State payload is JSON `{ "value": <printable-value> }`; total sensors also include `last_reset`. A value that is already JSON-like MAY be published directly for compatibility. `--` becomes an empty value.
- **REQ-HA-010** — Default templates are `value_json.value` for booleans/strings, with `float` for floats, `int` for integers, and `as_datetime` for successfully parsed date-times. A configured alternate value field replaces `value`.
- **REQ-HA-011** — Invalid floats MUST suppress state update. Date-times MUST be converted to local full-date-time form before publication when parseable.

## Measurement metadata

| Value/unit | Device class | Default icon |
|---|---|---|
| boolean | `power` | `mdi:check-circle-outline` |
| W/kW/MW or unitless power | `power` | `mdi:lightning-bolt` |
| kWp | none | `mdi:lightning-bolt` |
| Wh/kWh/MWh or energy | `energy` | `mdi:transmission-tower` |
| var/kvar | `reactive_power` | `mdi:lightning-bolt` |
| VA | `apparent_power` | `mdi:lightning-bolt` |
| Hz | `frequency` | `mdi:sine-wave` |
| V | `voltage` | `mdi:current-dc` |
| A | `current` | `mdi:current-ac` |
| temperature | `temperature` | `mdi:thermometer` |
| percent | none | `mdi:percent` |
| date-time | `timestamp` | `mdi:clock-outline` |
| kg | `weight` | `mdi:weight` |
| km | `distance` | `mdi:map-marker-distance` |
| irradiance | `irradiance` | `mdi:weather-sunny` |
| currency | `monetary` | `mdi:currency-usd` |

- **REQ-HA-012** — Reactive power value and discovery metadata MUST both use canonical `var`; device class MUST be `reactive_power`.
- **REQ-HA-013** — Only numeric sensors may have unit, state class, or last-reset metadata. Text, boolean, and timestamp sensors MUST NOT be marked as measurements.
- **REQ-HA-014** — Boot and instant/5/15/30-minute numeric points use state class `measurement`. Daily/monthly/yearly/total numeric points use `total` and a `last_reset` datetime template.
- **REQ-HA-015** — Friendly name MUST be `group - description` when both differ, otherwise the non-empty one, then point ID, then endpoint path. Duplicate group/description text MUST appear once.
- **REQ-HA-016** — Missing point metadata MAY be filled from device-point attributes: unit first, then description/group, then value type. Existing normalized metadata MUST not be overwritten.
- **REQ-HA-019** — A numeric peak-power sensor normalized to `kWp` MUST retain the `kWp` unit and its frequency-derived state metadata, MUST omit `device_class`, and MUST use `mdi:lightning-bolt`. Standard `W`, `kW`, and `MW` power sensors MUST continue to use device class `power`.
- **REQ-HA-020** — Canonical plant PV power MUST be attached to the plant device and exposed as `sensor.gosungrow_virtual_<ps_id>_pv_power`. It MUST be a numeric `kW` sensor with device class `power`, state class `measurement`, and stable endpoint-path identity under `REQ-HA-003`.

## Select entities

- **REQ-HA-017** — Select discovery MUST publish allowed options, retained state, a retained command topic, `{{ value }}` command/value templates, and subscribe at QoS 0.
- **REQ-HA-018** — Non-admin MQTT consumers MAY send commands; input validation is limited to parsing/canonicalization and MUST NOT be represented as an authorization boundary.

## Prohibited behavior

- Applying `state_class=measurement` to textual sensors.
- Publishing a unit that disagrees with the normalized state value.
- Publishing a `device_class` together with a unit that Home Assistant does not accept for that class.
- Using display names as unique identity.
