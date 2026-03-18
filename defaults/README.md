# GoSungrow

GoSungrow is maintained here primarily as a Home Assistant add-on that reads Sungrow iSolarCloud data and publishes it through MQTT discovery.

## Repository focus

- Home Assistant OS add-on packaging
- MQTT publishing for Home Assistant discovery
- Managed Home Assistant dashboard installation

## CLI note

The Go binary is still available inside the add-on image and for local development, but the supported deployment path in this repository is the Home Assistant add-on.

Useful commands:

- `GoSungrow api login`
- `GoSungrow mqtt run`
- `GoSungrow ha install-dashboard`

For installation and add-on configuration, use the repository `README.md` and `addon/gosungrow/DOCS.md`.
