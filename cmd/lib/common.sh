#!/usr/bin/env bash

set -euo pipefail

project_root() {
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
  printf '%s\n' "$script_dir"
}

redact_stream() {
  perl -pe '
    s/(Bearer\s+)[A-Za-z0-9._-]+/${1}[REDACTED]/gi;
    s/\b(sk|rk|pk)_[A-Za-z0-9_-]+\b/[REDACTED]/g;
    s/\b(api[_-]?key|token|secret|password|credential)\b\s*[:=]\s*["'\''"]?[^,"'\''\s}]+/"$1:[REDACTED]"/gi;
  '
}

latest_codex_rollout() {
  find "${HOME}/.codex/sessions" -type f -name 'rollout-*.jsonl' 2>/dev/null | sort | tail -n 1
}

parse_mode() {
  local mode="summary"

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --summary)
        mode="summary"
        ;;
      --details)
        mode="details"
        ;;
      --raw)
        mode="raw"
        ;;
    esac
    shift
  done

  printf '%s\n' "$mode"
}

should_follow() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --once)
        printf '0\n'
        return 0
        ;;
    esac
    shift
  done

  printf '1\n'
}
