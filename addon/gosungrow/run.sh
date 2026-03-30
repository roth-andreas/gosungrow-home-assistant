#!/usr/bin/with-contenv bashio
set -euo pipefail

readonly DEFAULT_HOST="https://augateway.isolarcloud.com"
readonly DEFAULT_APPKEY="B0455FBE7AA0328DB57B59AA729F05D8"
readonly DEFAULT_DASHBOARD_URL_PATH="gosungrow-flow"
readonly DEFAULT_DASHBOARD_TITLE="GoSungrow Flow"

GOSUNGROW_USER="$(bashio::config 'gosungrow_user')"
GOSUNGROW_PASSWORD="$(bashio::config 'gosungrow_password')"
GOSUNGROW_MQTT_HOST="$(bashio::services mqtt 'host' 2>/dev/null || true)"
GOSUNGROW_MQTT_PORT="$(bashio::services mqtt 'port' 2>/dev/null || true)"
GOSUNGROW_MQTT_USER="$(bashio::services mqtt 'username' 2>/dev/null || true)"
GOSUNGROW_MQTT_PASSWORD="$(bashio::services mqtt 'password' 2>/dev/null || true)"

if [ -z "$GOSUNGROW_USER" ]; then
  bashio::log.fatal 'Missing required option: gosungrow_user'
fi

if [ -z "$GOSUNGROW_PASSWORD" ]; then
  bashio::log.fatal 'Missing required option: gosungrow_password'
fi

if [ -z "$GOSUNGROW_MQTT_HOST" ]; then
  bashio::log.fatal 'Home Assistant MQTT service not available. Install and start the Mosquitto broker app first.'
fi

bashio::log.info 'Using Home Assistant MQTT service settings.'

export GOSUNGROW_USER
export GOSUNGROW_PASSWORD
export GOSUNGROW_HOST="$DEFAULT_HOST"
export GOSUNGROW_APPKEY="$DEFAULT_APPKEY"
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

bashio::log.info 'Refreshing iSolarCloud session.'
GoSungrow api login >/dev/null

if bashio::config.true 'install_dashboard'; then
  bashio::log.info "Installing managed Home Assistant dashboard at ${DEFAULT_DASHBOARD_URL_PATH}."
  dashboard_args=(
    ha install-dashboard
    "--asset-dir=$GOSUNGROW_ASSET_DIR"
    "--url-path=${DEFAULT_DASHBOARD_URL_PATH}"
    "--title=${DEFAULT_DASHBOARD_TITLE}"
  )

  if ! GoSungrow "${dashboard_args[@]}"; then
    bashio::log.warning 'Managed dashboard installation failed; continuing without changing Home Assistant dashboards.'
  fi
fi

bashio::log.info "Starting GoSungrow against MQTT broker ${GOSUNGROW_MQTT_HOST}:${GOSUNGROW_MQTT_PORT}."

run_mqtt_with_login_retry() {
  local log_file
  local rc
  local attempt

  attempt=0
  while true; do
    log_file="$(mktemp)"
    set +e
    GoSungrow mqtt run 2>&1 | tee "$log_file"
    rc=${PIPESTATUS[0]}
    set -e

    if [ "$rc" -eq 0 ]; then
      rm -f "$log_file"
      return 0
    fi

    if grep -qiE 'er_token_login_invalid|need to login again' "$log_file"; then
      attempt=$((attempt + 1))
      bashio::log.warning "GoSungrow session expired. Refreshing login and restarting mqtt run (attempt ${attempt})."
      GoSungrow api login >/dev/null
      rm -f "$log_file"
      sleep 2
      continue
    fi

    rm -f "$log_file"
    return "$rc"
  done
}

run_mqtt_with_login_retry
