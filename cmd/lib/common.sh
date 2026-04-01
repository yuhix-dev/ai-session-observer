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

parse_pane_target() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --pane-target)
        shift
        printf '%s\n' "${1:-}"
        return 0
        ;;
    esac
    shift
  done
}

tmux_cmd() {
  env -u TMUX tmux "$@"
}

current_pane_pid() {
  local pane_pid="${1:-}"
  local pane_target="${2:-}"

  if [ -n "$pane_target" ]; then
    tmux_cmd display-message -p -t "$pane_target" '#{pane_pid}' 2>/dev/null || true
    return 0
  fi

  printf '%s\n' "$pane_pid"
}

# Print PID and all descendant PIDs recursively
get_descendant_pids() {
  local pid="$1"
  kill -0 "$pid" 2>/dev/null || return 0
  printf '%s\n' "$pid"
  local children
  # `pgrep -P` has proven unreliable for the tmux pane -> codex -> subagent chain
  # on macOS. Walk the process table directly so watcher discovery follows every
  # rollout file the active Codex tree keeps open.
  children=$(
    ps -ax -o pid=,ppid= 2>/dev/null \
      | awk -v parent="$pid" '$2 == parent { print $1 }'
  ) || return 0
  [ -n "$children" ] || return 0
  for child in $children; do
    get_descendant_pids "$child"
  done
}

# Find rollout files currently open by the process tree of pane_pid.
# `lsof` may return non-zero when part of the process tree exits mid-scan;
# treat that as a partial read instead of a hard failure.
pane_codex_rollout_files() {
  local pane_pid="$1"
  local pid_list
  pid_list=$(get_descendant_pids "$pane_pid" | tr '\n' ',' | sed 's/,$//')
  [ -z "$pid_list" ] && return 0
  {
    lsof -p "$pid_list" 2>/dev/null || true
  } \
    | awk '$NF ~ /rollout-.*\.jsonl$/ { print $NF }' \
    | sort -u
}

# Build a human-readable label for a Codex rollout file.
# Falls back to the basename when the JSONL does not expose enough metadata.
codex_rollout_label() {
  local file="$1"

  jq -Rrs '
    def entries:
      split("\n")
      | map(fromjson? | select(. != null));

    def clean:
      gsub("[\r\n]+"; " ")
      | gsub("[[:space:]]+"; " ")
      | sub("^ +"; "")
      | sub(" +$"; "");

    def clip($n):
      if length > $n then .[0:($n - 3)] + "..." else . end;

    def headline($text; $n):
      ($text | clean
       | if test("[。.!?]") then match("^[^。.!?]+[。.!?]?").string else . end
       | clip($n));

    def subagent_spawn($meta):
      if ($meta.source | type) == "object" then
        ($meta.source.subagent.thread_spawn // {})
      else
        {}
      end;

    def actor_nick($meta):
      $meta.agent_nickname // (subagent_spawn($meta).agent_nickname // "Codex");

    def actor_role($meta):
      $meta.agent_role // (subagent_spawn($meta).agent_role // "");

    def text_of($entry):
      if $entry.type == "event_msg" and $entry.payload.type == "task_complete" then
        ($entry.payload.last_agent_message // "" | clean)
      elif $entry.type == "event_msg" and $entry.payload.type == "agent_message" then
        ($entry.payload.message // "" | clean)
      elif $entry.type == "response_item"
         and $entry.payload.type == "message"
         and $entry.payload.role == "assistant"
      then
        ([
          $entry.payload.content[]?
          | select(.type == "text" or .type == "output_text")
          | (.text // "")
        ] | join(" ") | clean)
      else
        ""
      end;

    (entries) as $entries
    | ([ $entries[] | select(.type == "session_meta") | .payload ] | first // {}) as $meta
    | (if any($entries[]; .type == "event_msg" and .payload.type == "task_complete") then "Completed"
       elif any($entries[]; .type == "event_msg" and .payload.type == "turn_aborted") then "Aborted"
       elif any($entries[]; .type == "event_msg" and .payload.type == "task_started") then "Started"
       else "Active"
       end) as $status
    | ([ $entries[] | select(.type == "event_msg" and .payload.type == "task_complete") | text_of(.) ] | map(select(length > 0)) | last // "") as $completion_summary
    | ([ $entries[] | select(.type == "event_msg" and .payload.type == "agent_message") | text_of(.) ] | map(select(length > 0)) | last // "") as $commentary_summary
    | ([ $entries[] | select(.type == "response_item" and .payload.type == "message" and .payload.role == "assistant") | text_of(.) ] | map(select(length > 0)) | last // "") as $assistant_summary
    | (if $completion_summary != "" then $completion_summary
       elif $commentary_summary != "" then $commentary_summary
       else $assistant_summary
       end) as $raw_summary
    | (headline($raw_summary; 96)) as $summary
    | "\((actor_nick($meta)))\(if (actor_role($meta)) != "" then " [\((actor_role($meta)))]" else "" end): \($status)\(if $summary != "" then " - \($summary)" else "" end)"
  ' "$file" 2>/dev/null
}

claude_session_label() {
  local file="$1"
  local base
  base="$(basename "$file" .jsonl)"

  case "$file" in
    */subagents/agent-*.jsonl)
      printf 'subagent %s\n' "${base#agent-}"
      ;;
    *)
      printf 'session %s\n' "${base}"
      ;;
  esac
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

stop_background_jobs() {
  local pid
  while IFS= read -r pid; do
    [ -n "$pid" ] || continue
    kill "$pid" 2>/dev/null || true
  done < <(jobs -pr)
}
