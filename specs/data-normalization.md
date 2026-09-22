# Data normalization and virtual points

Status: Normative  
Scope: Typed JSON values, reflection-to-point conversion, metadata, formulas, and filtering

## Typed values

- **REQ-DATA-001** — Strings are trimmed for emptiness checks. `""`, `--`, and case-insensitive `null` are empty scalar placeholders.
- **REQ-DATA-002** — Integers, counts, UUIDs, and plant identifiers MUST accept compatible JSON strings or numbers. Invalid/null placeholders MUST produce an invalid value without inventing zero; malformed non-placeholder data MAY report a conversion error.
- **REQ-DATA-003** — Unit-value objects accept string, integer, float, or boolean `value` with a `unit`, retain a typed representation, original printable value, validity, device identity, and conversion error state.
- **REQ-DATA-004** — Float values in `W`, `Wh`, `Wp`, or `g` MUST normalize by division by 1000 to `kW`, `kWh`, `kWp`, or `kg`. Lowercase `w` normalizes to `W`. Non-numeric strings remain unchanged and invalid.
- **REQ-DATA-005** — Reactive power spellings `var`, `Var`, `VAr`, `VAR` map 1:1 to `var`; `kvar` variants multiply by 1,000; `Mvar` variants multiply by 1,000,000; lowercase-prefix `mvar` variants divide by 1,000. Prefix case is significant. The final unit MUST be `var` and normalization MUST be idempotent.
- **REQ-DATA-006** — Unit categories include energy, power, reactive power, currency, weight, voltage, current, frequency, resistance, percent, temperature, and duration/date-time. Unknown units remain unclassified rather than guessed.
- **REQ-DATA-007** — Plant timestamps without offsets MUST preserve their wall-clock fields and receive the plant timezone offset. Known IANA timezone is preferred; a valid fixed Sungrow timezone offset is the fallback. Invalid timezone metadata MUST leave the original value unchanged.

## Point construction

- **REQ-DATA-008** — Structured API responses MUST recursively become data entries using JSON names plus point metadata: explicit/derived ID, parent/device, name, unit, update frequency, timestamp, icon, alias/virtual flags, and validity.
- **REQ-DATA-009** — `PointIgnore` fields and invalid aggregate children MUST not be published as standalone points. Arrays marked flatten MUST become one value; data-table fields retain row structure for human output.
- **REQ-DATA-010** — Update frequencies are exactly `instant`, `5mins`, `15mins`, `30mins`, `boot`, `daily`, `monthly`, `yearly`, and `total`.
- **REQ-DATA-011** — Query-device point data MUST be copied into a `virtual.<ps_key>` namespace, falling back to `virtual.<ps_id>` only when the key is empty. Each copied point keeps source ID/name/unit/timestamp/group and gets the selected target as device.
- **REQ-DATA-012** — Device-target virtual builders run only for Energy Storage System devices (`device_type=14`). Canonical plant `pv_power` aggregation is a separate per-plant virtual operation permitted for every plant with a usable `ps_id`. A missing source point MUST skip only dependent virtual points and MUST NOT panic or create a zero-valued substitute.
- **REQ-DATA-013** — An `*_active` virtual is a Boolean created from its source point: `true` exactly when the source's first value is nonzero, otherwise `false`; its unit becomes `--` and value type becomes `Bool`. Copied non-active source values retain their source semantics.

## Virtual power formulas

All formulas use values in compatible normalized units.

| Virtual point | Definition | Frequency |
|---|---|---|
| `battery_charge_power` | copy `p13126` | 5 min |
| `battery_discharge_power` | copy `p13150` | 5 min |
| `battery_power` | `signed(discharge p13150, charge p13126)` | 5 min |
| `battery_power_active` | active/state form of `battery_power` | 5 min |
| `battery_to_load_power` | copy discharge `p13150` | 5 min |
| `battery_to_load_power_active` | active/state form | 5 min |
| `battery_to_grid_power_active` | explicit `0` placeholder derived from discharge metadata | 5 min |
| `pv_power` | copy `p13003` | source/instant |
| `pv_power_active` | active/state form | source/instant |
| `pv_to_grid_power` | copy `p13121` | source/instant |
| `pv_to_grid_power_active` | active/state form | source/instant |
| `pv_to_battery_power` | copy `p13126` | source/instant |
| `pv_to_battery_power_active` | active/state form | source/instant |
| `pv_to_load_power` | `pv_power - pv_to_battery_power - pv_to_grid_power`, precision 3 | source/instant |
| `pv_to_load_power_active` | active/state form | source/instant |
| `grid_to_load_power` | copy `p13149` | source/instant |
| `grid_to_load_power_active` | active/state form | source/instant |
| `grid_power` | `signed(export p13121, import p13149)` | source/instant |
| `grid_power_active` | active/state form | source/instant |
| `grid_to_battery_power_active` | explicit `0` placeholder derived from import metadata | source/instant |
| `load_power` | copy `p13119` | source/instant |
| `load_power_active` | active/state form | source/instant |

