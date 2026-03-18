# Home Assistant Add-on: GoSungrow

This add-on runs GoSungrow inside Home Assistant OS and publishes Sungrow iSolarCloud entities through MQTT discovery.

## What changed from the old Raspberry Pi Docker setup

The old deployment used Docker Compose and `.env` values on Raspberry Pi OS.
Home Assistant OS does not support arbitrary host Docker workloads, so this add-on replaces that setup.
This add-on pulls a prebuilt image from GHCR instead of building from a copied source snapshot inside the add-on folder.

## Install

### Option 1: Local add-on on the Pi

1. Copy the folder `addon/gosungrow` to `/addons/gosungrow` on your Home Assistant OS device.
2. Refresh the add-on store.
3. Install `GoSungrow`.
4. Home Assistant will pull `ghcr.io/roth-andreas/gosungrow-addon-{arch}` for your architecture.

### Option 2: Custom add-on repository from GitHub

1. Push this repo somewhere Home Assistant can reach.
2. Add the repository URL in the Home Assistant add-on store.
3. Install `GoSungrow` from that repository.

## Configuration

- `gosungrow_user`: Your iSolarCloud username.
- `gosungrow_password`: Your iSolarCloud password.
- `use_homeassistant_mqtt`: Recommended. If enabled, the add-on reads MQTT settings from Home Assistant's MQTT service.
- `mqtt_host`, `mqtt_port`, `mqtt_user`, `mqtt_password`: Only needed if you are not using the Home Assistant MQTT service.
- `gosungrow_host`: Defaults to `https://augateway.isolarcloud.com`.
- `gosungrow_appkey`: Defaults to the app key currently used by this project.
- `install_dashboard`: If enabled, the add-on creates or updates a managed Lovelace dashboard with embedded flow assets. No Home Assistant restart is required.
- `dashboard_url_path`: URL path used for the managed dashboard.
- `dashboard_title`: Sidebar title for the managed dashboard.
- `dashboard_force_update`: Replace an existing dashboard at the same URL path even if it was edited outside GoSungrow.
- `debug`: Enables GoSungrow debug mode.

## Image publishing

The production image is published by GitHub Actions from this repo:

- Workflow: `.github/workflows/homeassistant-addon.yml`
- Image: `ghcr.io/roth-andreas/gosungrow-addon-{arch}`

If you fork this repo, change the `image` field in `addon/gosungrow/config.yaml` to your own registry path.

## Migration from the old `.env`

Old `.env` key to new add-on option mapping:

- `GOSUNGROW_USER` -> `gosungrow_user`
- `GOSUNGROW_PASSWORD` -> `gosungrow_password`
- `GOSUNGROW_HOST` -> `gosungrow_host`
- `GOSUNGROW_APPKEY` -> `gosungrow_appkey`
- `GOSUNGROW_MQTT_HOST` -> `mqtt_host`
- `GOSUNGROW_MQTT_PORT` -> `mqtt_port`
- `GOSUNGROW_MQTT_USER` -> `mqtt_user`
- `GOSUNGROW_MQTT_PASSWORD` -> `mqtt_password`

## Persistence

Runtime state is stored in `/data/.GoSungrow` inside the add-on data volume.
Managed dashboard state is stored in `/data/.GoSungrow/dashboard_state.json`.

## Notes

- Do not leave the old Docker container running against the same MQTT broker at the same time.
- If you use Home Assistant's Mosquitto add-on, keep `use_homeassistant_mqtt: true` and leave the manual MQTT fields empty.
- The add-on uses Home Assistant's websocket API to create a storage-mode dashboard, so a Home Assistant restart is not required.
- For local development, build the image from the repo root with `docker build -f addon/gosungrow/Dockerfile .`.
