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
sed \
  -e 's|/usr/local/lib/gosungrow/configuration.sh|"$startup_dir/configuration.sh"|' \
  -e "s|/usr/local/lib/gosungrow/recovery_policy.sh|$script_dir/recovery_policy.sh|" \
  -e 's|gosungrow_read_app_options /data/options.json|gosungrow_read_app_options "$startup_case/options.json"|' \
  "$script_dir/run.sh" > "$startup_dir/run.sh"
sed 's|/proc/uptime|"$startup_case/uptime"|' "$script_dir/configuration.sh" > "$startup_dir/configuration.sh"
export startup_dir

advance_startup_clock() {
  local seconds="$1" current elapsed
  read -r current < "$startup_case/uptime"
  elapsed=$((10#${current%.*} * 100 + 10#${current#*.} + 10#${seconds%.*} * 100 + 10#${seconds#*.}))
  printf '%d.%02d\n' "$((elapsed / 100))" "$((elapsed % 100))" > "$startup_case/uptime"
}
curl() {
  local header output limit status body request url
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --header) header="$2"; shift ;;
      --output) output="$2"; shift ;;
      --max-time) limit="$2"; shift ;;
      --noproxy|--write-out) shift ;;
      http://*) url="$1" ;;
    esac
    shift
  done
  printf '%s\n' "$limit" >> "$startup_case/requests"
  request="$(wc -l < "$startup_case/requests")"
  if [ "$header" != @- ] || [ "$(cat)" != "Authorization: Bearer sentinel-supervisor-token" ] ||
    [ "$url" != http://supervisor/services/mqtt ]; then
    return 2
  fi

  case "$startup_scenario:$request" in
    missing_broker:*|delayed:1|cancel_wait:*)
      status=400
      body='{"result":"error","message":"Service not enabled"}'
      ;;
    delayed:2) return 7 ;;
    delayed:3) status=429; body='{"result":"error"}' ;;
    delayed:4) status=503; body='{"result":"error"}' ;;
    delayed:5|hanging:*)
      advance_startup_clock "$limit"
      return 28
      ;;
    delayed:*)
      status=200
      body='{"result":"ok","data":{"host":"supervisor-broker","port":2883,"username":"sentinel-mqtt-user","password":"sentinel mqtt-secret & $! *"}}'
      ;;
    unauthorized:*) status=401; body='{"result":"error","message":"sentinel-raw-response"}' ;;
    forbidden:*) status=403; body='{"result":"error","message":"sentinel-raw-response"}' ;;
    rejected:*) status=400; body='{"result":"error","message":"Service not enabled for sentinel-raw-response"}' ;;
    invalid_response:*) status=200; body='{"result":"ok","data":{"port":1883,"password":"sentinel-raw-response"}}' ;;
    invalid_port:*) status=200; body='{"result":"ok","data":{"host":"supervisor-broker","port":65536}}' ;;
    cancel_request:*)
      printf '%s\n' "$BASHPID" > "$startup_case/active"
      exec sleep 60
      ;;
    *)
      status=200
      body='{"result":"ok","data":{"host":"supervisor-broker"}}'
      ;;
  esac
  printf '%s\n' "$body" > "$output"
  printf '%s\n' "$status"
  printf '%s\n' 'sentinel-raw-stderr' >&2
}
sleep() {
  printf '%s\n' "$1" >> "$startup_case/waits"
  if [ "$startup_scenario" = cancel_wait ]; then
    printf '%s\n' "$BASHPID" > "$startup_case/active"
    exec sleep 60
  fi
  advance_startup_clock "$1"
}
bashio::log.fatal() { printf '%s\n' "$*"; return 0; }
bashio::log.info() { printf '%s\n' "$*"; }
bashio::log.warning() { printf '%s\n' "$*"; }
GoSungrow() {
  printf '%s\n' "$*" >> "$startup_case/commands"
  case "$startup_scenario:$1 $2" in
    'dashboard_failure:ha install-dashboard'|'cloud_denial:mqtt run')
      printf '%s\n' 'iSolarCloud HTTP denial: status=403' 'GoSungrow-Failure-Class: non_recoverable'
      return 1
      ;;
  esac
}
export -f bashio::log.fatal bashio::log.info bashio::log.warning
export -f GoSungrow curl sleep advance_startup_clock

