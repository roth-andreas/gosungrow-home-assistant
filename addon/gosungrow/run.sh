#!/usr/bin/with-contenv bashio
set -euo pipefail

readonly DEFAULT_HOST="https://augateway.isolarcloud.com"
readonly DEFAULT_APPKEY="B0455FBE7AA0328DB57B59AA729F05D8"
readonly DEFAULT_DASHBOARD_URL_PATH="gosungrow-flow"
readonly DEFAULT_DASHBOARD_TITLE="GoSungrow Flow"

source /usr/local/lib/gosungrow/recovery_policy.sh
source /usr/local/lib/gosungrow/configuration.sh

if ! gosungrow_read_app_options /data/options.json; then
  bashio::log.fatal 'Could not read valid app options from /data/options.json. Check the GoSungrow app configuration.'
  exit 1
fi

if [ -z "$GOSUNGROW_USER" ]; then
  bashio::log.fatal 'Missing required option: gosungrow_user'
  exit 1
fi

if [ -z "$GOSUNGROW_PASSWORD" ]; then
  bashio::log.fatal 'Missing required option: gosungrow_password'
  exit 1
fi

if ! GOSUNGROW_HOST="$(gosungrow_resolve_host "$GOSUNGROW_HOST" "$DEFAULT_HOST")"; then
  bashio::log.fatal 'Unsupported gosungrow_host. Select a canonical iSolarCloud gateway.'
  exit 1
fi

if [ -n "$GOSUNGROW_MQTT_HOST" ]; then
  bashio::log.info 'Using custom MQTT broker settings.'
else
  bashio::log.info 'Using Home Assistant MQTT service settings.'
  gosungrow_discover_supervisor_mqtt || exit 1
fi

export GOSUNGROW_USER
export GOSUNGROW_PASSWORD
export GOSUNGROW_HOST
export GOSUNGROW_APPKEY="$DEFAULT_APPKEY"
export GOSUNGROW_MQTT_HOST
export GOSUNGROW_MQTT_PORT
export GOSUNGROW_MQTT_USER
export GOSUNGROW_MQTT_PASSWORD
export GOSUNGROW_DEBUG
export GOSUNGROW_ASSET_DIR="${GOSUNGROW_ASSET_DIR:-/opt/gosungrow/assets}"
export GOSUNGROW_DASHBOARD_LANGUAGE
export GOSUNGROW_DASHBOARD_ENABLED
export GOSUNGROW_DASHBOARD_FORCE_UPDATE

mkdir -p "$(dirname "$GOSUNGROW_CONFIG")"

GoSungrow config write \
  --dashboard-enabled="$GOSUNGROW_DASHBOARD_ENABLED" \
  --dashboard-force-update="$GOSUNGROW_DASHBOARD_FORCE_UPDATE" \
  --host="$GOSUNGROW_HOST" \
  --appkey="$GOSUNGROW_APPKEY" \
  --user="$GOSUNGROW_USER" \
  --password="$GOSUNGROW_PASSWORD" >/dev/null

find "$(dirname "$GOSUNGROW_CONFIG")" -maxdepth 1 -type f -name '*.json' -size 0 -delete 2>/dev/null || true

install_managed_dashboard() {
  local action
  local dashboard_log
  local dashboard_rc
  local failure_class
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

  if [ "$GOSUNGROW_DASHBOARD_FORCE_UPDATE" = true ]; then
    dashboard_args+=("--force-update")
  fi

  dashboard_log="$(mktemp)"
  set +e
  GoSungrow "${dashboard_args[@]}" 2>&1 | tee "$dashboard_log"
  dashboard_rc=${PIPESTATUS[0]}
  set -e

  if [ "$dashboard_rc" -ne 0 ]; then
    failure_class="$(gosungrow_failure_class_from_log "$dashboard_log")"
    rm -f "$dashboard_log"
    if [ "$failure_class" = "operator_action_required" ]; then
      bashio::log.warning "Managed dashboard ${action} needs operator action: restart Home Assistant Core once. Restarting the GoSungrow app is insufficient; reconciliation will retry automatically. MQTT startup will continue."
    else
      bashio::log.warning "Managed dashboard ${action} failed; continuing without changing Home Assistant dashboards."
    fi
    return "$dashboard_rc"
  fi
  rm -f "$dashboard_log"
}

if [ "$GOSUNGROW_DASHBOARD_ENABLED" = true ]; then
  install_managed_dashboard "Installing" || true
fi

bashio::log.info "Starting GoSungrow against MQTT broker ${GOSUNGROW_MQTT_HOST}:${GOSUNGROW_MQTT_PORT}."

run_mqtt_with_login_retry() {
  local log_file
  local rc
  local attempt
  local mqtt_args
  local failure_class
  local retry_policy
  local retry_delay

  mqtt_args=(
    mqtt run
    "--mqtt-host=$GOSUNGROW_MQTT_HOST"
    "--mqtt-port=$GOSUNGROW_MQTT_PORT"
    "--mqtt-user=$GOSUNGROW_MQTT_USER"
    "--mqtt-password=$GOSUNGROW_MQTT_PASSWORD"
    "--dashboard-enabled=$GOSUNGROW_DASHBOARD_ENABLED"
    "--dashboard-force-update=$GOSUNGROW_DASHBOARD_FORCE_UPDATE"
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

    failure_class="$(gosungrow_failure_class_from_log "$log_file")"
    retry_policy="$(gosungrow_retry_policy "$failure_class")"
    case "$retry_policy" in
      docker_dns)
        attempt=$((attempt + 1))
        case "$attempt" in
          1) retry_delay=15 ;;
          2) retry_delay=30 ;;
          3) retry_delay=60 ;;
          4) retry_delay=120 ;;
          *) retry_delay=300 ;;
        esac
        bashio::log.warning "Docker DNS is unavailable before GoSungrow finished MQTT initialization. Retrying in ${retry_delay}s (attempt ${attempt})."
        ;;
      recoverable_remote)
        attempt=$((attempt + 1))
        retry_delay=$(( attempt < 10 ? attempt * 3 : 30 ))
        bashio::log.warning "Recoverable GoSungrow runtime error. Refreshing login and restarting mqtt run (attempt ${attempt})."
        GoSungrow api login >/dev/null || true
        ;;
      *)
        rm -f "$log_file"
        return "$rc"
        ;;
    esac

    rm -f "$log_file"
    sleep "$retry_delay"
  done
}

run_mqtt_with_login_retry
