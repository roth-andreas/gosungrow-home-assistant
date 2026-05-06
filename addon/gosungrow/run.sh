#!/usr/bin/with-contenv bashio
set -euo pipefail

readonly DEFAULT_HOST="https://augateway.isolarcloud.com"
readonly DEFAULT_APPKEY="B0455FBE7AA0328DB57B59AA729F05D8"
readonly DEFAULT_DASHBOARD_URL_PATH="gosungrow-flow"
readonly DEFAULT_DASHBOARD_TITLE="GoSungrow Flow"

optional_config() {
  local value
  value="$(bashio::config "$1")"
  if [ "$value" = "null" ] || [ "$value" = "~" ]; then
    value=""
  fi
  printf '%s' "$value"
}

GOSUNGROW_USER="$(bashio::config 'gosungrow_user')"
GOSUNGROW_PASSWORD="$(bashio::config 'gosungrow_password')"
CUSTOM_MQTT_HOST="$(optional_config 'mqtt_host')"
CUSTOM_MQTT_PORT="$(optional_config 'mqtt_port')"
CUSTOM_MQTT_USER="$(optional_config 'mqtt_username')"
CUSTOM_MQTT_PASSWORD="$(optional_config 'mqtt_password')"
SUPERVISOR_MQTT_HOST="$(bashio::services mqtt 'host' 2>/dev/null || true)"
SUPERVISOR_MQTT_PORT="$(bashio::services mqtt 'port' 2>/dev/null || true)"
SUPERVISOR_MQTT_USER="$(bashio::services mqtt 'username' 2>/dev/null || true)"
SUPERVISOR_MQTT_PASSWORD="$(bashio::services mqtt 'password' 2>/dev/null || true)"

if [ -n "$CUSTOM_MQTT_HOST" ]; then
  GOSUNGROW_MQTT_HOST="$CUSTOM_MQTT_HOST"
  GOSUNGROW_MQTT_PORT="${CUSTOM_MQTT_PORT:-1883}"
  GOSUNGROW_MQTT_USER="$CUSTOM_MQTT_USER"
  GOSUNGROW_MQTT_PASSWORD="$CUSTOM_MQTT_PASSWORD"
else
  GOSUNGROW_MQTT_HOST="$SUPERVISOR_MQTT_HOST"
  GOSUNGROW_MQTT_PORT="$SUPERVISOR_MQTT_PORT"
  GOSUNGROW_MQTT_USER="$SUPERVISOR_MQTT_USER"
  GOSUNGROW_MQTT_PASSWORD="$SUPERVISOR_MQTT_PASSWORD"
fi

if [ -z "$GOSUNGROW_USER" ]; then
  bashio::log.fatal 'Missing required option: gosungrow_user'
fi

if [ -z "$GOSUNGROW_PASSWORD" ]; then
  bashio::log.fatal 'Missing required option: gosungrow_password'
fi

if [ -z "$GOSUNGROW_MQTT_HOST" ]; then
  bashio::log.fatal 'No MQTT broker configured. Set mqtt_host in the app config or install/start the Home Assistant Mosquitto broker app.'
fi

if [ -n "$CUSTOM_MQTT_HOST" ]; then
  bashio::log.info 'Using custom MQTT broker settings.'
else
  bashio::log.info 'Using Home Assistant MQTT service settings.'
fi

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
export GOSUNGROW_DASHBOARD_LANGUAGE="$(bashio::config 'dashboard_language')"

mkdir -p "$(dirname "$GOSUNGROW_CONFIG")"

GoSungrow config write \
  --host="$GOSUNGROW_HOST" \
  --appkey="$GOSUNGROW_APPKEY" \
  --user="$GOSUNGROW_USER" \
  --password="$GOSUNGROW_PASSWORD" >/dev/null

find "$(dirname "$GOSUNGROW_CONFIG")" -maxdepth 1 -type f -name '*.json' -size 0 -delete 2>/dev/null || true

bashio::log.info 'Refreshing iSolarCloud session.'
GoSungrow api login >/dev/null

install_managed_dashboard() {
  local action
  action="${1:-Installing}"

  bashio::log.info "${action} managed Home Assistant dashboard at ${DEFAULT_DASHBOARD_URL_PATH}."
  local dashboard_args=(
    ha install-dashboard
    "--asset-dir=$GOSUNGROW_ASSET_DIR"
    "--url-path=${DEFAULT_DASHBOARD_URL_PATH}"
    "--title=${DEFAULT_DASHBOARD_TITLE}"
    "--language=${GOSUNGROW_DASHBOARD_LANGUAGE:-auto}"
    "--diagnostic-context=${action}"
  )

  if ! GoSungrow "${dashboard_args[@]}"; then
    bashio::log.warning "Managed dashboard ${action} failed; continuing without changing Home Assistant dashboards."
    return 1
  fi
}

reconcile_managed_dashboard_after_mqtt_start() {
  local attempt
  local delay

  attempt=0
  for delay in ${GOSUNGROW_DASHBOARD_RECONCILE_DELAYS:-30 90 180 300}; do
    sleep "$delay" || return 0
    attempt=$((attempt + 1))
    install_managed_dashboard "Reconciling after MQTT startup (${attempt})" || true
  done
}

dashboard_reconcile_pid=""
if bashio::config.true 'install_dashboard'; then
  install_managed_dashboard "Installing" || true
  reconcile_managed_dashboard_after_mqtt_start &
  dashboard_reconcile_pid="$!"
  trap 'if [ -n "${dashboard_reconcile_pid:-}" ]; then kill "$dashboard_reconcile_pid" 2>/dev/null || true; fi' EXIT
fi

bashio::log.info "Starting GoSungrow against MQTT broker ${GOSUNGROW_MQTT_HOST}:${GOSUNGROW_MQTT_PORT}."

run_mqtt_with_login_retry() {
  local log_file
  local rc
  local attempt
  local mqtt_args

  mqtt_args=(
    mqtt run
    "--mqtt-host=$GOSUNGROW_MQTT_HOST"
    "--mqtt-port=${GOSUNGROW_MQTT_PORT:-1883}"
    "--mqtt-user=$GOSUNGROW_MQTT_USER"
    "--mqtt-password=$GOSUNGROW_MQTT_PASSWORD"
  )

  attempt=0
  while true; do
    log_file="$(mktemp)"
    set +e
    GoSungrow "${mqtt_args[@]}" 2>&1 | tee "$log_file"
    rc=${PIPESTATUS[0]}
    set -e

    if [ "$rc" -eq 0 ]; then
      rm -f "$log_file"
      return 0
    fi

    if grep -qiE 'panic: runtime error|fatal error:' "$log_file"; then
      bashio::log.warning "Non-recoverable local GoSungrow runtime error. Not refreshing login; exiting so the underlying error remains visible."
      rm -f "$log_file"
      return "$rc"
    fi

    if grep -qiE 'er_token_login_invalid|need to login again|API httpResponse is 5[0-9]{2}|internal server error|bad gateway|service unavailable|gateway timeout|no such host|temporary failure in name resolution|server misbehaving|network is unreachable|connection refused|context deadline exceeded|i/o timeout' "$log_file"; then
      attempt=$((attempt + 1))
      bashio::log.warning "Recoverable GoSungrow runtime error. Refreshing login and restarting mqtt run (attempt ${attempt})."
      GoSungrow api login >/dev/null || true
      rm -f "$log_file"
      sleep 3
      continue
    fi

    rm -f "$log_file"
    return "$rc"
  done
}

run_mqtt_with_login_retry
