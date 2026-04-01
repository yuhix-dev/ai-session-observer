#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import os
import re
import signal
import subprocess
import sys
import time
from collections.abc import Iterable
from pathlib import Path


HOME = Path.home()
POLL_INTERVAL = float(os.environ.get("AI_SESSION_POLL_INTERVAL", "0.1"))

TOKEN_PATTERNS = [
    (re.compile(r"(Bearer\s+)[A-Za-z0-9._-]+", re.IGNORECASE), r"\1[REDACTED]"),
    (re.compile(r"\b(?:sk|rk|pk)_[A-Za-z0-9_-]+\b"), "[REDACTED]"),
    (
        re.compile(
            r'\b(api[_-]?key|token|secret|password|credential)\b\s*[:=]\s*["\']?[^,"\'\s}]+',
            re.IGNORECASE,
        ),
        lambda match: f"{match.group(1)}:[REDACTED]",
    ),
]


def redact_text(text: str) -> str:
    redacted = text
    for pattern, replacement in TOKEN_PATTERNS:
        redacted = pattern.sub(replacement, redacted)
    return redacted


def clean(text: str) -> str:
    text = re.sub(r"[\r\n]+", " ", text)
    text = re.sub(r"\s+", " ", text)
    return text.strip()


def clip(text: str, limit: int) -> str:
    return text if len(text) <= limit else text[: limit - 3] + "..."


def headline(text: str, limit: int) -> str:
    normalized = clean(text)
    match = re.match(r"^[^。.!?]+[。.!?]?", normalized)
    if match:
        normalized = match.group(0)
    return clip(normalized, limit)


def run_command(args: list[str]) -> subprocess.CompletedProcess[str]:
    return subprocess.run(args, text=True, capture_output=True, check=False)


def tmux_display_pane_pid(pane_target: str) -> str:
    result = run_command(["env", "-u", "TMUX", "tmux", "display-message", "-p", "-t", pane_target, "#{pane_pid}"])
    return result.stdout.strip() if result.returncode == 0 else ""


def current_pane_pid(pane_pid: str | None, pane_target: str | None) -> str:
    if pane_target:
        return tmux_display_pane_pid(pane_target)
    return pane_pid or ""


def process_table() -> dict[int, list[int]]:
    result = run_command(["ps", "-ax", "-o", "pid=,ppid="])
    children: dict[int, list[int]] = {}
    for line in result.stdout.splitlines():
        parts = line.split()
        if len(parts) != 2:
            continue
        pid, ppid = (int(parts[0]), int(parts[1]))
        children.setdefault(ppid, []).append(pid)
    return children


def get_descendant_pids(root_pid: str | None) -> list[int]:
    if not root_pid:
        return []
    try:
        root = int(root_pid)
    except ValueError:
        return []
    try:
        os.kill(root, 0)
    except OSError:
        return []

    children = process_table()
    discovered: list[int] = []
    stack = [root]
    seen: set[int] = set()
    while stack:
        pid = stack.pop()
        if pid in seen:
            continue
        seen.add(pid)
        discovered.append(pid)
        stack.extend(children.get(pid, []))
    return discovered


def lsof_rollout_files(pids: Iterable[int]) -> list[Path]:
    pid_values = [str(pid) for pid in pids]
    if not pid_values:
        return []
    result = run_command(["lsof", "-p", ",".join(pid_values)])
    paths: set[Path] = set()
    for line in result.stdout.splitlines():
        parts = line.split()
        if not parts:
            continue
        maybe_path = parts[-1]
        if re.search(r"rollout-.*\.jsonl$", maybe_path):
            paths.add(Path(maybe_path))
    return sorted(paths)


def pane_codex_rollout_files(pane_pid: str | None) -> list[Path]:
    return lsof_rollout_files(get_descendant_pids(pane_pid))


def pane_claude_session_ids(pane_pid: str | None) -> list[str]:
    session_ids: set[str] = set()
    for pid in get_descendant_pids(pane_pid):
        session_file = HOME / ".claude" / "sessions" / f"{pid}.json"
        if not session_file.is_file():
            continue
        try:
            data = json.loads(session_file.read_text())
        except Exception:
            continue
        session_id = data.get("sessionId")
        if session_id:
            session_ids.add(session_id)
    return sorted(session_ids)


