# GoSungrow Home Assistant Add-on

Custom Home Assistant add-on for importing Sungrow iSolarCloud data into Home Assistant through MQTT discovery.

This repository is based on the original [MickMake/GoSungrow](https://github.com/MickMake/GoSungrow) project and is now maintained here by Andreas Roth with a clear focus on Home Assistant add-on deployment.

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

## 🎯 What You Get

- Sungrow iSolarCloud data in Home Assistant through MQTT discovery
- automatic entity creation in Home Assistant
- an automatically managed `GoSungrow Flow` dashboard
- support for `aarch64` and `amd64`

## ✅ Requirements

- Home Assistant OS or another Home Assistant installation that supports add-ons
- an iSolarCloud account
- outbound network access from Home Assistant to iSolarCloud
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

## 👀 What Happens On First Start

On a healthy setup, the add-on:

1. refreshes the iSolarCloud session
2. installs or updates the managed dashboard
3. connects to MQTT
4. publishes discovery and state messages
5. lets Home Assistant create the entities

## 📊 Visualizations

The add-on installs a managed `GoSungrow Flow` dashboard automatically.

This repository also includes an optional example for Home Assistant's official energy cards:

- `examples/home-assistant-energy-cards.yaml`

## 🙏 Credit

- Original GoSungrow codebase and reverse engineering: MickMake
- Home Assistant add-on packaging and maintenance in this repository: Andreas Roth
