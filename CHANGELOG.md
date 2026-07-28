# Changelog

## 3.2.1

- Highlight safer automatic source recommendations in the Data Sources dialog without changing the active source automatically.
- Place the recommended replacement first and explain the two-step select-and-confirm action in English, German, and Swedish.

## 3.2.0.8

- Match daily energy sources through canonical Sungrow point semantics and Home Assistant entity-registry unique IDs, independent of renamed or translated entity names.
- Preserve installed automatic and manual mappings while offering safer canonical sources for explicit adoption; unsupported calculated direct-solar values remain visible only as warned legacy mappings.
- Show direct solar consumption as unavailable when no native `p13116` or `p83097` source exists, and keep ambiguous calculated values out of new defaults and candidate searches.
- Keep the Data Sources selector stable across live Home Assistant updates, including expanded candidates, search, pending selection, scroll position, and keyboard focus.
- Interpret Data Last Update Time using the plant timezone supplied by Sungrow and publish the resulting instant as RFC 3339.
- Bound iSolarCloud HTTP requests to 60 seconds and add endpoint-aware synchronization progress and duration logs while preserving MQTT state through recoverable timeouts.

## 3.2.0.7

- Add constrained semantic matching for solar power, daily production, grid import/export, and daily home consumption.
- Reject incompatible device roles, directions, phase readings, and lifetime totals for these dashboard metrics.
- Preserve every existing automatic mapping and manual override; safer matches are offered for explicit review instead of silently remapping installed dashboards.
- Use confident semantic matches automatically for newly generated dashboards and keep target bindings isolated when metrics previously shared an entity.

## 3.2.0.6

- Normalize supported Sungrow reactive-power units to Home Assistant's canonical `var` unit before MQTT discovery and state publication.
- Keep reactive-power values, point metadata, entity configuration, and MQTT discovery units aligned to prevent recurring unit-repair prompts.
- Preserve entity identity and all non-reactive sensors unchanged. Existing recorder history is not deleted or rewritten.
- Leave reactive-energy normalization and MQTT discovery lifecycle changes for separately audited follow-ups.

## 3.2.0.5

- Honor Home Assistant language, system, and explicit DMY/MDY/YMD date preferences in Energy Summary chart labels.
- Relabel cached chart buckets when frontend date preferences change without refetching recorder statistics.

## 3.2.0.4

- Add an administrator-only Data Sources view to review automatic dashboard matches and choose persistent overrides.
- Rank compatible candidates by metric, unit, and plant affinity without changing the existing automatic defaults.
- Preserve manual dashboard mappings through restarts, upgrades, and forced dashboard reconciliation.
- Warn when direct solar consumption is physically inconsistent with solar production.
- Add responsive, localized source-selection UI and desktop/mobile preview scenarios.
- Isolate overrides per dashboard target, reject incompatible persisted values, and protect unrelated user edits with a normalized structure hash.
- Use live Home Assistant values for source health, cap candidate payloads, verify saves transactionally, and refresh the Lovelace configuration without reloading Home Assistant.
- Give the Data Sources workspace a full-width responsive layout and shorten shared sensor-name prefixes while retaining complete names for tooltips and accessibility.
- Make manual Live Flow node sources authoritative so Solar, Home, Grid, and Battery selections immediately change their displayed values while automatic sources retain the existing directional calculations.
- Shorten shared sensor-name prefixes in Data Sources rows as well as the configuration dialog while retaining full identity details in tooltips and accessibility labels.

## 3.1.11

- Keep an established MQTT session alive during Docker DNS outages.
- Retry iSolarCloud DNS failures with a capped 15-to-300-second backoff and resume normal syncing automatically after recovery.
- Avoid unnecessary token refreshes, gateway rotation, and process restarts for `127.0.0.11:53` resolver failures.
- Retry initialization when DNS is unavailable at app startup instead of failing on a one-shot login.
- Improve DNS outage and recovery logging and troubleshooting guidance.
