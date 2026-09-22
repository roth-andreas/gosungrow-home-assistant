# ADR-006: Transactional dashboard assets

Status: Accepted
Date: 2026-09-22

## Context

The managed dashboard depends on three custom elements delivered by one JavaScript bundle. Embedded data URLs, mutable local filenames, and resource mutation before delivery verification can leave Home Assistant dashboards referencing an unavailable or stale module. Browser caches and partial installer failures make that state difficult to diagnose and recover from safely.

## Decision

GoSungrow delivers the co-versioned card bundle from a content-addressed `/local/gosungrow/` URL. Installation stages and hashes immutable bytes, verifies the exact response through Home Assistant, activates and re-reads the Lovelace resource, saves and verifies the dashboard, and persists state only after those steps succeed. A later activation failure rolls the resource and dashboard back to their prior working values.

A fresh installation that cannot activate the module degrades to a resource-independent dashboard composed only of native Home Assistant cards. It preserves targets, view paths, and resolved source decisions so a later reconciliation can promote it automatically. The last verified bundle is retained through upgrades, and lifecycle diagnostics describe bounded metadata rather than asset bodies.

## Consequences

- Browser cache invalidation follows naturally from the content hash; a browser opened before resource registration may need one ordinary reload but never a forced hard refresh.
- The three custom cards remain one self-contained, synchronously registering bundle and are released together.
- Resource and dashboard mutation require explicit rollback and boundary tests.
- Native fallback remains usable for monitoring while enhanced presentation and source editing are unavailable.

## Rejected alternatives

- Embedded data URLs and mutable local filenames obscure delivery failures and cache identity.
- CDN delivery adds a runtime dependency and weakens local availability.
- HACS packaging, a separate web service, or a native Home Assistant integration would split ownership and deployment without solving transactional activation.
- Removing the custom cards would discard the enhanced flow, summary, and source-selection experience rather than isolating its failure mode.
