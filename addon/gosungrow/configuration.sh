#!/usr/bin/env bash

gosungrow_trim() {
  local value
  value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s' "$value"
}

gosungrow_resolve_host() {
  local configured_host
  local fallback_host

  configured_host="$(gosungrow_trim "${1:-}")"
  fallback_host="$2"
  if [ -z "$configured_host" ]; then
    printf '%s' "$fallback_host"
    return 0
  fi

  case "$configured_host" in
    https://augateway.isolarcloud.com | \
      https://gateway.isolarcloud.com | \
      https://gateway.isolarcloud.eu | \
      https://gateway.isolarcloud.com.hk | \
      https://gateway.isolarcloud.com.cn | \
      https://gateway.isolarcloud.in)
      printf '%s' "$configured_host"
      ;;
    *)
      return 1
      ;;
  esac
}

gosungrow_read_app_options() {
  local -a options=()

  # Supervisor writes this validated snapshot before launching the app.
  mapfile -d '' -t options < <(jq -js '
    def option($key; $kind; $default):
      (if .[$key] == null then $default else .[$key] end)
      | select(type == $kind) | tostring | select(index("\u0000") == null);
    def optional_string($key):
      option($key; "string"; "") | if . == "null" or . == "~" then "" else . end;

    select(length == 1 and (.[0] | type == "object")) | .[0]
    | [
      option("gosungrow_user"; "string"; ""),
      option("gosungrow_password"; "string"; ""),
      optional_string("gosungrow_host"),
      optional_string("mqtt_host"),
      option("mqtt_port"; "number"; 1883),
      optional_string("mqtt_username"),
      optional_string("mqtt_password"),
      option("debug"; "boolean"; false),
      option("dashboard_language"; "string"; "auto"),
      option("install_dashboard"; "boolean"; false),
      option("dashboard_force_update"; "boolean"; false)
    ]
    | select(length == 11) | .[] | ., "\u0000"
  ' "$1" 2>/dev/null)
  if [ "${#options[@]}" -ne 11 ] || ! [[ "${options[4]}" =~ ^[0-9]{1,5}$ ]] ||
    (( 10#${options[4]} < 1 || 10#${options[4]} > 65535 )); then
    return 1
  fi

  GOSUNGROW_USER="${options[0]}"
  GOSUNGROW_PASSWORD="${options[1]}"
  GOSUNGROW_HOST="${options[2]}"
  GOSUNGROW_MQTT_HOST="${options[3]}"
  GOSUNGROW_MQTT_PORT="${options[4]}"
  GOSUNGROW_MQTT_USER="${options[5]}"
  GOSUNGROW_MQTT_PASSWORD="${options[6]}"
  GOSUNGROW_DEBUG="${options[7]}"
  GOSUNGROW_DASHBOARD_LANGUAGE="${options[8]}"
  GOSUNGROW_DASHBOARD_ENABLED="${options[9]}"
  GOSUNGROW_DASHBOARD_FORCE_UPDATE="${options[10]}"
}

gosungrow_uptime_centiseconds() {
  local uptime
  read -r uptime _ < /proc/uptime || return 1
  printf '%s\n' "$((10#${uptime%.*} * 100 + 10#${uptime#*.}))"
}

gosungrow_supervisor_mqtt_outcome() {
  local curl_status="$1" http_status="$2" body="$3"

  case "$curl_status" in
    0) ;;
    28) printf 'request-timeout'; return ;;
    5|6|7|8|18|52|55|56) printf 'transport-error'; return ;;
    *) printf 'request-failed'; return ;;
  esac

  case "$http_status" in
    2[0-9][0-9]) printf 'response-received' ;;
    401|403) printf 'access-denied' ;;
    429|5[0-9][0-9]) printf 'http-%s' "$http_status" ;;
    400)
      if jq -es 'length == 1 and (.[0] | .result == "error" and .message == "Service not enabled")' "$body" >/dev/null 2>&1; then
        printf 'service-unavailable'
      else
        printf 'http-rejected'
      fi
      ;;
    *) printf 'http-rejected' ;;
  esac
}