prepare_startup() {
  export startup_scenario="$1"
  export startup_case="$startup_dir/$startup_scenario"
  mkdir -p "$startup_case"
  printf '100.00\n' > "$startup_case/uptime"
  touch "$startup_case/requests" "$startup_case/waits"

  local user=synthetic-user password=synthetic-password gateway="" broker="" broker_user="" broker_password=""
  local port=1883 dashboard=false force=false
  case "$startup_scenario" in
    missing_options) return ;;
    invalid_options)
      printf '%s\n' '{"gosungrow_password":"sentinel-raw-options",' > "$startup_case/options.json"
      return
      ;;
    missing_user) user="" ;;
    missing_password) password="" ;;
    invalid_gateway) gateway=https://unsupported.example ;;
    custom|custom_offline)
      broker=custom-broker
      broker_user=custom-user
      broker_password=custom-secret
      if [ "$startup_scenario" = custom_offline ]; then port=2884; fi
      ;;
    enabled) dashboard=true; force=true ;;
    dashboard_failure) dashboard=true ;;
    force_only) force=true ;;
  esac
  jq -n --arg user "$user" --arg password "$password" --arg gateway "$gateway" \
    --arg broker "$broker" --arg broker_user "$broker_user" --arg broker_password "$broker_password" \
    --argjson port "$port" --argjson dashboard "$dashboard" --argjson force "$force" '
      {gosungrow_user: $user, gosungrow_password: $password, gosungrow_host: $gateway,
       mqtt_host: $broker, mqtt_port: $port, mqtt_username: $broker_user, mqtt_password: $broker_password,
       debug: true, install_dashboard: $dashboard, dashboard_force_update: $force}
    ' > "$startup_case/options.json"
}

assert_log() {
  if ! grep -Fq -- "$1" "$startup_case/log"; then
    printf '%s omitted the diagnostic %s\n' "$startup_scenario" "$1" >&2
    cat "$startup_case/log" >&2
    exit 1
  fi
}

for scenario in missing_options invalid_options missing_user missing_password invalid_gateway missing_token missing_broker \
  hanging unauthorized forbidden rejected invalid_response invalid_port supervisor delayed custom custom_offline enabled force_only dashboard_failure; do
  prepare_startup "$scenario"
  startup_calls="$startup_case/commands"
  token=sentinel-supervisor-token
  if [ "$scenario" = missing_token ] || [ "$scenario" = custom_offline ]; then token=""; fi
  startup_rc=0
  SUPERVISOR_TOKEN="$token" GOSUNGROW_CONFIG="$startup_case/config/config.json" \
    bash "$startup_dir/run.sh" > "$startup_case/log" 2>&1 || startup_rc=$?
  if grep -Eq 'sentinel-|mqtt-secret|custom-secret' "$startup_case/log"; then
    printf '%s leaked a credential or raw response\n' "$startup_scenario" >&2
    exit 1
  fi
  requests="$(wc -l < "$startup_case/requests" | tr -d '[:space:]')"
  waits="$(paste -sd ' ' "$startup_case/waits")"
  case "$startup_scenario" in
    missing_options|invalid_options|missing_user|missing_password|invalid_gateway|missing_token|missing_broker|hanging|unauthorized|forbidden|rejected|invalid_response|invalid_port)
      assert_equal 1 "$startup_rc" "$startup_scenario stops startup"
      if [ -e "$startup_calls" ] || [ -e "$startup_case/config" ]; then
        printf '%s started runtime work before resolving configuration\n' "$startup_scenario" >&2
        exit 1
      fi
      expected_requests=1
      expected_waits=""
      case "$startup_scenario" in
        missing_options|invalid_options) expected_requests=0; assert_log 'Could not read valid app options' ;;
        missing_user) expected_requests=0; assert_log 'Missing required option: gosungrow_user' ;;
        missing_password) expected_requests=0; assert_log 'Missing required option: gosungrow_password' ;;
        invalid_gateway) expected_requests=0; assert_log 'Unsupported gosungrow_host' ;;
        missing_token) expected_requests=0; assert_log 'access-denied' ;;
        unauthorized|forbidden) assert_log 'first failure: access-denied'; assert_log 'Check GoSungrow access' ;;
        rejected) assert_log 'first failure: http-rejected' ;;
        invalid_response|invalid_port) assert_log 'first failure: invalid-response'; assert_log 'Check the MQTT settings' ;;
        missing_broker)
          expected_requests=11
          expected_waits='2.00 4.00 8.00 15.00 15.00 15.00 15.00 15.00 15.00 15.00 1.00'
          assert_equal '1.00' "$(tail -n 1 "$startup_case/requests")" 'last request uses only the remaining budget'
          assert_log 'deadline-exhausted, first failure: service-unavailable'
          assert_log 'Restart GoSungrow after registration succeeds'
          ;;
        hanging)
          expected_requests=8
          expected_waits='2.00 4.00 8.00 15.00 15.00 15.00 15.00 6.00'
          if grep -Fvxq '5.00' "$startup_case/requests"; then
            printf '%s\n' 'A hanging request was given an incorrect timeout' >&2
            exit 1
          fi
          assert_log 'deadline-exhausted, first failure: request-timeout'
          ;;
      esac
      if [ "$startup_scenario" = missing_broker ] || [ "$startup_scenario" = hanging ]; then
        assert_equal '220.00' "$(cat "$startup_case/uptime")" "$startup_scenario consumes only the total budget"
      fi
      ;;
    supervisor|delayed|custom|custom_offline|enabled|force_only|dashboard_failure)
      assert_equal 0 "$startup_rc" "$startup_scenario starts successfully"
      expected_broker=supervisor-broker
      expected_port=1883
      expected_auth='--mqtt-user= --mqtt-password='
      expected_requests=1
      expected_waits=""
      case "$startup_scenario" in
        custom|custom_offline)
          expected_broker=custom-broker
          if [ "$startup_scenario" = custom_offline ]; then expected_port=2884; fi
          expected_auth='--mqtt-user=custom-user --mqtt-password=custom-secret'
          expected_requests=0
          ;;
        delayed)
          expected_port=2883
          expected_auth='--mqtt-user=sentinel-mqtt-user --mqtt-password=sentinel mqtt-secret & $! *'
          expected_requests=6
          expected_waits='2.00 4.00 8.00 15.00 15.00'
          assert_log 'attempt 1 after 0.00s: service-unavailable. Retrying in 2.00s.'
          assert_log 'attempt 6 after 49.00s: ready.'
          ;;
      esac
      if ! grep -Fq -- "mqtt run --mqtt-host=$expected_broker --mqtt-port=$expected_port $expected_auth" "$startup_calls"; then
        printf '%s\n' "$startup_scenario did not pass its resolved MQTT broker to the CLI" >&2
        exit 1
      fi
      if grep -Fq 'api login' "$startup_calls"; then
        printf '%s triggered login during MQTT discovery\n' "$startup_scenario" >&2
        exit 1
      fi
      expected_enabled=false
      expected_force=false
      if [ "$startup_scenario" = enabled ] || [ "$startup_scenario" = dashboard_failure ]; then expected_enabled=true; fi
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
      if [ "$startup_scenario" = dashboard_failure ]; then
        assert_log 'iSolarCloud HTTP denial: status=403'
        assert_log 'Managed dashboard Installing failed'
      fi
      if [ "$expected_enabled" = true ] && ! grep -Fq 'ha install-dashboard' "$startup_calls"; then
        printf '%s\n' "$startup_scenario omitted the initial install before MQTT" >&2
        exit 1
      fi
      ;;
  esac
  assert_equal "$expected_requests" "$requests" "$startup_scenario Supervisor requests"
  assert_equal "$expected_waits" "$waits" "$startup_scenario retry waits"
