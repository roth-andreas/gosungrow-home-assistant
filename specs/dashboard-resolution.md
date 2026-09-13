# Dashboard metric resolution

Status: Normative  
Scope: Placeholder discovery, candidate compatibility, scoring, canonical semantics, ambiguity, and deterministic selection

## Candidate gate

- **REQ-RES-001** — Only `sensor.*` entities containing `gosungrow` are automatic candidates. Placeholders have form `sensor.gosungrow_virtual_<ps_key|ps_id>_<metric>`.
- **REQ-RES-002** — On multiple targets, a candidate MUST contain the target key or plant ID as a bounded identifier. On a single target, an otherwise compatible GoSungrow entity receives weak fallback affinity.
- **REQ-RES-003** — Candidate state MUST be finite numeric and not empty, unknown, unavailable, none, or null. Present units MUST match metric kind; absent units are accepted with a penalty.
- **REQ-RES-004** — Exact usable virtual entity by target key, then plant ID, MUST win with score `9999`.

## General scoring

For non-exact candidates the score MUST be additive:

| Factor | Score |
|---|---:|
| target key affinity | +120 |
| plant ID affinity | +90 |
| single-target fallback affinity | +10 |
| exact metric suffix | +240 |
| ordered alias suffix | `214 - 3 × alias index` |
| each required token group | +30 |
| compatible explicit unit | +28 |
| missing unit | -18 |
| each forbidden token | -28 |
| virtual namespace | +16 |
| information namespace | +6 |
| `sensor.gosungrow_` prefix | +8 |
| suffix `_active` | -4 |
| token-only rather than suffix match | -24 |

- **REQ-RES-005** — Candidate must match a suffix alias or every token group. Forbidden tokens penalize legacy matching; canonical semantic contracts reject their forbidden meanings outright.
- **REQ-RES-006** — Ties choose the shorter entity ID; equal-length ties preserve input state order. Diagnostic candidate lists contain the best five unique entities in score-descending, length-ascending, stable-input order.
- **REQ-RES-007** — `p1` is a `p13112` alias and `p24` is a `pv_power` alias only with inverter context. Inverter context is an explicit inverter name or a virtual key encoding device type 1.
- **REQ-RES-008** — Source preference bonuses use substring matching unless stated otherwise: PV `_pv_power`/`_solar_power`/`_total_dc_power` +48, inverter-context suffix `_p24` +44, `_p83076`/`_p83033`/`_p83002` +18; grid `_p8018`/`_p83032`/`_meter_active_power`/`_meter_ac_power` +42, `_p83549`/`_grid_active_power` +18; daily yield inverter-context suffix `_p1`, `_p83009`, `_yield_today`, or `_today_yield` +48, `_p83022`/`_daily_yield_of_plant` +18, `_p83018`/`_theoretical` -60.

## Metric profiles

The ordered aliases below are normative; tokens are alternatives within `/` groups and all groups are required.

