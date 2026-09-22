# ADR-006: Transactional dashboard assets

Status: Accepted
Date: 2026-09-22

## Context

The managed dashboard depends on three custom elements delivered by one JavaScript bundle. Embedded data URLs, mutable local filenames, and resource mutation before delivery verification can leave Home Assistant dashboards referencing an unavailable or stale module. Browser caches and partial installer failures make that state difficult to diagnose and recover from safely.

## Decision

GoSungrow delivers the co-versioned card bundle from a content-addressed `/local/gosungrow/` URL. Installation stages and hashes immutable bytes, verifies the exact response through Home Assistant, activates and re-reads the Lovelace resource, saves and verifies the dashboard, and persists state only after those steps succeed. A later activation failure rolls the resource and dashboard back to their prior working values.

Home Assistant API/websocket transport and static-resource transport are separate trust boundaries. The Supervisor proxy is used only for supported API and websocket routes; it does not expose `/local`. Under Supervisor, authenticated `/core/info` metadata is authoritative for the direct Home Assistant scheme and port, while `homeassistant` remains the stable internal hostname. GoSungrow requests that metadata afresh for each reconciliation using only the minimal Supervisor API permission and default role; it neither guesses or probes ports nor persists discovered topology. Standalone operation derives the static origin from its websocket endpoint. The Supervisor credential is sent only to `/core/info`; static requests carry no credential, do not follow redirects, and retain standard TLS certificate verification.

A fresh installation that cannot activate the module degrades to a resource-independent dashboard composed only of native Home Assistant cards. It preserves targets, view paths, and resolved source decisions so a later reconciliation can promote it automatically. The last verified bundle is retained through upgrades, and lifecycle diagnostics describe bounded metadata rather than asset bodies.

Home Assistant registers `/local` during frontend startup only when the config-root `www` directory already exists. A first GoSungrow installation can therefore stage a correct file after Core startup while `/local` still returns 404. GoSungrow classifies only that verified-stage/HTTP-404 boundary as requiring operator action, preserves or installs native mode, and asks for one user-controlled Core restart. It never restarts Core itself. The next reconciliation promotes automatically after the route becomes available, and the unavailable resource is never registered or exposed to browser negative caching.

## Consequences

- Browser cache invalidation follows naturally from the content hash; a browser opened before resource registration may need one ordinary reload but never a forced hard refresh.
- The three custom cards remain one self-contained, synchronously registering bundle and are released together.
- Resource and dashboard mutation require explicit rollback and boundary tests.
- Deployment-topology tests must distinguish the Supervisor websocket proxy from Home Assistant's direct static-file origin.
- Native fallback remains usable for monitoring while enhanced presentation and source editing are unavailable.
- First-time installations may require one explicit Core restart when `/local` was not registered at startup; ordinary upgrades and delivery failures do not.

## Rejected alternatives

- Embedded data URLs and mutable local filenames obscure delivery failures and cache identity.
- CDN delivery adds a runtime dependency and weakens local availability.
- HACS packaging, a separate web service, or a native Home Assistant integration would split ownership and deployment without solving transactional activation.
- Reusing the Supervisor websocket origin for `/local` is invalid because the proxy exposes supported API and websocket routes rather than Home Assistant frontend static files.
- Hardcoding port 80 or 8123 is incompatible with other managed and custom Home Assistant Core ports.
- Port probing, silent fallback, or persisted topology can select the wrong service, conceal permission or metadata failures, and become stale across Core restarts.
- Disabling TLS certificate verification would weaken the direct static-resource trust boundary.
- Blindly retrying an unregistered `/local` route cannot repair the Core startup condition.
- Automatically restarting Core is an unacceptable availability side effect; restarting only the GoSungrow app does not register the route.
- Registering the resource before HTTP verification exposes browsers to a missing content-addressed URL and possible negative caching.
- Removing the custom cards would discard the enhanced flow, summary, and source-selection experience rather than isolating its failure mode.
