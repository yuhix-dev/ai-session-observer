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

parse_pane_pid() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --pane-pid)
        shift
        printf '%s\n' "${1:-}"
        return 0
        ;;
    esac
    shift
  done
}

# Print PID and all descendant PIDs recursively
get_descendant_pids() {
  local pid="$1"
  kill -0 "$pid" 2>/dev/null || return 0
  printf '%s\n' "$pid"
  local children
  children=$(pgrep -P "$pid" 2>/dev/null) || return 0
  for child in $children; do
    get_descendant_pids "$child"
  done
}

# Find rollout files currently open by the process tree of pane_pid
pane_codex_rollout_files() {
  local pane_pid="$1"
  local pid_list
  pid_list=$(get_descendant_pids "$pane_pid" | tr '\n' ',' | sed 's/,$//')
  [ -z "$pid_list" ] && return 0
  lsof -p "$pid_list" 2>/dev/null \
    | awk '$NF ~ /rollout-.*\.jsonl$/ { print $NF }' \
    | sort -u
}

# Find Claude session IDs for the process tree of pane_pid
# using ~/.claude/sessions/{pid}.json written by Claude Code
pane_claude_session_ids() {
  local pane_pid="$1"
  get_descendant_pids "$pane_pid" | while IFS= read -r pid; do
    local sfile="${HOME}/.claude/sessions/${pid}.json"
    [ -f "$sfile" ] || continue
    jq -r '.sessionId // empty' "$sfile" 2>/dev/null
  done | sort -u
}

latest_claude_project_log() {
  find "${HOME}/.claude/projects" -type f -name '*.jsonl' 2>/dev/null | sort | tail -n 1
}

pane_claude_session_files() {
  local pane_pid="$1"
  pane_claude_session_ids "$pane_pid" | while IFS= read -r sid; do
    [ -n "$sid" ] || continue
    find "${HOME}/.claude/projects" -type f -name "${sid}.jsonl" 2>/dev/null
  done | sort -u
}
