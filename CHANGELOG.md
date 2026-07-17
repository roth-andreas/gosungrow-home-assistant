# Changelog

## 3.2.0

- Add an administrator-only Data Sources view to review automatic dashboard matches and choose persistent overrides.
- Rank compatible candidates by metric, unit, and plant affinity without changing the existing automatic defaults.
- Preserve manual dashboard mappings through restarts, upgrades, and forced dashboard reconciliation.
- Warn when direct solar consumption is physically inconsistent with solar production.
- Add responsive, localized source-selection UI and desktop/mobile preview scenarios.
- Isolate overrides per dashboard target, reject incompatible persisted values, and protect unrelated user edits with a normalized structure hash.
- Use live Home Assistant values for source health, cap candidate payloads, and save changes transactionally without reloading Home Assistant.

## 3.1.11

- Keep an established MQTT session alive during Docker DNS outages.
- Retry iSolarCloud DNS failures with a capped 15-to-300-second backoff and resume normal syncing automatically after recovery.
- Avoid unnecessary token refreshes, gateway rotation, and process restarts for `127.0.0.11:53` resolver failures.
- Retry initialization when DNS is unavailable at app startup instead of failing on a one-shot login.
- Improve DNS outage and recovery logging and troubleshooting guidance.
