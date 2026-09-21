#!/usr/bin/env bash

gosungrow_failure_class_from_log() {
  local log_file
  local failure_class

  log_file="$1"
  failure_class="$(sed -n 's/^GoSungrow-Failure-Class: //p' "$log_file" | head -n 1)"
  case "$failure_class" in
    recoverable_remote|docker_dns|non_recoverable)
      printf '%s' "$failure_class"
      ;;
    *)
      printf '%s' 'unclassified'
      ;;
  esac
}

gosungrow_retry_policy() {
  case "$1" in
    recoverable_remote)
      printf '%s' 'recoverable_remote'
      ;;
    docker_dns)
      printf '%s' 'docker_dns'
      ;;
    *)
      printf '%s' 'stop'
      ;;
  esac
}
