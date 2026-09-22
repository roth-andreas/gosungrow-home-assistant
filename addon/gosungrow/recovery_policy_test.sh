#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$script_dir/recovery_policy.sh"

test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT

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

remote_log="$test_dir/remote.log"
printf '%s\n' \
  'GoSungrow-Failure-Class: recoverable_remote' \
  'ERROR: terminal stop reason includes lookup gateway on 127.0.0.11:53: no such host' \
  'GoSungrow-Failure-Class: docker_dns' >"$remote_log"
remote_class="$(gosungrow_failure_class_from_log "$remote_log")"
assert_equal 'recoverable_remote' "$remote_class" 'stable remote classification'
assert_equal 'recoverable_remote' "$(gosungrow_retry_policy "$remote_class")" 'remote retry policy'

dns_log="$test_dir/dns.log"
printf '%s\n' \
  'GoSungrow-Failure-Class: docker_dns' \
  'ERROR: name resolution failed' >"$dns_log"
dns_class="$(gosungrow_failure_class_from_log "$dns_log")"
assert_equal 'docker_dns' "$dns_class" 'stable Docker DNS classification'
assert_equal 'docker_dns' "$(gosungrow_retry_policy "$dns_class")" 'Docker DNS retry policy'

unknown_log="$test_dir/unknown.log"
printf '%s\n' 'GoSungrow-Failure-Class: future_class' >"$unknown_log"
assert_equal 'unclassified' "$(gosungrow_failure_class_from_log "$unknown_log")" 'unknown classification'
assert_equal 'stop' "$(gosungrow_retry_policy unclassified)" 'unclassified policy'

missing_log="$test_dir/missing.log"
printf '%s\n' 'ERROR: API httpResponse is 500 Internal Server Error' >"$missing_log"
assert_equal 'unclassified' "$(gosungrow_failure_class_from_log "$missing_log")" 'missing classification'
assert_equal 'stop' "$(gosungrow_retry_policy non_recoverable)" 'non-recoverable policy'

action_log="$test_dir/action.log"
printf '%s\n' 'GoSungrow-Failure-Class: operator_action_required' >"$action_log"
action_class="$(gosungrow_failure_class_from_log "$action_log")"
assert_equal 'operator_action_required' "$action_class" 'operator-action classification'
assert_equal 'operator_action_required' "$(gosungrow_retry_policy "$action_class")" 'operator-action policy'

printf '%s\n' 'Recovery policy tests passed.'
