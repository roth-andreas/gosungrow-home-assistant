# Domain model and energy semantics

Status: Normative  
Scope: Identifiers, devices, measurements, periods, and energy meanings

## Entities

- **REQ-DOM-001** — A plant is identified by `ps_id`; a device/target is identified by `ps_key` and belongs to one plant.
- **REQ-DOM-002** — `ps_id` and `ps_key` MUST be treated as opaque strings after trimming. Numeric and composite forms MUST round-trip without truncation or numeric coercion.
- **REQ-DOM-003** — Empty strings, whitespace, `NULL`, `null`, `--`, and composite placeholder values lacking a usable identifier MUST be treated as absent where the typed field requires an identifier or number.
- **REQ-DOM-004** — A point consists of stable ID, display name, value, unit, value type, device/parent identity, timestamp, update frequency, icon, and validity flags. Unknown response fields MAY be retained but MUST NOT change known semantics.
- **REQ-DOM-005** — Device roles relevant to selection are: `1=inverter`, `7=meter`, `11=plant`, `14=energy storage system`, `22=communication`; other types are `unknown` for semantic matching.

## Measurement semantics

- **REQ-DOM-006** — Power is instantaneous rate and accepts `W`, `kW`, or `MW`; energy is accumulated quantity and accepts `Wh`, `kWh`, or `MWh`; state of charge accepts `%`.
- **REQ-DOM-007** — Daily, monthly, yearly, total, boot, and instantaneous values MUST retain distinct update-period metadata. A daily value MUST NOT be substituted by a lifetime value merely because names overlap.
- **REQ-DOM-008** — Missing, unknown, unavailable, non-numeric, NaN, or infinite states MUST NOT be treated as zero during dashboard source selection.
- **REQ-DOM-009** — A genuine numeric zero is available data and MUST remain distinguishable from absence.
- **REQ-DOM-010** — Boolean points MUST be modeled as binary state rather than numeric measurement.

## Canonical dashboard meanings

| Metric | Meaning | Kind | Direction/period |
|---|---|---|---|
| `pv_power` | total current solar production | power | source |
| `load_power` | current home/load consumption | power | sink |
| `grid_power` | net grid exchange | power | import positive, export negative when synthesized |
| `battery_power` | net battery flow | power | legacy virtual convention: discharge negative, otherwise charge positive |
| `pv_to_load_power` | current solar supplied directly to load | power | PV → load |
| `pv_to_battery_power` | current solar charging battery | power | PV → battery |
| `pv_to_grid_power` | current solar export | power | PV → grid |
| `grid_to_load_power` | current grid import serving load | power | grid → load |
| `battery_to_load_power` | current battery discharge serving load | power | battery → load |
| `p13141` | battery state of charge | percent | current |
| `p13112` | solar production | energy | current day |
| `p13116` | direct solar consumption | energy | current day |
| `p13174` | solar energy sent to battery | energy | current day |
| `p13173` | energy exported to grid | energy | current day |
| `p13147` | energy imported from grid | energy | current day |
| `p13199` | total home consumption | energy | current day |
| `p13029` | battery discharge | energy | current day |

- **REQ-DOM-011** — Grid import and export MUST NOT be interchanged; battery charge and discharge MUST NOT be interchanged.
- **REQ-DOM-012** — Direct solar consumption MUST NOT be inferred from an arbitrary consumption or production sensor.
- **REQ-DOM-013** — A single inverter value MUST NOT silently represent plant production when multiple inverter-like devices make it incomplete.
- **REQ-DOM-014** — Physical validation MUST flag direct solar consumption materially above solar production, using relative tolerance `5%` and absolute tolerance `0.1` in the entities' compatible energy unit.
- **REQ-DOM-015** — The live flow card's displayed battery node uses the presentation convention discharge positive and charge negative, derived from directional flows. The published legacy `battery_power` virtual point retains the opposite sign convention specified in [data-normalization.md](data-normalization.md); consumers MUST NOT silently assume the two conventions are identical.

## Device-type catalog

The transport MAY expose other recognized types: 3 grid connection, 4 combiner box, 5 meteo station, 6 transformer, 8 UPS, 9 data logger, 10 string, 12 circuit protection, 13 splitting device, 15 sampling device, 16 EMU, 17 unit, 18 temperature/humidity, 19 intelligent distribution cabinet, 20 display, 21 AC distribution cabinet, 23 system BMS, 24 rack BMS, 25 DC-DC, 26 EMS, 28 wind converter, 29 SVG, 30 PT cabinet, 31 bus protection, 32 cleaning robot, 33 DC cabinet, 34 public measurement/control, 35 anti-islanding, 36 frequency/voltage emergency control, 37 PCS, 38 cell BMS, 39 power quality, 40 shuttle, 41 optimizer, 42 tracking-axis communication, 43 battery, 44 battery cluster management, 45 local controller, 46 networking, 47 storage unit, 48 DC container, 50 I/O module, and 99 other.

## Prohibited behavior

- Parsing opaque IDs as integers for identity or dropping composite suffixes.
- Equating similarly named power and energy fields.
- Publishing invalid numeric placeholders as valid measurements.