done

for scenario in cancel_request cancel_wait; do
  prepare_startup "$scenario"
  SUPERVISOR_TOKEN=sentinel-supervisor-token GOSUNGROW_CONFIG="$startup_case/config/config.json" \
    bash "$startup_dir/run.sh" > "$startup_case/log" 2>&1 &
  wrapper_pid=$!
  for ((poll = 0; poll < 200; poll++)); do
    if [ -s "$startup_case/active" ]; then break; fi
    command sleep 0.01
  done
  if ! read -r child_pid < "$startup_case/active"; then
    kill "$wrapper_pid" 2>/dev/null || true
    wait "$wrapper_pid" || true
    printf '%s never reached its blocking operation\n' "$startup_scenario" >&2
    exit 1
  fi
  kill -TERM "$wrapper_pid"
  startup_rc=0
  wait "$wrapper_pid" || startup_rc=$?
  if [ "$startup_rc" -eq 0 ] || kill -0 "$child_pid" 2>/dev/null || [ -e "$startup_case/commands" ]; then
    printf '%s did not cancel the active operation before runtime startup\n' "$startup_scenario" >&2
    exit 1
  fi
  assert_log 'discovery stopped (shutdown'
done

prepare_startup cloud_denial
startup_rc=0
SUPERVISOR_TOKEN="sentinel-supervisor-token" GOSUNGROW_CONFIG="$startup_case/config/config.json" \
  bash "$startup_dir/run.sh" > "$startup_case/log" 2>&1 || startup_rc=$?
assert_equal 1 "$startup_rc" 'iSolarCloud HTTP 403 propagates the MQTT exit code'
assert_log 'iSolarCloud HTTP denial: status=403'
assert_equal 1 "$(grep -c '^mqtt run ' "$startup_case/commands")" 'iSolarCloud HTTP 403 stops after one MQTT invocation'
if grep -Fq 'api login' "$startup_case/commands"; then
  printf '%s\n' 'iSolarCloud HTTP 403 triggered login' >&2
  exit 1
fi

printf '%s\n' 'Configuration policy and wrapper startup tests passed.'