def pane_claude_session_files(pane_pid: str | None) -> list[Path]:
    project_root = HOME / ".claude" / "projects"
    files: set[Path] = set()
    for session_id in pane_claude_session_ids(pane_pid):
        files.update(project_root.rglob(f"{session_id}.jsonl"))
    return sorted(files)


def latest_codex_rollout() -> Path | None:
    session_root = HOME / ".codex" / "sessions"
    files = sorted(session_root.rglob("rollout-*.jsonl"))
    return files[-1] if files else None


def latest_claude_project_log() -> Path | None:
    project_root = HOME / ".claude" / "projects"
    files = sorted(project_root.rglob("*.jsonl"))
    return files[-1] if files else None


def read_jsonl_entries(path: Path) -> list[dict]:
    entries: list[dict] = []
    try:
        with path.open() as handle:
            for line in handle:
                line = line.strip()
                if not line:
                    continue
                try:
                    payload = json.loads(line)
                except json.JSONDecodeError:
                    continue
                if isinstance(payload, dict):
                    entries.append(payload)
    except OSError:
        return []
    return entries


def codex_assistant_message(entry: dict) -> str:
    payload = entry.get("payload", {})
    if payload.get("type") != "message" or payload.get("role") != "assistant":
        return ""
    parts: list[str] = []
    for item in payload.get("content", []):
        if item.get("type") in {"text", "output_text"}:
            parts.append(item.get("text", ""))
    return clean(" ".join(parts))


def codex_rollout_label(path: Path) -> str:
    entries = read_jsonl_entries(path)
    meta = next((entry.get("payload", {}) for entry in entries if entry.get("type") == "session_meta"), {})
    source = meta.get("source", {})
    if not isinstance(source, dict):
        source = {}
    thread_spawn = source.get("subagent", {}).get("thread_spawn", {})
    actor_nick = meta.get("agent_nickname") or thread_spawn.get("agent_nickname") or "Codex"
    actor_role = meta.get("agent_role") or thread_spawn.get("agent_role") or ""

    status = "Active"
    if any(entry.get("type") == "event_msg" and entry.get("payload", {}).get("type") == "task_complete" for entry in entries):
        status = "Completed"
    elif any(entry.get("type") == "event_msg" and entry.get("payload", {}).get("type") == "turn_aborted" for entry in entries):
        status = "Aborted"
    elif any(entry.get("type") == "event_msg" and entry.get("payload", {}).get("type") == "task_started" for entry in entries):
        status = "Started"

    completion_summary = ""
    commentary_summary = ""
    assistant_summary = ""
    for entry in entries:
        payload = entry.get("payload", {})
        if entry.get("type") == "event_msg" and payload.get("type") == "task_complete":
            completion_summary = clean(payload.get("last_agent_message", ""))
        elif entry.get("type") == "event_msg" and payload.get("type") == "agent_message":
            commentary_summary = clean(payload.get("message", ""))
        else:
            message = codex_assistant_message(entry)
            if message:
                assistant_summary = message

    summary_source = completion_summary or commentary_summary or assistant_summary
    label = f"{actor_nick}"
    if actor_role:
        label += f" [{actor_role}]"
    label += f": {status}"
    if summary_source:
        label += f" - {headline(summary_source, 96)}"
    return label


def claude_session_label(path: Path) -> str:
    if "/subagents/agent-" in str(path):
        return f"subagent {path.stem.removeprefix('agent-')}"
    return f"session {path.stem}"


def text_summary(content: object, limit: int) -> str:
    if isinstance(content, str):
        return headline(content, limit)
    if isinstance(content, list):
        parts: list[str] = []
        for item in content:
            if isinstance(item, dict) and item.get("type") == "text":
                parts.append(item.get("text", ""))
        return headline(" ".join(parts), limit) if parts else ""
    return ""


def format_codex_summary(entry: dict) -> str | None:
    entry_type = entry.get("type")
    payload = entry.get("payload", {})
    if entry_type == "event_msg" and payload.get("type") == "task_started":
        return f"start\tturn {(payload.get('turn_id') or '')[:8]}"
    if entry_type == "event_msg" and payload.get("type") == "task_complete":
        return f"done\t{headline(payload.get('last_agent_message', ''), 110)}"
    if entry_type == "event_msg" and payload.get("type") == "agent_message":
        return f"note\t{headline(payload.get('message', ''), 110)}"
    if entry_type == "event_msg" and payload.get("type") == "user_message":
        return f"user\t{headline(payload.get('message', ''), 110)}"
    if entry_type == "event_msg" and payload.get("type") == "turn_aborted":
        return "abort\tturn"
    if entry_type == "response_item" and payload.get("type") == "function_call" and payload.get("name") == "spawn_agent":
        return f"spawn\t{payload.get('name', '')}"
    return None


