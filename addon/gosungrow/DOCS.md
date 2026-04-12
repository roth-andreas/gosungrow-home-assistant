# Home Assistant App: GoSungrow

GoSungrow connects to Sungrow iSolarCloud, publishes MQTT discovery data for Home Assistant, and installs managed dashboards for live flow and trends.

## Before You Install

This app requires MQTT.

Before installing `GoSungrow`, make sure Home Assistant already has:

1. a running MQTT broker
2. the `MQTT` integration under `Settings > Devices & services`

For most users, that means installing and starting the `Mosquitto broker` app first.

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

- `install_dashboard`: create or update the managed dashboard automatically
- `dashboard_language`: `auto` (default) or explicit locale (`en`, `de`, `sv`)
- `debug`: enable verbose logging

## What Happens On Startup

On a healthy setup, the app:

1. refreshes the iSolarCloud session
2. installs or updates the managed dashboards
3. connects to MQTT
4. publishes entity discovery and state updates

No Home Assistant restart is required for the managed dashboards.

## Notes

- Runtime state is stored in `/data/.GoSungrow`.
- The managed dashboard state is stored in `/data/.GoSungrow/dashboard_state.json`.
- If no entities appear, verify MQTT first.
- The app uses the standard iSolarCloud host, app key, Home Assistant MQTT service, and managed dashboard path internally.
- Managed dashboard text follows Home Assistant language when available (fallback: English).
- If you are updating from an older version with more options, open the app configuration once and save it to clear legacy fields.
