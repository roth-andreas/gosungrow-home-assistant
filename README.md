# GoSungrow Home Assistant Add-on

Custom Home Assistant add-on for importing Sungrow iSolarCloud data into Home Assistant through MQTT discovery.

This repository is based on the original [MickMake/GoSungrow](https://github.com/MickMake/GoSungrow) project and is maintained here by Andreas Roth with a clear focus on Home Assistant add-on deployment.

![Home Assistant Add-on](https://img.shields.io/badge/Home%20Assistant-Add--on-41BDF5?logo=homeassistant&logoColor=white)
![MQTT Required](https://img.shields.io/badge/MQTT-Required-F97316?logo=eclipsemosquitto&logoColor=white)
![Architectures](https://img.shields.io/badge/Arch-aarch64%20%7C%20amd64-16A34A?logo=raspberrypi&logoColor=white)
![Maintained](https://img.shields.io/badge/Maintained%20by-Andreas%20Roth-2563EB)

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

Detailed setup, troubleshooting, and maintainer notes are below.

## ⚠️ Read This First

> [!IMPORTANT]
> This add-on depends on MQTT. Install and verify MQTT in Home Assistant before you install `GoSungrow`.

Before you install `GoSungrow`, you must already have these working in Home Assistant:

1. an MQTT broker
2. the Home Assistant `MQTT` integration

For most users that means:

1. install the `Mosquitto broker` add-on
2. start it
3. confirm Home Assistant shows `MQTT` under `Settings > Devices & services`
4. only then install `GoSungrow`

If MQTT is not installed first, this add-on has nowhere to publish its entities.

## 🎯 What This Fork Is For

Use this repository if you want to:

- run GoSungrow directly on Home Assistant as a custom add-on
- get Sungrow entities into Home Assistant through MQTT discovery
- avoid maintaining a separate Docker container on another machine

This repository is intentionally centered on the Home Assistant add-on use case. It is not meant to be the full historical upstream CLI/API manual.

## ✨ Why This Fork Exists

The original GoSungrow project is the technical base. This fork packages it in a way that is practical for Home Assistant users today:

- add-on definition for Home Assistant
- GitHub Actions publishing to GHCR
- current repository structure for custom add-on installation
- active maintenance for the Home Assistant deployment path

## 🔄 How It Works

1. The add-on logs in to Sungrow iSolarCloud.
2. It reads your plant and device data.
3. It publishes discovery and state messages to MQTT.
4. Home Assistant creates entities from those MQTT messages.

That means this is not a native Home Assistant integration. MQTT is the transport layer between GoSungrow and Home Assistant.

## 📊 Visualizations

This repository includes two ways to build Sungrow dashboards in Home Assistant:

- a modern setup based on Home Assistant's official energy cards
- a custom Sungrow flow card using the included image assets

Start here:

- `docs/examples/home-assistant-energy-cards.yaml`
- `docs/examples/home-assistant-sungrow-flow.yaml`

Example dashboard from the original GoSungrow Lovelace setup:

![Sungrow Home Assistant dashboard example](docs/SunGrowOnHASSIO1.png)

## ✅ Supported Setup

This repository is aimed at:

- Home Assistant OS
- Home Assistant installations that support add-ons
- `aarch64` and `amd64`

You also need:

- an iSolarCloud account
- working network access from Home Assistant to iSolarCloud
- MQTT working in Home Assistant before you start this add-on

## 🚀 Installation

### Recommended Path

> [!TIP]
> If you already use the `Mosquitto broker` add-on and Home Assistant already shows the `MQTT` integration, keep `use_homeassistant_mqtt: true` and leave the manual MQTT fields empty.

1. In Home Assistant, install and start `Mosquitto broker` if you do not already have an MQTT broker.
2. Confirm `MQTT` appears under `Settings > Devices & services`.
3. Open the Add-on Store.
4. Add this repository as a custom repository:
   - `https://github.com/roth-andreas/gosungrow-home-assistant`
5. Refresh the Add-on Store.
6. Install `GoSungrow`.
7. Open the add-on configuration.
8. Set:
   - `gosungrow_user`
   - `gosungrow_password`
9. Keep `use_homeassistant_mqtt: true` unless you intentionally use an external MQTT broker.
10. Start the add-on.

### Local Add-on Installation

If you want to install it from files instead of from GitHub:

1. copy `addon/gosungrow` to `/addons/gosungrow`
2. refresh the Add-on Store
3. install the add-on
4. configure credentials
5. start it

This add-on still pulls its container image from GHCR.

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
- `debug`: verbose logging

Use the manual MQTT fields only if you are not using Home Assistant's built-in MQTT service wiring.

## 👀 First Start: What To Expect

On a healthy setup, the sequence is:

1. the add-on starts
2. it connects to iSolarCloud
3. it connects to MQTT
4. Home Assistant begins discovering entities

If entities do not appear immediately, check the add-on logs before changing configuration. The two most common causes are:

- MQTT was not installed first
- iSolarCloud credentials are wrong

## 🩺 Common Problems

### The add-on starts but no entities appear

Check:

- `Mosquitto broker` or another MQTT broker is actually running
- `MQTT` is present in `Settings > Devices & services`
- `use_homeassistant_mqtt` matches your intended setup
- the add-on logs show a successful MQTT connection

### The add-on shows MQTT errors

Check:

- you installed MQTT before GoSungrow
- Home Assistant has a working broker connection
- you did not leave wrong manual MQTT values in the add-on config

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

## 🔁 For Users Of The Old Docker Setup

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

## 🧰 For Maintainers And Forks

Important files:

- `addon/gosungrow/config.yaml`
- `addon/gosungrow/Dockerfile`
- `addon/gosungrow/run.sh`
- `.github/workflows/homeassistant-addon.yml`
- `repository.yaml`

Published images:

- `ghcr.io/roth-andreas/gosungrow-addon-aarch64`
- `ghcr.io/roth-andreas/gosungrow-addon-amd64`

If you fork this repository, update:

- `addon/gosungrow/config.yaml`
- `repository.yaml`
- `addon/gosungrow/Dockerfile`

The workflow already publishes to `ghcr.io/${github.repository_owner}/...`.

## 🛠️ Development

Local checks:

```bash
go build .
docker build -f addon/gosungrow/Dockerfile .
```

## 🔐 Security

- do not commit iSolarCloud credentials
- do not commit MQTT passwords
- do not commit SSH keys
- keep secrets in Home Assistant add-on configuration or GitHub secrets

## 🙏 Credit

- Original GoSungrow codebase and reverse engineering: MickMake
- Home Assistant add-on packaging and maintenance in this repository: Andreas Roth
