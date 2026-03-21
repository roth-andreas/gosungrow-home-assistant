# Home Assistant Add-on: GoSungrow

GoSungrow connects to Sungrow iSolarCloud, publishes MQTT discovery data for Home Assistant, and installs managed dashboards for live flow and trends.

## Before You Install

This add-on requires MQTT.

Before installing `GoSungrow`, make sure Home Assistant already has:

1. a running MQTT broker
2. the `MQTT` integration under `Settings > Devices & services`

For most users, that means installing and starting the `Mosquitto broker` add-on first.

## Install

1. Open the Home Assistant Add-on Store.
2. Add this repository as a custom repository:
   - `https://github.com/roth-andreas/gosungrow-home-assistant`
3. Refresh the Add-on Store.
4. Install `GoSungrow`.
5. Enter your `gosungrow_user` and `gosungrow_password`.
6. Start the add-on.

## Configuration

Required:

- `gosungrow_user`
- `gosungrow_password`

Optional:

- `install_dashboard`: create or update the managed dashboard automatically
- `debug`: enable verbose logging

## What Happens On Startup

On a healthy setup, the add-on:

1. refreshes the iSolarCloud session
2. installs or updates the managed dashboards
3. connects to MQTT
4. publishes entity discovery and state updates

No Home Assistant restart is required for the managed dashboards.

## Notes

- Runtime state is stored in `/data/.GoSungrow`.
- The managed dashboard state is stored in `/data/.GoSungrow/dashboard_state.json`.
- If no entities appear, verify MQTT first.
- The add-on uses the standard iSolarCloud host, app key, Home Assistant MQTT service, and managed dashboard path internally.
- If you are updating from an older version with more options, open the add-on configuration once and save it to clear legacy fields.
