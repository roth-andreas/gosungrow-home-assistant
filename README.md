# GoSungrow Home Assistant Add-on

GoSungrow publishes Sungrow iSolarCloud data into Home Assistant over MQTT discovery.

This repository packages that functionality as a custom Home Assistant add-on. It is based on the original [MickMake/GoSungrow](https://github.com/MickMake/GoSungrow) project and is now maintained here by Andreas Roth.

## What This Repository Is

This repository is for people who want to run GoSungrow on Home Assistant without maintaining a separate Docker deployment on another Linux host.

It provides:

- a Home Assistant add-on definition
- a container image build for the add-on
- GitHub Actions to build and publish images to GHCR
- a repository layout that Home Assistant can consume as a custom add-on source

It does not try to be the general upstream documentation for every historical GoSungrow CLI workflow.

## What You Get

- Sungrow data imported from iSolarCloud
- MQTT discovery entities in Home Assistant
- add-on packaging for `aarch64` and `amd64`
- a GitHub-based publish flow for your own fork

## How It Works

1. The add-on starts the GoSungrow binary inside a Home Assistant add-on container.
2. GoSungrow logs in to the Sungrow iSolarCloud API with your account.
3. The add-on publishes entities to MQTT.
4. Home Assistant discovers those entities through MQTT discovery.

The add-on can either:

- use Home Assistant's MQTT service automatically, or
- publish to a manually configured MQTT broker

## Requirements

- Home Assistant OS or another Home Assistant installation that supports add-ons
- an MQTT broker
- the MQTT integration enabled in Home Assistant
- an iSolarCloud account

Supported add-on architectures in this repository:

- `aarch64`
- `amd64`

## Quick Start

### Option 1: Install from this GitHub repository

1. In Home Assistant, open the Add-on Store.
2. Add this repository as a custom repository:
   - `https://github.com/roth-andreas/gosungrow-home-assistant`
3. Refresh the Add-on Store.
4. Install `GoSungrow`.
5. Open the add-on configuration.
6. Set `gosungrow_user` and `gosungrow_password`.
7. If you already use Home Assistant MQTT or Mosquitto, keep `use_homeassistant_mqtt: true`.
8. Start the add-on.

### Option 2: Install locally on a Home Assistant host

1. Copy `addon/gosungrow` to `/addons/gosungrow`.
2. Refresh the Add-on Store.
3. Install `GoSungrow`.
4. Configure credentials and MQTT settings.
5. Start the add-on.

Important:

- This add-on pulls a prebuilt image from GHCR.
- If image pulls fail, check that the package is published and public.

## Configuration

Required options:

- `gosungrow_user`
- `gosungrow_password`

Available options:

- `gosungrow_user`: iSolarCloud username
- `gosungrow_password`: iSolarCloud password
- `gosungrow_host`: API host, default `https://augateway.isolarcloud.com`
- `gosungrow_appkey`: iSolarCloud app key used by this project
- `use_homeassistant_mqtt`: use Home Assistant's MQTT service wiring
- `mqtt_host`: manual MQTT broker host
- `mqtt_port`: manual MQTT broker port
- `mqtt_user`: manual MQTT username
- `mqtt_password`: manual MQTT password
- `debug`: enable verbose GoSungrow logging

Recommended setup for most users:

- `use_homeassistant_mqtt: true`
- leave `mqtt_host`, `mqtt_port`, `mqtt_user`, and `mqtt_password` at defaults unless you intentionally use an external broker

## First Startup Expectations

After the add-on starts:

1. it logs in to iSolarCloud
2. it connects to MQTT
3. it begins publishing discovery topics and entity data

If entities do not appear immediately, give Home Assistant a short time to process MQTT discovery and then check the add-on logs.

## Repository Layout

Key files:

- `addon/gosungrow/config.yaml`: Home Assistant add-on metadata and options
- `addon/gosungrow/Dockerfile`: add-on image build
- `addon/gosungrow/run.sh`: runtime entrypoint
- `.github/workflows/homeassistant-addon.yml`: validation and image publishing workflow
- `repository.yaml`: custom add-on repository metadata

## Publishing and Releases

This repository is designed to publish add-on images through GitHub Actions.

Workflow:

- `.github/workflows/homeassistant-addon.yml`

Published images:

- `ghcr.io/roth-andreas/gosungrow-addon-aarch64`
- `ghcr.io/roth-andreas/gosungrow-addon-amd64`

The workflow:

1. validates add-on and app version alignment
2. runs `go build .`
3. performs an add-on Docker smoke build
4. publishes images to GHCR

If you fork this repository, update these references:

- `addon/gosungrow/config.yaml`
- `repository.yaml`
- `addon/gosungrow/Dockerfile`

The workflow already publishes to `ghcr.io/${github.repository_owner}/...`, so image publishing follows the repository owner automatically.

## Migration Notes

If you previously ran GoSungrow as a separate Docker container on Raspberry Pi OS:

- do not run the old container and this add-on against the same MQTT broker at the same time
- move your old environment values into add-on options
- keep MQTT topic ownership with only one active publisher

Old environment variable to new option mapping:

- `GOSUNGROW_USER` -> `gosungrow_user`
- `GOSUNGROW_PASSWORD` -> `gosungrow_password`
- `GOSUNGROW_HOST` -> `gosungrow_host`
- `GOSUNGROW_APPKEY` -> `gosungrow_appkey`
- `GOSUNGROW_MQTT_HOST` -> `mqtt_host`
- `GOSUNGROW_MQTT_PORT` -> `mqtt_port`
- `GOSUNGROW_MQTT_USER` -> `mqtt_user`
- `GOSUNGROW_MQTT_PASSWORD` -> `mqtt_password`

## Troubleshooting

### Add-on starts but no entities appear

Check:

- Home Assistant MQTT integration is enabled
- the broker is reachable
- the add-on logs show a successful MQTT connection
- your iSolarCloud credentials are correct

### Home Assistant cannot install or pull the image

Check:

- the GitHub Actions workflow completed successfully
- the GHCR package exists
- the GHCR package visibility allows pulls

### Add-on cannot connect to MQTT

Check:

- `use_homeassistant_mqtt` matches your setup
- manual broker settings are correct if you are not using the Home Assistant MQTT service

### Add-on cannot connect to iSolarCloud

Check:

- username and password
- network connectivity from Home Assistant
- whether the configured API host is still valid for your account region

## Development

Local validation:

```bash
go build .
docker build -f addon/gosungrow/Dockerfile .
```

This repository uses an image-based add-on design. The add-on does not carry a copied application snapshot inside the add-on folder.

## Security

- Do not commit iSolarCloud credentials.
- Do not commit MQTT passwords.
- Do not commit SSH keys.
- Keep secrets in Home Assistant add-on configuration or GitHub secrets only.

## Credit

- Original GoSungrow codebase and reverse engineering: MickMake
- Home Assistant add-on packaging and maintenance in this repository: Andreas Roth
