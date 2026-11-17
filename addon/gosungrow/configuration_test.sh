#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$script_dir/configuration.sh"

readonly default_host="https://augateway.isolarcloud.com"

assert_equal() {
  local want
  local got
  local scenario

  want="$1"
  got="$2"
  scenario="$3"
  if [ "$got" != "$want" ]; then
    printf '%s: got %s, want %s\n' "$scenario" "$got" "$want" >&2
    exit 1
  fi
}

assert_equal "$default_host" "$(gosungrow_resolve_host '' "$default_host")" 'empty host uses default'
assert_equal "$default_host" "$(gosungrow_resolve_host '   ' "$default_host")" 'whitespace host uses default'

for supported_host in \
  'https://augateway.isolarcloud.com' \
  'https://gateway.isolarcloud.com' \
  'https://gateway.isolarcloud.eu' \
  'https://gateway.isolarcloud.com.hk' \
  'https://gateway.isolarcloud.com.cn' \
  'https://gateway.isolarcloud.in'; do
  assert_equal "$supported_host" \
    "$(gosungrow_resolve_host "  $supported_host  " "$default_host")" \
    "supported host $supported_host is trimmed and selected"
done

if gosungrow_resolve_host 'https://unsupported.example' "$default_host" >/dev/null; then
  printf '%s\n' 'unsupported host was accepted' >&2
  exit 1
fi

# Run the actual wrapper with synthetic config and service responses. Bashio's
# fatal logger returns success, so the wrapper must explicitly stop startup.
startup_dir="$(mktemp -d)"
trap 'rm -rf "$startup_dir"' EXIT
sed "s|/usr/local/lib/gosungrow/|$script_dir/|g" "$script_dir/run.sh" > "$startup_dir/run.sh"

bashio::config() {
  case "$1" in
    gosungrow_user) [ "$startup_scenario" = missing_user ] || printf 'synthetic-user' ;;
    gosungrow_password) [ "$startup_scenario" = missing_password ] || printf 'synthetic-password' ;;
    mqtt_host) [ "$startup_scenario" != custom ] || printf 'custom-broker' ;;
    *) return 0 ;;
  esac
  return 0
}
bashio::services() {
  if [ "$startup_scenario" = missing_broker ] || [ "$startup_scenario" = custom ]; then
    printf '%s\n' 'Synthetic Supervisor service lookup failure' >&2
    return 1
  fi
  case "$2" in
    host) printf 'supervisor-broker' ;;
    port) printf '1883' ;;
    username) printf 'synthetic-mqtt-user' ;;
    password) printf 'synthetic-mqtt-password' ;;
  esac
}
bashio::config.true() {
  case "$startup_scenario:$1" in
    enabled:install_dashboard|enabled:dashboard_force_update|force_only:dashboard_force_update) return 0 ;;
    *) return 1 ;;
  esac
}
bashio::log.fatal() { printf '%s\n' "$*"; return 0; }
bashio::log.info() { printf '%s\n' "$*"; }
GoSungrow() { printf '%s\n' "$*" >> "$startup_calls"; }
export -f bashio::config bashio::services bashio::config.true bashio::log.fatal bashio::log.info GoSungrow

for startup_scenario in missing_user missing_password missing_broker supervisor custom enabled force_only; do
  startup_calls="$startup_dir/$startup_scenario.calls"
  export startup_scenario startup_calls
  startup_rc=0
  GOSUNGROW_CONFIG="$startup_dir/config/config.json" bash "$startup_dir/run.sh" > "$startup_dir/$startup_scenario.log" 2>&1 || startup_rc=$?
  case "$startup_scenario" in
    missing_user|missing_password|missing_broker)
      assert_equal 1 "$startup_rc" "$startup_scenario stops startup"
      if [ -e "$startup_calls" ]; then
        printf '%s\n' "$startup_scenario invoked GoSungrow before validating configuration" >&2
        exit 1
      fi
      case "$startup_scenario" in
        missing_user) expected_message='Missing required option: gosungrow_user' ;;
        missing_password) expected_message='Missing required option: gosungrow_password' ;;
        missing_broker) expected_message='No MQTT broker configured.' ;;
      esac
      if ! grep -Fq "$expected_message" "$startup_dir/$startup_scenario.log"; then
        printf '%s\n' "$startup_scenario omitted its configuration diagnostic" >&2
        exit 1
      fi
      ;;
    supervisor|custom|enabled|force_only)
      assert_equal 0 "$startup_rc" "$startup_scenario starts successfully"
      expected_broker=supervisor-broker
      if [ "$startup_scenario" = custom ]; then expected_broker=custom-broker; fi
      if ! grep -Fq -- "mqtt run --mqtt-host=$expected_broker --mqtt-port=1883" "$startup_calls"; then
        printf '%s\n' "$startup_scenario did not pass its resolved MQTT broker to the CLI" >&2
        exit 1
      fi
      expected_enabled=false
      expected_force=false
      if [ "$startup_scenario" = enabled ]; then expected_enabled=true; fi
      if [ "$startup_scenario" = enabled ] || [ "$startup_scenario" = force_only ]; then expected_force=true; fi
      for command in 'config write' 'mqtt run'; do
        if ! grep -F "$command " "$startup_calls" | grep -Fq -- "--dashboard-enabled=$expected_enabled --dashboard-force-update=$expected_force"; then
          printf '%s\n' "$startup_scenario omitted explicit dashboard settings from $command" >&2
          exit 1
        fi
      done
      if [ "$expected_enabled" = false ] && grep -Fq 'ha install-dashboard' "$startup_calls"; then
        printf '%s\n' "$startup_scenario installed a dashboard while disabled" >&2
        exit 1
      fi
      if [ "$expected_enabled" = true ] && ! grep -Fq 'ha install-dashboard' "$startup_calls"; then
        printf '%s\n' "$startup_scenario omitted the initial install before MQTT" >&2
        exit 1
      fi
      ;;
  esac
done

printf '%s\n' 'Configuration policy and wrapper startup tests passed.'
