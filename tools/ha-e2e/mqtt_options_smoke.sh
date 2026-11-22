#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."
export MSYS_NO_PATHCONV=1

core_image=${HA_CORE_IMAGE:-ghcr.io/home-assistant/home-assistant:2026.10.4}
broker_image=eclipse-mosquitto:2.0.22

run_case() (
  local scenario=$1
  local fixture="gosungrow-options-${scenario}-${RANDOM}-${RANDOM}"
  local core="${fixture}-core" broker="${fixture}-broker"

  cleanup() {
    local status=$?
    if [ "$status" -ne 0 ]; then
      docker logs "$core" >&2 || true
      docker logs "$broker" >&2 || true
    fi
    docker rm -fv "$core" "$broker" >/dev/null 2>&1 || true
    docker network rm "$fixture" >/dev/null 2>&1 || true
    return "$status"
  }
  trap cleanup EXIT

  docker network create "$fixture" >/dev/null
  docker run -d --name "$broker" --network "$fixture" --network-alias mqtt \
    -p 127.0.0.1::1883 --entrypoint /bin/sh "$broker_image" \
    -c 'printf "listener 1883\nallow_anonymous true\n" >/tmp/smoke.conf; exec mosquitto -c /tmp/smoke.conf' >/dev/null
  docker run -d --name "$core" --network "$fixture" -p 127.0.0.1::8123 "$core_image" >/dev/null

  HA_E2E_MQTT_CORE_URL="http://$(docker port "$core" 8123/tcp)" \
  HA_E2E_MQTT_BROKER_ADDR="$(docker port "$broker" 1883/tcp)" \
  HA_E2E_MQTT_CASE="$scenario" \
    go test ./cmd -run '^TestRuntimeSelectHomeAssistant$' -count=1 -v -timeout 5m
)

for scenario in fresh retained registry; do
  run_case "$scenario"
done
