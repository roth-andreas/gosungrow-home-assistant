# Dashboard custom cards

Status: Normative  
Scope: Live flow, aggregate statistics, formatting, localization, interaction, and registration

## Common contract

- **REQ-CARD-001** — The bundle MUST register `gosungrow-energy-flow-card-v2`, `gosungrow-energy-summary-card-v1`, and `gosungrow-source-mapping-card-v1`, and advertise all three in `window.customCards`.
- **REQ-CARD-002** — Cards MUST reject missing required entity configuration, render in shadow DOM, escape all interpolated user/entity text, use Home Assistant theme variables, and rerender on `hass` updates.

## Live energy flow

- **REQ-CARD-003** — The flow card requires the ten live metrics in [domain-model.md](domain-model.md), renders PV, grid, home, and battery nodes plus five directional edges, and switches to compact layout below 700 px effective width.
- **REQ-CARD-004** — Automatic node values MUST be calculated from flows: PV=`PV→load + PV→grid + PV→battery`; grid=`grid→load - PV→grid`; home=`PV→load + grid→load + battery→load`; battery=`battery→load - PV→battery`.
- **REQ-CARD-005** — If a node entity is manually selected and differs from its `automatic_entities` entry, its direct entity state MUST replace the computed node value. Otherwise computed value is used when any contributing flow is available, with direct value as fallback.
- **REQ-CARD-006** — A flow is active only above absolute `0.01`. Active line width scales as `3.4 + min(magnitude,6)×1.2`; inactive edges and labels are hidden. The PV→battery edge label is hidden when it duplicates the signed battery node within `0.01`.
- **REQ-CARD-007** — Entity values MUST format with current unit, localized number formatting, adaptive precision, and explicit unavailable text. Clicking a configured node MUST emit `hass-more-info` for that entity.

## Energy summary

- **REQ-CARD-008** — Summary metrics and order are production, consumption, to grid, from grid, to battery, from battery with colors `#f59e0b`, `#38bdf8`, `#8b5cf6`, `#cbd5e1`, `#ec4899`, `#14b8a6`.
- **REQ-CARD-009** — Periods are day, month, year; default bucket counts are 14, 12, 5 and configured counts MUST be integers 1–60. The browser MUST request Home Assistant `recorder/statistics_during_period` daily rows with types `max`, `state`, and `sum` for all views.
- **REQ-CARD-010** — Completed daily sensors use recorder `max` when finite, then `state`, then `sum`. The current day's recorder row MUST be ignored and replaced with the live daily entity value.
- **REQ-CARD-011** — Day charts keep daily buckets; month and year charts sum completed daily buckets plus today's live value into calendar month/year. Missing baseline on an early installation MUST retain the first observed completed day rather than compute a delta.
- **REQ-CARD-012** — Invalid/missing live and recorder data MUST remain absent. Empty results show statistics-unavailable/no-statistics state and MUST NOT invent zero buckets.
- **REQ-CARD-013** — Statistic starts accept ISO timestamps and millisecond epoch values. Buckets are generated in local calendar time, ordered chronologically, trimmed to the configured count, and relabeled without refetch when locale/date preference changes.
- **REQ-CARD-014** — Day labels follow Home Assistant `date_format`: `DMY` day/month, `MDY` month/day, `YMD` uses month/day for compactness, `system` uses system locale, `language`/missing follows language locale. Month uses localized short month and year uses localized two-digit year.
- **REQ-CARD-015** — Bars MUST be keyboard focusable images with localized ARIA labels and tooltips. Tooltip rows include every available configured metric in contract order with its color, label, and formatted value.

## Localization

- **REQ-CARD-016** — Explicit card labels override built-in labels. Runtime locale lookup order is full Home Assistant/browser locale, base language, then English.
- **REQ-CARD-017** — The frontend MUST support the same `en`, `de`, `sv`, `es` semantics as server-generated labels and preserve English fallback.
- **REQ-CARD-018** — The delivered dashboard module MUST be self-contained with no third-party runtime imports. Evaluation MUST synchronously register the energy-flow, energy-summary, and source-mapping custom elements and their `window.customCards` metadata. Re-evaluation or upgrade MUST tolerate earlier compatible element and metadata registrations without throwing or duplicating entries.

## Prohibited behavior

- Summing a current-day recorder value together with the live daily value.
- Treating unavailable series as zero.
- Replacing manual node readings with computed values.
