# Home Assistant Add-on: GoSungrow

This add-on logs in to Sungrow iSolarCloud, publishes MQTT discovery/state data for Home Assistant, and installs a managed dashboard for Sungrow energy flow.

## Before you install

This add-on requires MQTT.

Before installing `GoSungrow`, make sure Home Assistant already has:

1. a running MQTT broker
2. the `MQTT` integration under `Settings > Devices & services`

For most users, that means installing and starting the `Mosquitto broker` add-on first.

## Install

### Option 1: Add custom repository from GitHub

1. Open the Home Assistant Add-on Store.
2. Add this repository as a custom repository:
   - `https://github.com/roth-andreas/gosungrow-home-assistant`
3. Refresh the Add-on Store.
4. Install `GoSungrow`.

### Option 2: Local add-on on the Home Assistant box

1. Copy `addon/gosungrow` to `/addons/gosungrow`.
2. Refresh the Add-on Store.
3. Install `GoSungrow`.

In both cases, Home Assistant will pull:

- `ghcr.io/roth-andreas/gosungrow-addon-{arch}`

## Configuration

Required:

- `gosungrow_user`
- `gosungrow_password`

Recommended:

- `use_homeassistant_mqtt: true`
- leave `mqtt_host`, `mqtt_port`, `mqtt_user`, and `mqtt_password` empty

Other options:

- `gosungrow_host`: defaults to `https://augateway.isolarcloud.com`
- `gosungrow_appkey`: application key used for login requests
- `install_dashboard`: create or update the managed dashboard automatically
- `dashboard_url_path`: URL path for the managed dashboard
- `dashboard_title`: title shown in the Home Assistant sidebar
- `dashboard_force_update`: replace an existing dashboard at the same URL path even if it was edited outside GoSungrow
- `debug`: enable verbose logging

## What happens on startup

On a healthy setup, the add-on:

1. refreshes the iSolarCloud session
2. installs or updates the managed dashboard
3. connects to MQTT
4. starts publishing entity discovery and state

No Home Assistant restart is required for the managed dashboard.

## Migration from the old Docker setup

Old environment variable to add-on option mapping:

- `GOSUNGROW_USER` -> `gosungrow_user`
- `GOSUNGROW_PASSWORD` -> `gosungrow_password`
- `GOSUNGROW_HOST` -> `gosungrow_host`
- `GOSUNGROW_APPKEY` -> `gosungrow_appkey`
- `GOSUNGROW_MQTT_HOST` -> `mqtt_host`
- `GOSUNGROW_MQTT_PORT` -> `mqtt_port`
- `GOSUNGROW_MQTT_USER` -> `mqtt_user`
- `GOSUNGROW_MQTT_PASSWORD` -> `mqtt_password`

Do not run the old Docker container against the same MQTT broker at the same time as this add-on.

## Persistence

Runtime state is stored in:

- `/data/.GoSungrow`

Managed dashboard state is stored in:

- `/data/.GoSungrow/dashboard_state.json`

## Troubleshooting

- If no entities appear, verify MQTT is installed and working first.
- If login fails, check your iSolarCloud credentials and outbound network access.
- If Home Assistant cannot pull the add-on image, verify the GitHub Actions workflow finished and the GHCR image is available.
