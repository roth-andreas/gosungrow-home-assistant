# GoSungrow

GoSungrow is maintained here primarily as a Home Assistant app that reads Sungrow iSolarCloud data and publishes it through MQTT discovery.

## Repository focus

- Home Assistant OS app packaging
- MQTT publishing for Home Assistant discovery
- Managed Home Assistant dashboard installation

## CLI note

The Go binary is still available inside the app image and for local development, but the supported deployment path in this repository is the Home Assistant app.

Useful commands:

- `GoSungrow api login`
- `GoSungrow mqtt run`
- `GoSungrow ha install-dashboard`

For installation and app configuration, use the repository `README.md` and `addon/gosungrow/DOCS.md`.
