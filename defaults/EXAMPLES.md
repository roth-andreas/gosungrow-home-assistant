# Examples

## Home Assistant app workflow

1. Configure credentials with Home Assistant app options.
2. Start the app.
3. Let the app publish MQTT discovery entities.
4. Let the app install the managed dashboard.

## Local development examples

```bash
GoSungrow api login
GoSungrow mqtt run
GoSungrow ha install-dashboard --url-path=gosungrow-flow
```

## Manual MQTT example

```bash
GoSungrow mqtt run --mqtt-host=broker.local --mqtt-port=1883
```
