#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
if [ -n "${HA_E2E_FIXTURE_PARENT:-${RUNNER_TEMP:-}}" ]; then
  fixture_parent="${HA_E2E_FIXTURE_PARENT:-$RUNNER_TEMP}"
  mkdir -p "$fixture_parent"
  fixture="$(mktemp -d "$fixture_parent/gosungrow-ha-bootstrap.XXXXXX")"
else
  fixture="$(mktemp -d)"
fi
container="gosungrow-ha-bootstrap-${RANDOM}-${RANDOM}"
host_port="${HA_E2E_BOOTSTRAP_PORT:-8124}"
core_image="${HA_CORE_IMAGE:-ghcr.io/home-assistant/home-assistant:2026.9.3}"
base_url="http://127.0.0.1:${host_port}"

wait_for_core() {
  local attempt
  for attempt in $(seq 1 90); do
    if curl --fail --silent --show-error "$base_url/api/onboarding" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  printf 'Home Assistant did not become ready within 180 seconds\n' >&2
  return 1
}

cleanup() {
  docker logs "$container" 2>/dev/null || true
  docker rm -f "$container" >/dev/null 2>&1 || true
  rm -rf "$fixture"
}
trap cleanup EXIT

cp "$repo_root/tools/ha-e2e/configuration.yaml" "$fixture/configuration.yaml"
asset="$repo_root/addon/gosungrow/assets/gosungrow-energy-flow-card-v2.js"
asset_hash="$(sha256sum "$asset" | awk '{print $1}')"
asset_name="gosungrow-dashboard-cards.${asset_hash:0:12}.js"
resource_url="/local/gosungrow/$asset_name"

docker run -d --name "$container" \
  -p "${host_port}:8123" \
  -v "$fixture:/config" \
  "$core_image"

wait_for_core
HA_E2E_URL="$base_url" \
HA_E2E_RESOURCE_URL="$resource_url" \
HA_E2E_BOOTSTRAP_PHASE=unavailable \
  node "$repo_root/tools/ha-e2e/local_route_bootstrap.mjs"

mkdir -p "$fixture/www/gosungrow"
cp "$asset" "$fixture/www/gosungrow/$asset_name"
HA_E2E_URL="$base_url" \
HA_E2E_RESOURCE_URL="$resource_url" \
HA_E2E_BOOTSTRAP_PHASE=unavailable \
  node "$repo_root/tools/ha-e2e/local_route_bootstrap.mjs"

docker restart "$container" >/dev/null
wait_for_core
HA_E2E_URL="$base_url" \
HA_E2E_RESOURCE_URL="$resource_url" \
HA_E2E_BOOTSTRAP_PHASE=available \
  node "$repo_root/tools/ha-e2e/local_route_bootstrap.mjs"

HA_E2E_URL="$base_url" \
HA_E2E_RESOURCE_URL="$resource_url" \
HA_E2E_RESOURCE_HASH="$asset_hash" \
  node "$repo_root/tools/ha-e2e/dashboard_resource_smoke.mjs"