def nested_get(obj: object, *keys: object) -> object:
    current = obj
    for key in keys:
        if isinstance(key, int):
            if not isinstance(current, list) or key >= len(current):
                return None
            current = current[key]
        else:
            if not isinstance(current, dict):
                return None
            current = current.get(key)
    return current


def format_claude_summary(entry: dict) -> str | None:
    entry_type = entry.get("type")
    if entry_type == "assistant":
        content = nested_get(entry, "message", "content") or []
        if isinstance(content, list) and content:
            first = content[0]
            if isinstance(first, dict) and first.get("type") == "tool_use":
                return f"tool\t{first.get('name', '?')}"
        summary = text_summary(content, 96)
        return f"reply\t{summary}" if summary else "assistant\tupdate"
    if entry_type == "user":
        content = nested_get(entry, "message", "content")
        if isinstance(content, list) and content:
            first = content[0]
            if isinstance(first, dict) and first.get("type") == "tool_result":
                return None
            summary = text_summary(content, 96)
            if summary:
                return f"user\t{summary}"
            if isinstance(first, dict):
                return f"user\t{first.get('type', 'message')}"
        return None
    if entry_type == "progress":
        data = entry.get("data", {})
        progress_type = data.get("type")
        if progress_type == "query_update":
            return f"search\t{clip(clean(data.get('query', '')), 96)}"
        if progress_type == "search_results_received":
            return f"search\t{data.get('resultCount', 0)} results"
        if progress_type == "agent_progress":
            prompt = data.get("prompt")
            if not prompt:
                prompt = nested_get(data, "message", "message", "content", 0, "text") or ""
            return f"delegate\t{clip(clean(str(prompt)), 96)}"
        if progress_type == "hook_progress":
            return None
        return f"progress\t{progress_type or 'update'}"
    if entry_type == "system":
        return f"system\t{entry.get('subtype', 'event')}"
    if entry_type == "agent-setting":
        return f"agent\t{entry.get('agentSetting', 'set')}"
    return None


def format_summary(kind: str, entry: dict) -> str | None:
    summary = format_codex_summary(entry) if kind == "codex" else format_claude_summary(entry)
    if not summary:
        return None
    timestamp = str(entry.get("timestamp", ""))
    time_part = re.sub(r".*T|\..*Z$", "", timestamp)
    if kind == "codex":
        return redact_text(f"{time_part}\t{summary}")
    session_id = str(entry.get("sessionId", ""))[:8]
    return redact_text(f"{time_part}\t{session_id}\t{summary}")


def format_details(kind: str, entry: dict) -> str | None:
    if kind == "codex":
        payload = {
            "timestamp": entry.get("timestamp"),
            "type": entry.get("type"),
            "payload": entry.get("payload"),
        }
    else:
        payload = {
            "timestamp": entry.get("timestamp"),
            "sessionId": entry.get("sessionId"),
            "type": entry.get("type"),
            "subtype": entry.get("subtype"),
            "agentSetting": entry.get("agentSetting"),
            "message": entry.get("message"),
        }
    return redact_text(json.dumps(payload, ensure_ascii=False, indent=2))


def format_line(kind: str, mode: str, raw_line: str) -> str | None:
    if mode == "raw":
        return redact_text(raw_line)
    try:
        entry = json.loads(raw_line)
    except json.JSONDecodeError:
        return None
    if not isinstance(entry, dict):
        return None
    if mode == "summary":
        return format_summary(kind, entry)
    if mode == "details":
        return format_details(kind, entry)
    raise ValueError(f"Unknown mode: {mode}")


def emit_formatted(kind: str, mode: str, lines: Iterable[str]) -> None:
    for raw_line in lines:
        text = raw_line.rstrip("\n")
        if not text:
            continue
        formatted = format_line(kind, mode, text)
        if formatted is None:
            continue
        sys.stdout.write(formatted + "\n")
    sys.stdout.flush()


def tail_lines(path: Path, count: int) -> list[str]:
    try:
        with path.open() as handle:
            lines = handle.readlines()
    except OSError:
        return []
    return lines[-count:]


