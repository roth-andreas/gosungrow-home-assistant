# Changelog

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