| Metric | Ordered aliases (including the metric itself) | Required token groups | Forbidden |
|---|---|---|---|
| `pv_power` | `pv_power`, `pv_power_active`, `solar_power`, `p13003`, `p24`, `p83076`, `p83076_map`, `p83033`, `p83002`, `total_dc_power`, `dc_power`, `plant_power`, `inverter_ac_power`, `total_active_power`, `active_power` | `pv`/`solar` + `power` | phase A/B/C canonically |
| `load_power` | `load_power`, `load_power_active`, `p13119`, `p83106`, `p83106_map`, `house_power`, `consumption_power`, `use_power`, `total_load_power`, `total_active_power`, `active_power` | `load`/`home`/`house`/`consumption`/`use` + `power` | — |
| `grid_power` | `grid_power`, `grid_power_active`, `net_grid_power`, `p13149`, `p13121`, `p8018`, `p83032`, `p83549`, `meter_active_power`, `meter_ac_power`, `grid_active_power`, `active_power`, `total_active_power`, `import_power`, `export_power` | `grid`/`meter`/`import`/`export`/`feed`/`purchased` + `power` | `battery` |
| `battery_power` | `battery_power`, `battery_power_active`, `es_power`, `p13126`, `p13150`, `p83081`, `p83081_map`, `p83128`, `p83128_map`, `battery_charge_power`, `battery_discharge_power`, `charge_power`, `discharge_power`, `total_active_power_of_optical_storage` | `battery`/`soc`/`es`/`storage`/`optical` + `power` | — |
| `pv_to_load_power` | `pv_to_load_power`, `pv_to_load_power_active`, `load_from_pv_power`, `pv_consumption_power` | `pv`/`solar` + `load`/`home`/`house`/`consumption`/`use` + `power` | conflicting direction canonically |
| `pv_to_battery_power` | `pv_to_battery_power`, `pv_to_battery_power_active`, `p13126`, `battery_charge_power` | `pv`/`solar` + `battery`/`es` + `power` | conflicting direction canonically |
| `pv_to_grid_power` | `pv_to_grid_power`, `pv_to_grid_power_active`, `p13121`, `export_power`, `feed_in_power` | `pv`/`solar` + `grid`/`export`/`feed` + `power` | conflicting direction canonically |
| `battery_to_load_power` | `battery_to_load_power`, `battery_to_load_power_active`, `p13150`, `battery_discharge_power` | `battery`/`es` + `load`/`home`/`house`/`consumption`/`use` + `power` | conflicting direction canonically |
| `grid_to_load_power` | `grid_to_load_power`, `grid_to_load_power_active`, `p13149`, `import_power`, `purchased_power` | `grid`/`import`/`purchased` + `load`/`home`/`house`/`consumption`/`use` + `power` | conflicting direction canonically |
| `p13112` | `p13112`, `p83009`, `yield_today`, `today_yield`, `daily_yield_by_inverter`, `p1`, `p83022`, `p83022y`, `pv_daily_energy`, `daily_pv_yield`, `daily_pv_energy`, `pv_yield`, `daily_yield_of_plant` | `pv`/`solar` + `energy`/`yield`/`production` | import/export/feed/`p13122`/`p13173`/theoretical canonically |
| `p13116` | `p13116`, `p83097`, `p83097_map`, `pv_to_load_energy`, `pv_consumption_energy`, `daily_load_energy_consumption_from_pv` | `pv`/`solar` + `load`/`home`/`house`/`consumption`/`use` + `energy` | calculated aliases canonically |
| `p13174` | `p13174`, `p83120`, `p83120_map`, `p83088`, `p83088_map`, `pv_to_battery_energy`, `battery_charge_energy`, `battery_charging_energy_from_pv`, `daily_battery_charging_energy_from_pv`, `energy_battery_charge`, `es_energy` | `battery`/`es` + `energy` + `charge`/`charging` | discharge canonically |
| `p13173` | `p13173`, `p83119`, `p83119_map`, `pv_to_grid_energy`, `feed_in_energy`, `export_energy`, `energy_feed_in` | `grid`/`export`/`feed` + `energy` | import/purchased/`p83102` canonically |
| `p13147` | `p13147`, `p83102`, `p83102_map`, `grid_to_load_energy`, `grid_import_energy`, `purchased_energy`, `energy_purchased` | `grid`/`import`/`purchased` + `energy` | export/feed/`p83119` canonically |
| `p13199` | `p13199`, `total_daily_energy`, `total_load_energy`, `daily_total_energy`, `load_energy`, `consumption_energy`, `house_energy`, `home_energy`, `daily_consumption_energy` | `load`/`home`/`house`/`consumption`/`use`/`total` + `energy` | `pv`/`grid`/`battery`/`feed`/`export`/`import` |
| `p13029` | `p13029`, `battery_discharge_energy`, `battery_to_load_energy`, `discharge_energy`, `energy_battery_discharge`, `daily_battery_discharge_energy` | `battery`/`es` + `energy` + `discharge`/`discharging` | charge canonically |
| `p13141` | `p13141`, `p83129`, `p83252`, `battery_soc`, `battery_level`, `battery_charge_percent`, `soc` | `battery`/`soc` + `percent`/`level`/`charge`/`soc` | — |

- **REQ-RES-009** — Directional power profiles are: PV→load, PV→battery (`p13126`), PV→grid (`p13121`), battery→load (`p13150`), grid→load (`p13149`); their tokens MUST mention both endpoints and power.

## Canonical daily matching

- **REQ-RES-010** — Registry unique ID MUST be used for canonical matching when any registry metadata is available; entity ID is used only when registry data is globally unavailable, and the result is unverified/non-confident.
- **REQ-RES-011** — Canonical terminal IDs and base scores are: `p13112` 1000 plant, `p83022` 980 plant, `p83009` 920 inverter, `p1` 900 inverter, verified yield aliases 880/870; `p13116` 1000 plant, `p83097` 980 plant; `p13173` 1000, `p83119` 980, export aliases 950/940/930; `p13147` 1000, `p83102` 980, import aliases 950/940/930; `p13199` 1000 followed by verified daily-total/load aliases 960 down to 910.
- **REQ-RES-012** — Canonical plant scope gains +80 and inverter role gains +30. Candidates sort score descending then entity ID ascending.
- **REQ-RES-013** — Canonical match is confident only when registry metadata exists and the winner leads the next non-equivalent result by at least 20, unless there is only one/equivalent duplicate. Inverter-scoped `p13112` additionally requires exactly one inverter in the target plant.
- **REQ-RES-014** — A legacy semantic winner is confident with one candidate, a lead of at least 20, or an equivalent `_2` duplicate having equal state/unit.
- **REQ-RES-015** — Daily semantics require an identity containing one of `_today`, `today_`, `_daily`, `daily_`, `_p13112`, `_p13116`, `_p13147`, `_p13173`, `_p13199`, or a valid RFC3339 `last_reset` between 6 hours in the future and 48 hours old.

## Prohibited behavior

- Resolving a candidate only because it has a plausible current value.
- Silently choosing one of several equally plausible non-equivalent candidates.
- Canonically accepting a renamed entity by entity ID when registry metadata is available but mismatched.
