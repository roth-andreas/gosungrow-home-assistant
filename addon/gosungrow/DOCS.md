# Home Assistant App: GoSungrow

GoSungrow connects to Sungrow iSolarCloud, publishes MQTT discovery data for Home Assistant, and installs managed dashboards for live flow and trends.

Exact program behavior is defined by the repository's `specs/` directory; this document is installation and operating guidance.

## Before You Install

This app requires MQTT.

Before installing `GoSungrow`, make sure Home Assistant already has:

1. a running MQTT broker
2. the `MQTT` integration under `Settings > Devices & services`

For most users, that means installing and starting the `Mosquitto broker` app first. If Home Assistant already uses an external MQTT broker, enter that broker in the optional MQTT settings instead.

## Install

1. Open the Home Assistant App Store.
2. Add this repository as a custom repository:
   - `https://github.com/roth-andreas/gosungrow-home-assistant`
3. Refresh the App Store.
4. Install `GoSungrow`.
5. Enter your `gosungrow_user` and `gosungrow_password`.
6. Start the app.

## Configuration

Required:

- `gosungrow_user`
- `gosungrow_password`

Optional:

- `mqtt_host`: custom MQTT broker host; leave empty to use the Home Assistant MQTT service
- `mqtt_port`: custom MQTT broker port; defaults to `1883`
- `mqtt_username`: custom MQTT username; leave empty to use the Home Assistant MQTT service credentials
- `mqtt_password`: custom MQTT password; leave empty to use the Home Assistant MQTT service credentials
- `install_dashboard`: create or update the managed dashboard automatically
- `dashboard_language`: `auto` (default) or explicit locale (`en`, `de`, `sv`, `es`)
- `debug`: enable verbose logging

## What Happens On Startup

On a healthy setup, the app:

1. installs or updates the managed dashboards
2. initializes the iSolarCloud session and connects to MQTT
3. publishes entity discovery and state updates
4. reconciles the managed dashboard again shortly after MQTT startup, so fresh installs can remap newly created entities

No Home Assistant restart is required for the managed dashboards.

The dashboard cards are served locally from a content-addressed `/local/gosungrow/` module. GoSungrow verifies the exact file directly through Home Assistant before referencing it; API and WebSocket traffic continue through Supervisor. Static verification sends no Supervisor credential and keeps the previous verified version for rollback. A browser that was already open during first installation may need one ordinary page reload; a hard refresh is not required.

If the enhanced card module cannot be activated on a fresh installation, GoSungrow installs a native Home Assistant dashboard automatically. Live MQTT-backed metrics remain usable while flow visualization and source editing are temporarily reduced. A later reconciliation promotes the dashboard to enhanced mode without a Home Assistant restart.

## Correcting Dashboard Data Sources

GoSungrow continues to choose dashboard sensors automatically. If a Sungrow model exposes different point names or meanings, a Home Assistant administrator can open the managed dashboard's **Data Sources** view and override an individual dashboard metric.

- Automatic matching remains the default until an administrator explicitly changes a metric.
- Candidate lists are filtered by plant, measurement type, unit, and usable state, with the strongest matches shown first.
- Values, availability, confidence, and physical-consistency warnings use current Home Assistant state; candidate snapshots are not stored in the dashboard.
- A **Needs review** badge identifies suspicious relationships, such as direct solar consumption exceeding solar production.
- Selecting a candidate is a preview step. The dashboard changes only after **Use this source** succeeds, and failed or stale saves leave the current mapping untouched.
- **Reset to automatic** removes a manual choice at any time.
- Overrides apply only to the managed GoSungrow dashboard. MQTT entities, automations, Home Assistant Energy configuration, and other dashboards are not changed.
- Overrides are stored in the managed dashboard and preserved when GoSungrow updates it. A missing manually selected entity remains visible as unavailable instead of silently changing back.
- Installations with multiple plants or targets are isolated: changing one target cannot rewrite another target that happens to share a plant-level sensor.

For multi-inverter and microinverter plants, GoSungrow publishes a stable `sensor.gosungrow_virtual_<plant-id>_pv_power` entity when it can use a native plant total or build a complete sum from compatible producer readings. AC and DC readings are never mixed, and incomplete device topology does not produce a partial total. New dashboards select this entity automatically. Existing manual selections remain unchanged; an older pinned automatic device source is marked for review and offered the plant aggregate as the preferred replacement.

Non-administrator users can inspect the selected sources but cannot modify them.

## Notes

- Runtime state is stored in `/data/.GoSungrow`.
- The managed dashboard state is stored in `/data/.GoSungrow/dashboard_state.json`.
- If no entities appear, verify MQTT first.
- The app uses the standard iSolarCloud host, app key, and managed dashboard path internally.
- MQTT uses the custom broker settings when `mqtt_host` is set, otherwise it falls back to the Home Assistant MQTT service.
- Managed dashboard text follows Home Assistant language when available (fallback: English).
- If you are updating from an older version with more options, open the app configuration once and save it to clear legacy fields.

## Troubleshooting DNS Errors

When GoSungrow classifies the startup failure as Docker DNS, Docker's internal resolver cannot resolve the first iSolarCloud gateway. The request did not reach Sungrow, so changing iSolarCloud credentials will not help.

A failed regional-gateway search may also contain a later `127.0.0.11:53` message. In that case, GoSungrow keeps the earlier login or gateway failure, reports the first failure and terminal stop reason, and uses normal remote recovery instead of claiming a general Docker-DNS outage. Check the complete ordered summary to distinguish credentials or regional-server selection from a direct DNS failure.

After MQTT has initialized, GoSungrow keeps MQTT connected and retries iSolarCloud after 15, 30, 60, 120, and then every 300 seconds. Existing Home Assistant entities retain their last published values. Normal syncing resumes automatically when DNS recovers. If DNS is already unavailable during startup, the app wrapper keeps retrying initialization with a capped delay.

Suggested checks:

1. In Home Assistant, check `Settings > System > Network` and make sure DNS points to a reliable resolver.
2. If you use Pi-hole, AdGuard, a router DNS proxy, VPN DNS, or custom firewall rules, verify that the Home Assistant host can resolve `gateway.isolarcloud.eu` and `augateway.isolarcloud.com`.
3. Check whether other apps also report lookups through `127.0.0.11:53`; that indicates a host-level DNS problem.
4. Restart Home Assistant OS or the Docker host if its embedded resolver remains unhealthy. Restarting only GoSungrow may coincide with recovery, but it cannot repair Docker DNS.

Do not configure a fixed iSolarCloud IP address. The HTTPS certificate and Sungrow's routing depend on the hostname.

## Troubleshooting Dashboard Cards

Dashboard lifecycle logs report the asset phase, short canonical URL, verification route, Supervisor metadata outcome and discovered port/TLS setting, response status and MIME type, resource action, dashboard mode, rollback, and cleanup result. `supervisor-core-info` is expected for the Home Assistant app; `websocket-origin` is expected for standalone CLI connections. If the dashboard remains in native fallback mode, check the metadata outcome and confirm that Home Assistant can serve `/local/gosungrow/` JavaScript with status 200 and a JavaScript content type. Do not add a CDN or data-URL resource manually; GoSungrow will retry discovery and activation during reconciliation.

## Troubleshooting Startup JSON Errors

After a sudden power loss, Home Assistant storage can occasionally contain an empty or truncated GoSungrow cache file. If startup logs show `unexpected end of JSON input`, restart the add-on once. GoSungrow removes empty cache files at startup and treats corrupt token or API response cache files as stale data, then logs in and fetches fresh data again.
