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
