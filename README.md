# GoSungrow Home Assistant Add-on

Custom Home Assistant add-on for importing Sungrow iSolarCloud data into Home Assistant through MQTT discovery.

This repository is based on the original [MickMake/GoSungrow](https://github.com/MickMake/GoSungrow) project and is maintained here by Andreas Roth with a clear focus on Home Assistant add-on deployment.

![Home Assistant Add-on](https://img.shields.io/badge/Home%20Assistant-Add--on-41BDF5?logo=homeassistant&logoColor=white)
![MQTT Required](https://img.shields.io/badge/MQTT-Required-F97316?logo=eclipsemosquitto&logoColor=white)
![Architectures](https://img.shields.io/badge/Arch-aarch64%20%7C%20amd64-16A34A?logo=raspberrypi&logoColor=white)
![Maintained](https://img.shields.io/badge/Maintained%20by-Andreas%20Roth-2563EB)

## ⚠️ Read This First

> [!IMPORTANT]
> Install and verify MQTT in Home Assistant before you install `GoSungrow`.

This add-on is not a native Home Assistant integration. It publishes entities through MQTT discovery, so you need:

1. a running MQTT broker
2. the Home Assistant `MQTT` integration

For most users that means:

1. install and start the `Mosquitto broker` add-on
2. confirm `MQTT` appears under `Settings > Devices & services`
3. only then install `GoSungrow`

## ⚡ Quick Installation

If you want the shortest path, do this:

1. Install and start the `Mosquitto broker` add-on in Home Assistant.
2. Confirm `MQTT` appears under `Settings > Devices & services`.
3. Open the Add-on Store and add this repository as a custom repository:
   - `https://github.com/roth-andreas/gosungrow-home-assistant`
4. Install `GoSungrow`.
5. Enter:
   - `gosungrow_user`
   - `gosungrow_password`
6. Leave `use_homeassistant_mqtt: true`.
7. Start the add-on.
8. Open the automatically created `GoSungrow Flow` dashboard.

## 🎯 What This Repository Is For

Use this repository if you want to:

- run GoSungrow directly on Home Assistant as a custom add-on
- publish Sungrow entities to Home Assistant through MQTT discovery
- get an automatically managed Home Assistant dashboard for Sungrow energy flows

This repository is intentionally centered on the Home Assistant add-on use case. It is not trying to preserve the full historical upstream CLI/API surface.

## 🔄 How It Works

1. The add-on logs in to Sungrow iSolarCloud.
2. It discovers your plant and device metadata.
3. It publishes MQTT discovery and state messages.
4. It installs or updates a managed Home Assistant dashboard over the Home Assistant websocket API.
5. Home Assistant creates the entities from MQTT.

## 📊 Visualizations

The add-on installs a managed `GoSungrow Flow` dashboard automatically.

The repository also includes an optional example for Home Assistant's official energy cards:

- `examples/home-assistant-energy-cards.yaml`

## ✅ Supported Setup

This repository is aimed at:

- Home Assistant OS
- Home Assistant installations that support add-ons
- `aarch64` and `amd64`

You also need:

- an iSolarCloud account
- working outbound network access from Home Assistant to iSolarCloud
- MQTT working in Home Assistant before the add-on starts

## ⚙️ Configuration

Required:

- `gosungrow_user`
- `gosungrow_password`

Recommended for most users:

- `use_homeassistant_mqtt: true`
- leave `mqtt_host`, `mqtt_port`, `mqtt_user`, and `mqtt_password` empty

Available options:

- `gosungrow_user`: iSolarCloud username
- `gosungrow_password`: iSolarCloud password
- `gosungrow_host`: iSolarCloud API host
- `gosungrow_appkey`: application key used for login requests
- `use_homeassistant_mqtt`: use Home Assistant's MQTT service wiring
- `mqtt_host`: manual MQTT broker host
- `mqtt_port`: manual MQTT broker port
- `mqtt_user`: manual MQTT username
- `mqtt_password`: manual MQTT password
- `install_dashboard`: create or update the managed `GoSungrow Flow` dashboard automatically
- `dashboard_url_path`: URL path used for the managed dashboard
- `dashboard_title`: sidebar title used for the managed dashboard
- `dashboard_force_update`: replace an existing dashboard at the same URL path even if it was edited outside GoSungrow
- `debug`: verbose logging

## 👀 First Start: What To Expect

On a healthy setup, the sequence is:

1. the add-on starts
2. it refreshes the iSolarCloud session
3. it installs or updates the managed dashboard
4. it connects to MQTT
5. Home Assistant begins discovering entities

If entities do not appear immediately, check the add-on logs before changing configuration.

## 🩺 Troubleshooting

### The add-on starts but no entities appear

Check:

- `Mosquitto broker` or another MQTT broker is actually running
- `MQTT` is present in `Settings > Devices & services`
- `use_homeassistant_mqtt` matches your intended setup
- the add-on logs show a successful MQTT connection

### The add-on shows MQTT errors

Check:

- MQTT was installed before GoSungrow
- Home Assistant has a working broker connection
- you did not leave incorrect manual MQTT values in the add-on config

### The add-on shows iSolarCloud login errors

Check:

- username and password
- outbound network connectivity from Home Assistant
- whether the configured API host matches your region if you changed it manually

### Home Assistant cannot pull the add-on image

Check:

- the GitHub Actions workflow completed successfully
- the GHCR package exists
- the GHCR package visibility allows Home Assistant to pull it

## 🔁 Migration From The Old Docker Setup

If you previously ran GoSungrow as a separate Docker container on Raspberry Pi OS:

- stop the old container before using this add-on
- do not let both publish to the same MQTT broker
- move your old environment values into add-on configuration

Environment variable mapping:

- `GOSUNGROW_USER` -> `gosungrow_user`
- `GOSUNGROW_PASSWORD` -> `gosungrow_password`
- `GOSUNGROW_HOST` -> `gosungrow_host`
- `GOSUNGROW_APPKEY` -> `gosungrow_appkey`
- `GOSUNGROW_MQTT_HOST` -> `mqtt_host`
- `GOSUNGROW_MQTT_PORT` -> `mqtt_port`
- `GOSUNGROW_MQTT_USER` -> `mqtt_user`
- `GOSUNGROW_MQTT_PASSWORD` -> `mqtt_password`

## 🧪 Development And Validation

Local validation commands:

```bash
go test ./...
go build .
bash -n addon/gosungrow/run.sh
docker build \
  --build-arg BUILD_FROM=ghcr.io/home-assistant/amd64-base:latest \
  --build-arg BUILD_ARCH=amd64 \
  --build-arg BUILD_VERSION=test \
  -f addon/gosungrow/Dockerfile \
  .
```

The GitHub Actions workflow runs the same checks before publishing images:

- version alignment
- `go build .`
- `go test ./...`
- `bash -n addon/gosungrow/run.sh`
- add-on Docker smoke build

## 🚀 Publishing And Release Flow

Important files:

- `addon/gosungrow/config.yaml`
- `addon/gosungrow/Dockerfile`
- `addon/gosungrow/run.sh`
- `.github/workflows/homeassistant-addon.yml`
- `repository.yaml`

Published images:

- `ghcr.io/roth-andreas/gosungrow-addon-aarch64`
- `ghcr.io/roth-andreas/gosungrow-addon-amd64`

When maintainers change runtime behavior, the add-on version in `addon/gosungrow/config.yaml` should be bumped so Home Assistant can pull a fresh image.

## 🔐 Security

- do not commit iSolarCloud credentials
- do not commit MQTT passwords
- do not commit SSH keys
- keep secrets in Home Assistant add-on configuration or GitHub secrets

## 🙏 Credit

- Original GoSungrow codebase and reverse engineering: MickMake
- Home Assistant add-on packaging and maintenance in this repository: Andreas Roth