gosungrow_decode_supervisor_mqtt() {
  # NUL delimiters preserve whitespace and empty credentials without evaluating shell text.
  jq -jse '
    def credential:
      type == "string" and index("\u0000") == null;
    def port:
      type == "number" and . >= 1 and . <= 65535 and . == floor;

    select(length == 1) | .[0] | select(.result == "ok") | .data
    | select(
        type == "object"
        and (.host | credential and test("^[^[:space:][:cntrl:]]+$"))
        and (if has("port") then .port | port else true end)
        and (if has("username") then .username | credential else true end)
        and (if has("password") then .password | credential else true end)
      )
    | .host, "\u0000", ((.port // 1883) | tostring), "\u0000",
      (.username // ""), "\u0000", (.password // ""), "\u0000"
  ' "$1" 2>/dev/null
}

gosungrow_discover_supervisor_mqtt() {
  local started now remaining limit retry_wait timeout delay elapsed response_dir
  local active_pid="" first_failure=none outcome="" stop_reason=deadline-exhausted
  local attempt=0 curl_status http_status guidance result=1
  local -a delays=(200 400 800 1500) fields=()

  if [ -z "${SUPERVISOR_TOKEN:-}" ]; then
    bashio::log.fatal 'Supervisor MQTT discovery stopped (access-denied). Check GoSungrow access to the Supervisor API.'
    return 1
  fi
  if ! started="$(gosungrow_uptime_centiseconds)" || ! response_dir="$(mktemp -d)"; then
    bashio::log.fatal 'Supervisor MQTT discovery stopped. Could not initialize the startup deadline or response storage.'
    return 1
  fi

  # wait is interrupted by signals even while curl or sleep is running.
  trap '
    if [ -n "$active_pid" ]; then
      kill "$active_pid" 2>/dev/null || true
      wait "$active_pid" 2>/dev/null || true
    fi
    rm -f -- "$response_dir/body" "$response_dir/status" "$response_dir/fields"
    rmdir -- "$response_dir"
    bashio::log.info "Supervisor MQTT discovery stopped (shutdown, first failure: $first_failure)."
    exit 1
  ' INT TERM

  while true; do
    now="$(gosungrow_uptime_centiseconds)"
    remaining=$((12000 - (now - started)))
    if [ "$remaining" -le 0 ]; then
      break
    fi
    limit=$((remaining < 500 ? remaining : 500))
    printf -v timeout '%d.%02d' "$((limit / 100))" "$((limit % 100))"
    attempt=$((attempt + 1))

    # Keep the bearer token out of argv and response data out of Bashio logs.
    curl --disable --silent --noproxy '*' --max-time "$timeout" \
      --header @- --output "$response_dir/body" --write-out '%{http_code}\n' \
      http://supervisor/services/mqtt <<< "Authorization: Bearer $SUPERVISOR_TOKEN" \
      > "$response_dir/status" 2>/dev/null &
    active_pid=$!
    curl_status=0
    wait "$active_pid" || curl_status=$?
    active_pid=""
    http_status=""
    read -r http_status < "$response_dir/status" || true
    outcome="$(gosungrow_supervisor_mqtt_outcome "$curl_status" "$http_status" "$response_dir/body")"
    if [ "$outcome" = response-received ]; then
      if gosungrow_decode_supervisor_mqtt "$response_dir/body" > "$response_dir/fields"; then
        mapfile -d '' -t fields < "$response_dir/fields"
        outcome=ready
      else
        outcome=invalid-response
      fi
    fi

    now="$(gosungrow_uptime_centiseconds)"
    elapsed=$((now - started))
    remaining=$((12000 - elapsed))
    printf -v elapsed '%d.%02d' "$((elapsed / 100))" "$((elapsed % 100))"
    if [ "$outcome" = ready ] && [ "$remaining" -gt 0 ]; then
      GOSUNGROW_MQTT_HOST="${fields[0]}"
      GOSUNGROW_MQTT_PORT="${fields[1]}"
      GOSUNGROW_MQTT_USER="${fields[2]}"
      GOSUNGROW_MQTT_PASSWORD="${fields[3]}"
      bashio::log.info "Supervisor MQTT discovery attempt $attempt after ${elapsed}s: ready."
      result=0
      break
    fi

    if [ "$first_failure" = none ] && [ "$outcome" != ready ]; then
      first_failure="$outcome"
    fi
    case "$outcome" in
      service-unavailable|request-timeout|transport-error|http-429|http-5[0-9][0-9]|ready)
        if [ "$remaining" -le 0 ]; then
          bashio::log.warning "Supervisor MQTT discovery attempt $attempt after ${elapsed}s: $outcome."
          break
        fi
        retry_wait="${delays[attempt - 1]:-1500}"
        retry_wait=$((remaining < retry_wait ? remaining : retry_wait))
        printf -v delay '%d.%02d' "$((retry_wait / 100))" "$((retry_wait % 100))"
        bashio::log.warning "Supervisor MQTT discovery attempt $attempt after ${elapsed}s: $outcome. Retrying in ${delay}s."
        sleep "$delay" &
        active_pid=$!
        wait "$active_pid" || true
        active_pid=""
        ;;
      *)
        bashio::log.warning "Supervisor MQTT discovery attempt $attempt after ${elapsed}s: $outcome."
        stop_reason=permanent-failure
        break
        ;;
    esac
  done

  rm -f -- "$response_dir/body" "$response_dir/status" "$response_dir/fields"
  rmdir -- "$response_dir"
  trap - INT TERM

  if [ "$result" -ne 0 ]; then
    case "$outcome" in
      service-unavailable) guidance='Check that the MQTT provider registers with Supervisor. Restart GoSungrow after registration succeeds, or configure a custom broker with mqtt_host.' ;;
      access-denied) guidance='Check GoSungrow access to the Supervisor API.' ;;
      invalid-response) guidance='Check the MQTT settings registered by the provider in Supervisor.' ;;
      *) guidance='Check the Supervisor logs and MQTT service availability.' ;;
    esac
    bashio::log.fatal "Supervisor MQTT discovery stopped ($stop_reason, first failure: $first_failure, last outcome: ${outcome:-none}). $guidance"
  fi
  return "$result"
}
