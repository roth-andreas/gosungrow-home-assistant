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

printf '%s\n' 'Configuration policy tests passed.'
