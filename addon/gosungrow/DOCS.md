# Home Assistant App: GoSungrow

GoSungrow connects to Sungrow iSolarCloud, publishes MQTT discovery data for Home Assistant, and installs managed dashboards for live flow and trends.

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
- `dashboard_language`: `auto` (default) or explicit locale (`en`, `de`, `sv`)
- `debug`: enable verbose logging

## What Happens On Startup

On a healthy setup, the app:

1. installs or updates the managed dashboards
2. initializes the iSolarCloud session and connects to MQTT
3. publishes entity discovery and state updates
4. reconciles the managed dashboard again shortly after MQTT startup, so fresh installs can remap newly created entities

No Home Assistant restart is required for the managed dashboards.

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

If the log contains `lookup gateway.isolarcloud.eu on 127.0.0.11:53: no such host` or `server misbehaving`, Docker's internal DNS resolver cannot resolve iSolarCloud. The request did not reach Sungrow, so changing iSolarCloud credentials will not help.

After MQTT has initialized, GoSungrow keeps MQTT connected and retries iSolarCloud after 15, 30, 60, 120, and then every 300 seconds. Existing Home Assistant entities retain their last published values. Normal syncing resumes automatically when DNS recovers. If DNS is already unavailable during startup, the app wrapper keeps retrying initialization with a capped delay.

Suggested checks:

1. In Home Assistant, check `Settings > System > Network` and make sure DNS points to a reliable resolver.
2. If you use Pi-hole, AdGuard, a router DNS proxy, VPN DNS, or custom firewall rules, verify that the Home Assistant host can resolve `gateway.isolarcloud.eu` and `augateway.isolarcloud.com`.
3. Check whether other apps also report lookups through `127.0.0.11:53`; that indicates a host-level DNS problem.
4. Restart Home Assistant OS or the Docker host if its embedded resolver remains unhealthy. Restarting only GoSungrow may coincide with recovery, but it cannot repair Docker DNS.

Do not configure a fixed iSolarCloud IP address. The HTTPS certificate and Sungrow's routing depend on the hostname.

## Troubleshooting Startup JSON Errors

After a sudden power loss, Home Assistant storage can occasionally contain an empty or truncated GoSungrow cache file. If startup logs show `unexpected end of JSON input`, restart the add-on once. GoSungrow removes empty cache files at startup and treats corrupt token or API response cache files as stale data, then logs in and fetches fresh data again.