- **REQ-DATA-014** — Define `signed(lower, upper)` as `-lower` when `lower > 0`, otherwise `upper`. The virtual power table MUST be implemented exactly when all listed inputs exist. Consequently legacy `battery_power` is negative while discharging and otherwise exposes positive charge, whereas `grid_power` is negative while exporting and otherwise exposes positive import. Each base flow that has an active variant MUST produce that variant through `REQ-DATA-013`.

## Virtual energy formulas

| Virtual point | Definition | Frequency |
|---|---|---|
| `battery_discharge_energy` | copy `p13029` | daily |
| `battery_charge_energy` | copy `p13174` | daily |
| `battery_energy` | `signed(charge p13174, discharge p13029)` | daily |
| `battery_energy_active` | active/state form of `battery_energy` | daily |
| `battery_charge_energy_percent` | charge / `p13112` × 100, precision 1 | daily |
| `pv_daily_energy` | copy `p13112` | daily |
| `pv_to_grid_energy` | copy `p13173` | daily |
| `pv_to_grid_energy_percent` | export / production × 100, precision 1 | daily |
| `pv_to_battery_energy` | copy `p13174` | daily |
| `total_daily_energy`, `daily_total_energy`, `total_load_energy` | copy `p13199` | daily |
| `pv_consumption_energy` | `p13112 - p13173 - p13174`, precision 3, using `p13116` metadata | daily |
| `pv_consumption_energy_percent` | direct solar / production × 100, precision 1 | daily |
| `pv_to_load_energy` | `p13199 - p13147`, precision 3, using `p13116` metadata | daily |
| `pv_to_load_energy_percent` | direct solar / total load × 100, precision 1 | daily |
| `pv_daily_energy_percent` | (`p13199 - p13147`) / `p13199` × 100, precision 1 | daily |
| `pv_energy` | `pv_to_load_energy + pv_to_battery_energy + pv_to_grid_energy`, precision 3 | daily |
| `pv_total_energy` | copy `p13134` | daily |
| `grid_to_load_energy` | copy `p13147` | daily |
| `grid_to_load_energy_percent` | import / total load × 100, precision 1 | daily |
| `grid_energy` | `signed(export p13173, import p13147)` | daily |

- **REQ-DATA-015** — The virtual energy table MUST be implemented only when every operand exists and is valid.
- **REQ-DATA-016** — Calculated `pv_consumption_energy` and `pv_to_load_energy` are legacy compatibility outputs and MUST be marked/reviewed as calculated when considered for the dashboard; they MUST NOT outrank a native canonical direct-solar point.
- **REQ-DATA-017** — Percentage calculation is `(value / denominator) × 100` rounded to the stated precision. An absent operand omits the percentage; a present numeric zero denominator produces numeric `0` for legacy compatibility.

## Endpoint publication filter

- **REQ-DATA-018** — Default MQTT data selection MUST be: `queryDeviceList` include `virtual.*` and exclude raw device/device-type collections; realtime include `*`; `getPsList` include `virtual.*`; `getPsDetail` include `virtual.*`.
- **REQ-DATA-019** — When an existing endpoint selection file lacks a default endpoint or required include, defaults MUST be merged in without removing user excludes or includes.
- **REQ-DATA-020** — Only valid points, allowed by endpoint include/exclude patterns, with valid values MAY reach entity publication.

## Canonical plant PV power

- **REQ-DATA-021** — A canonical plant PV result MUST be emitted at `virtual.<ps_id>.pv_power` with name `Plant PV Power`, unit `kW`, update frequency `5mins`, precision three, and the oldest timestamp among selected contributors.
- **REQ-DATA-022** — Native candidates are non-virtual plant-scoped points with identifiers `p83076`, `p83076_map`, `p83033`, `p83002`, `plant_power`, `pv_power`, and `solar_power`, in that order. The first valid power-compatible candidate MUST win, and no device value may be added to it.
- **REQ-DATA-023** — Without a native total, producer leaves are non-plant devices whose discovered point metadata contains an AC candidate or DC candidate. AC precedence per device is inverter-context `p24`, `inverter_ac_power`, `total_active_power`, then `active_power`; DC precedence is `total_dc_power`, then `dc_power`. Using the complete plant topology from `REQ-API-031`, a producer leaf is a producer device whose UUID is not referenced as `UpUUID` by another producer candidate. Select at most one point per device. Use AC only when every producer leaf has a valid AC value; otherwise use DC only when every producer leaf has a valid DC value.
- **REQ-DATA-024** — Every selected contributor MUST be finite, belong to the same plant and successful collection snapshot, and have a unit convertible from `W`, `kW`, or `MW` to `kW`. An absent `ps_id`, empty producer set, incomplete topology, missing contributor, incompatible unit, mixed measurement basis, or invalid value MUST omit the aggregate rather than create zero or a partial total. Native plant totals do not require topology.

## Prohibited behavior

- Creating dependent virtual data when a source point is absent.
- Treating the legacy calculated direct-solar aliases as verified native `p13116`.
- Converting unknown unit spellings by case-insensitive prefix guessing.
- Mixing AC and DC contributors in plant `pv_power`.
- Summing a native or parent aggregate together with its producer leaves.
- Publishing a partial producer sum as complete plant production.
