# GoSungrow Home Assistant Add-on

Custom Home Assistant add-on for Sungrow iSolarCloud.

GoSungrow logs in to iSolarCloud, publishes entities to Home Assistant through MQTT discovery, and installs managed dashboards for live flow and trends.

This repository is based on the original [MickMake/GoSungrow](https://github.com/MickMake/GoSungrow) project and is maintained here by Andreas Roth with a focused Home Assistant add-on deployment model.

![Home Assistant Add-on](https://img.shields.io/badge/Home%20Assistant-Add--on-41BDF5?logo=homeassistant&logoColor=white)
![MQTT Required](https://img.shields.io/badge/MQTT-Required-F97316?logo=eclipsemosquitto&logoColor=white)
![Architectures](https://img.shields.io/badge/Arch-aarch64%20%7C%20amd64-16A34A?logo=raspberrypi&logoColor=white)
![Maintained](https://img.shields.io/badge/Maintained%20by-Andreas%20Roth-2563EB)

## Screenshots

<p align="center">
  <img src=".github/assets/dashboard-overview.png" alt="GoSungrow overview dashboard" width="49%" />
  <img src=".github/assets/dashboard-trends.png" alt="GoSungrow trends dashboard" width="49%" />
</p>

## Requirements

- Home Assistant installation with add-on support
- a working MQTT broker and the Home Assistant `MQTT` integration
- an iSolarCloud account
- outbound network access from Home Assistant to iSolarCloud

## Quick Start

1. Install and start the `Mosquitto broker` add-on in Home Assistant.
2. Confirm `MQTT` appears under `Settings > Devices & services`.
3. In the Add-on Store, add this repository as a custom repository:
   - `https://github.com/roth-andreas/gosungrow-home-assistant`
4. Install `GoSungrow`.
5. Enter your `gosungrow_user` and `gosungrow_password`.
6. Leave `use_homeassistant_mqtt: true` unless you intentionally use an external broker.
7. Start the add-on.
8. Open the automatically created `GoSungrow Flow` dashboards from the sidebar.

## What You Get

- MQTT-discovered Sungrow entities in Home Assistant
- a managed `Overview` dashboard for live flow and daily summary
- a managed `Trends` dashboard for deeper energy analysis
- support for `aarch64` and `amd64`

## Configuration

Required:

- `gosungrow_user`
- `gosungrow_password`

Recommended for most users:

- `use_homeassistant_mqtt: true`
- leave `mqtt_host`, `mqtt_port`, `mqtt_user`, and `mqtt_password` empty

Advanced options are documented in `addon/gosungrow/DOCS.md`.

## Notes

- This is not a native Home Assistant integration. MQTT must be working before the add-on starts.
- The add-on manages its own dashboard. If you want to refresh that dashboard after template changes, use `dashboard_force_update: true` for one restart.
- The repository also includes `examples/home-assistant-energy-cards.yaml` if you want to build a dashboard around Home Assistant's official Energy cards.

## Credit

- Original GoSungrow reverse engineering and codebase: MickMake
- Home Assistant add-on packaging and maintenance in this repository: Andreas Roth
