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

1. refreshes the iSolarCloud session
2. installs or updates the managed dashboards
3. connects to MQTT
4. publishes entity discovery and state updates
5. reconciles the managed dashboard again shortly after MQTT startup, so fresh installs can remap newly created entities

No Home Assistant restart is required for the managed dashboards.

## Notes

- Runtime state is stored in `/data/.GoSungrow`.
- The managed dashboard state is stored in `/data/.GoSungrow/dashboard_state.json`.
- If no entities appear, verify MQTT first.
- The app uses the standard iSolarCloud host, app key, and managed dashboard path internally.
- MQTT uses the custom broker settings when `mqtt_host` is set, otherwise it falls back to the Home Assistant MQTT service.
- Managed dashboard text follows Home Assistant language when available (fallback: English).
- If you are updating from an older version with more options, open the app configuration once and save it to clear legacy fields.

## Troubleshooting DNS Errors

If the log contains `lookup gateway.isolarcloud.eu on 127.0.0.11:53: no such host` or `server misbehaving`, the add-on is running but Docker's internal DNS resolver cannot resolve iSolarCloud. GoSungrow keeps the MQTT service alive and retries on the next cycle, but it cannot fetch fresh data until DNS works again.

Suggested checks:

1. In Home Assistant, check `Settings > System > Network` and make sure DNS points to a reliable resolver.
2. Restart the GoSungrow add-on after changing DNS.
3. If other add-ons also fail to resolve internet hostnames, restart Home Assistant OS or the Docker host.
4. If you use Pi-hole, AdGuard, a router DNS proxy, VPN DNS, or custom firewall rules, verify that the Home Assistant host can resolve `gateway.isolarcloud.eu` and `augateway.isolarcloud.com`.

## Troubleshooting Startup JSON Errors

After a sudden power loss, Home Assistant storage can occasionally contain an empty or truncated GoSungrow cache file. If startup logs show `unexpected end of JSON input`, restart the add-on once. GoSungrow removes empty cache files at startup and treats corrupt token or API response cache files as stale data, then logs in and fetches fresh data again.
