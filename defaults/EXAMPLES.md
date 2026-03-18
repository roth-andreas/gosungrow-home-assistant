# Examples

## Home Assistant add-on workflow

1. Configure credentials with Home Assistant add-on options.
2. Start the add-on.
3. Let the add-on publish MQTT discovery entities.
4. Let the add-on install the managed dashboard.

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
