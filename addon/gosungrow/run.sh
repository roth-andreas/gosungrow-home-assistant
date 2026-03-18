#!/usr/bin/with-contenv bashio
set -euo pipefail

readonly DEFAULT_HOST="https://augateway.isolarcloud.com"
readonly DEFAULT_APPKEY="B0455FBE7AA0328DB57B59AA729F05D8"

GOSUNGROW_USER="$(bashio::config 'gosungrow_user')"
GOSUNGROW_PASSWORD="$(bashio::config 'gosungrow_password')"
GOSUNGROW_HOST="$(bashio::config 'gosungrow_host')"
GOSUNGROW_APPKEY="$(bashio::config 'gosungrow_appkey')"
GOSUNGROW_MQTT_HOST="$(bashio::config 'mqtt_host')"
GOSUNGROW_MQTT_PORT="$(bashio::config 'mqtt_port')"
GOSUNGROW_MQTT_USER="$(bashio::config 'mqtt_user')"
GOSUNGROW_MQTT_PASSWORD="$(bashio::config 'mqtt_password')"
GOSUNGROW_DASHBOARD_URL_PATH="$(bashio::config 'dashboard_url_path')"
GOSUNGROW_DASHBOARD_TITLE="$(bashio::config 'dashboard_title')"

if bashio::config.true 'use_homeassistant_mqtt'; then
  service_host="$(bashio::services mqtt 'host' 2>/dev/null || true)"
  if [ -n "$service_host" ]; then
    GOSUNGROW_MQTT_HOST="$service_host"
    GOSUNGROW_MQTT_PORT="$(bashio::services mqtt 'port' 2>/dev/null || true)"
    GOSUNGROW_MQTT_USER="$(bashio::services mqtt 'username' 2>/dev/null || true)"
    GOSUNGROW_MQTT_PASSWORD="$(bashio::services mqtt 'password' 2>/dev/null || true)"
    bashio::log.info 'Using Home Assistant MQTT service settings.'
  else
    bashio::log.warning 'Home Assistant MQTT service not available; using manual MQTT settings from add-on options.'
  fi
fi

if [ -z "$GOSUNGROW_USER" ]; then
  bashio::log.fatal 'Missing required option: gosungrow_user'
fi

if [ -z "$GOSUNGROW_PASSWORD" ]; then
  bashio::log.fatal 'Missing required option: gosungrow_password'
fi

if [ -z "$GOSUNGROW_MQTT_HOST" ]; then
  bashio::log.fatal 'No MQTT host configured. Enable use_homeassistant_mqtt with Mosquitto, or set mqtt_host manually.'
fi

export GOSUNGROW_USER
export GOSUNGROW_PASSWORD
export GOSUNGROW_HOST="${GOSUNGROW_HOST:-$DEFAULT_HOST}"
export GOSUNGROW_APPKEY="${GOSUNGROW_APPKEY:-$DEFAULT_APPKEY}"
export GOSUNGROW_MQTT_HOST
export GOSUNGROW_MQTT_PORT="${GOSUNGROW_MQTT_PORT:-1883}"
export GOSUNGROW_MQTT_USER
export GOSUNGROW_MQTT_PASSWORD
export GOSUNGROW_DEBUG="$(bashio::config 'debug')"
export GOSUNGROW_ASSET_DIR="${GOSUNGROW_ASSET_DIR:-/opt/gosungrow/assets}"

mkdir -p "$(dirname "$GOSUNGROW_CONFIG")"

GoSungrow config write \
  --host="$GOSUNGROW_HOST" \
  --appkey="$GOSUNGROW_APPKEY" \
  --user="$GOSUNGROW_USER" \
  --password="$GOSUNGROW_PASSWORD" >/dev/null

if bashio::config.true 'install_dashboard'; then
  bashio::log.info "Installing managed Home Assistant dashboard at ${GOSUNGROW_DASHBOARD_URL_PATH:-gosungrow-flow}."
  dashboard_args=(
    ha install-dashboard
    "--asset-dir=$GOSUNGROW_ASSET_DIR"
    "--url-path=${GOSUNGROW_DASHBOARD_URL_PATH:-gosungrow-flow}"
    "--title=${GOSUNGROW_DASHBOARD_TITLE:-GoSungrow Flow}"
  )

  if bashio::config.true 'dashboard_force_update'; then
    dashboard_args+=("--force-update")
  fi

  if ! GoSungrow "${dashboard_args[@]}"; then
    bashio::log.warning 'Managed dashboard installation failed; continuing without changing Home Assistant dashboards.'
  fi
fi

bashio::log.info "Starting GoSungrow against MQTT broker ${GOSUNGROW_MQTT_HOST}:${GOSUNGROW_MQTT_PORT}."
exec GoSungrow mqtt run