class FileTail:
    def __init__(self, path: Path, start_at_end: bool) -> None:
        self.path = path
        self.offset = path.stat().st_size if start_at_end and path.exists() else 0
        self.pending = ""

    def read_new_lines(self) -> list[str]:
        try:
            size = self.path.stat().st_size
        except OSError:
            return []
        if size < self.offset:
            self.offset = 0
            self.pending = ""
        if size == self.offset:
            return []
        try:
            with self.path.open() as handle:
                handle.seek(self.offset)
                chunk = handle.read()
        except OSError:
            return []
        self.offset = size
        data = self.pending + chunk
        lines = data.splitlines(keepends=True)
        if lines and not lines[-1].endswith("\n"):
            self.pending = lines.pop()
        else:
            self.pending = ""
        return lines


def announce(kind: str, path: Path) -> None:
    if kind == "codex":
        label = codex_rollout_label(path) or path.name
        sys.stderr.write(f"codex-watch: +{label}\n")
    else:
        sys.stderr.write(f"claude-watch: +{claude_session_label(path)}\n")
    sys.stderr.flush()


def discover_files(kind: str, pane_pid: str | None) -> list[Path]:
    return pane_codex_rollout_files(pane_pid) if kind == "codex" else pane_claude_session_files(pane_pid)


def run_once(kind: str, mode: str, pane_pid: str | None, pane_target: str | None) -> int:
    active_pid = current_pane_pid(pane_pid, pane_target)
    if active_pid:
        files = discover_files(kind, active_pid)
        if kind == "claude" and not files:
            target = pane_target or pane_pid or ""
            sys.stderr.write(f"claude-watch: no active claude session in pane {target}\n")
            return 0
        count = 20 if kind == "codex" else 40
        for path in files:
            emit_formatted(kind, mode, tail_lines(path, count))
        return 0

    source = latest_codex_rollout() if kind == "codex" else latest_claude_project_log()
    if not source or not source.is_file():
        label = "latest rollout source not readable" if kind == "codex" else "source not readable"
        sys.stderr.write(f"{kind}-watch: {label}\n")
        return 1
    emit_formatted(kind, mode, tail_lines(source, 20))
    return 0


def run_follow(kind: str, mode: str, pane_pid: str | None, pane_target: str | None) -> int:
    tracked: dict[Path, FileTail] = {}
    stop = False

    def handle_signal(signum: int, _frame: object) -> None:
        nonlocal stop
        stop = True
        if signum == signal.SIGINT:
            raise KeyboardInterrupt

    signal.signal(signal.SIGTERM, handle_signal)
    signal.signal(signal.SIGINT, handle_signal)

    active_pid = current_pane_pid(pane_pid, pane_target)
    if not active_pid:
        source = latest_codex_rollout() if kind == "codex" else latest_claude_project_log()
        if not source or not source.is_file():
            label = "latest rollout source not readable" if kind == "codex" else "source not readable"
            sys.stderr.write(f"{kind}-watch: {label}\n")
            return 1
        emit_formatted(kind, mode, tail_lines(source, 20))
        tracked[source] = FileTail(source, start_at_end=True)

    while not stop:
        active_pid = current_pane_pid(pane_pid, pane_target)
        if active_pid:
            for path in discover_files(kind, active_pid):
                if path not in tracked:
                    tracked[path] = FileTail(path, start_at_end=True)
                    announce(kind, path)
        for follower in list(tracked.values()):
            emit_formatted(kind, mode, follower.read_new_lines())
        time.sleep(POLL_INTERVAL)
    return 0


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument("kind", choices=["claude", "codex"])
    parser.add_argument("--summary", dest="mode", action="store_const", const="summary", default="summary")
    parser.add_argument("--details", dest="mode", action="store_const", const="details")
    parser.add_argument("--raw", dest="mode", action="store_const", const="raw")
    parser.add_argument("--once", action="store_true")
    parser.add_argument("--pane-pid")
    parser.add_argument("--pane-target")
    return parser.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    if args.once:
        return run_once(args.kind, args.mode, args.pane_pid, args.pane_target)
    return run_follow(args.kind, args.mode, args.pane_pid, args.pane_target)


if __name__ == "__main__":
    try:
        raise SystemExit(main(sys.argv[1:]))
    except KeyboardInterrupt:
        raise SystemExit(130)
