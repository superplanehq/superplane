#!/usr/bin/env python3
"""Format Claude Code stream-json NDJSON into readable live logs."""

from __future__ import annotations

import json
import sys
from typing import Any


TOOL_RESULT_MAX_CHARS = 800
TOOL_RESULT_MAX_LINES = 24


def main() -> int:
    streamed_text = False
    in_text = False

    for raw in sys.stdin:
        line = raw.strip()
        if not line:
            continue
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            print(line, flush=True)
            continue
        if not isinstance(event, dict):
            continue

        event_type = event.get("type")
        if event_type == "system":
            format_system(event)
        elif event_type == "stream_event":
            streamed_text, in_text = format_stream_event(event, streamed_text, in_text)
        elif event_type == "assistant":
            if in_text:
                print(flush=True)
                in_text = False
            format_assistant(event, streamed_text)
            streamed_text = False
        elif event_type == "user":
            if in_text:
                print(flush=True)
                in_text = False
            format_user(event)
        elif event_type == "result":
            if in_text:
                print(flush=True)
                in_text = False
            format_result(event)
        elif event_type == "rate_limit_event":
            print("Rate limit notice — waiting to continue…", flush=True)

    if in_text:
        print(flush=True)
    return 0


def format_system(event: dict[str, Any]) -> None:
    if event.get("subtype") != "init":
        if event.get("subtype") == "api_retry":
            attempt = event.get("attempt", "?")
            max_retries = event.get("max_retries", "?")
            delay = event.get("retry_delay_ms")
            delay_part = f" in {delay}ms" if delay is not None else ""
            print(f"Retrying API ({attempt}/{max_retries}){delay_part}…", flush=True)
        return

    parts = ["Claude Code started"]
    if model := event.get("model"):
        parts.append(f"model={model}")
    if cwd := event.get("cwd"):
        parts.append(f"cwd={cwd}")
    print(" · ".join(parts), flush=True)
    print(flush=True)


def format_stream_event(
    event: dict[str, Any], streamed_text: bool, in_text: bool
) -> tuple[bool, bool]:
    payload = event.get("event")
    if not isinstance(payload, dict):
        return streamed_text, in_text

    kind = payload.get("type")
    if kind == "content_block_start":
        block = payload.get("content_block")
        if isinstance(block, dict) and block.get("type") == "text":
            if not in_text:
                print("Claude", flush=True)
            return streamed_text, True
        return streamed_text, in_text

    if kind == "content_block_delta":
        delta = payload.get("delta")
        if isinstance(delta, dict) and delta.get("type") == "text_delta":
            text = delta.get("text")
            if isinstance(text, str) and text:
                if not in_text:
                    print("Claude", flush=True)
                    in_text = True
                sys.stdout.write(text)
                sys.stdout.flush()
                return True, in_text
        return streamed_text, in_text

    if kind == "content_block_stop" and in_text:
        print(flush=True)
        print(flush=True)
        return streamed_text, False

    return streamed_text, in_text


def format_assistant(event: dict[str, Any], streamed_text: bool) -> None:
    message = event.get("message")
    if not isinstance(message, dict):
        return
    content = message.get("content")
    if not isinstance(content, list):
        return

    for block in content:
        if not isinstance(block, dict):
            continue
        block_type = block.get("type")
        if block_type == "text" and not streamed_text:
            text = block.get("text")
            if isinstance(text, str) and text.strip():
                print("Claude", flush=True)
                print(text.rstrip(), flush=True)
                print(flush=True)
        elif block_type == "tool_use":
            print(format_tool_use(block), flush=True)
            print(flush=True)
        elif block_type == "thinking":
            thinking = block.get("thinking")
            if isinstance(thinking, str) and thinking.strip():
                print("Thinking", flush=True)
                print(truncate_text(thinking.strip()), flush=True)
                print(flush=True)


def format_user(event: dict[str, Any]) -> None:
    message = event.get("message")
    if not isinstance(message, dict):
        return
    content = message.get("content")
    if not isinstance(content, list):
        return

    for block in content:
        if not isinstance(block, dict):
            continue
        if block.get("type") != "tool_result":
            continue
        body = tool_result_text(block.get("content"))
        if not body.strip():
            print("← tool result (empty)", flush=True)
            print(flush=True)
            continue
        print("← tool result", flush=True)
        print(indent(truncate_text(body.rstrip())), flush=True)
        print(flush=True)


def format_result(event: dict[str, Any]) -> None:
    is_error = bool(event.get("is_error"))
    status = "failed" if is_error else "done"
    parts = [f"✓ {status}" if not is_error else f"✗ {status}"]

    if (turns := event.get("num_turns")) is not None:
        parts.append(f"{turns} turns")
    if (cost := event.get("total_cost_usd")) is not None:
        try:
            parts.append(f"${float(cost):.4f}")
        except (TypeError, ValueError):
            parts.append(f"${cost}")
    if (duration_ms := event.get("duration_ms")) is not None:
        try:
            parts.append(f"{float(duration_ms) / 1000:.1f}s")
        except (TypeError, ValueError):
            pass

    print(" · ".join(parts), flush=True)

    result = event.get("result")
    if isinstance(result, str) and result.strip() and is_error:
        print(result.rstrip(), flush=True)


def format_tool_use(block: dict[str, Any]) -> str:
    name = str(block.get("name") or "tool")
    raw_input = block.get("input")
    detail = tool_input_detail(name, raw_input)
    if detail:
        return f"→ {name}\n{indent(detail)}"
    return f"→ {name}"


def tool_input_detail(name: str, raw_input: Any) -> str:
    if not isinstance(raw_input, dict):
        if raw_input is None:
            return ""
        return truncate_text(str(raw_input))

    lowered = name.lower()
    if lowered == "bash":
        command = raw_input.get("command")
        if isinstance(command, str) and command.strip():
            return command.strip()
    if lowered in {"read", "write", "edit", "notebookedit"}:
        for key in ("file_path", "path", "notebook_path"):
            value = raw_input.get(key)
            if isinstance(value, str) and value.strip():
                detail = value.strip()
                if lowered in {"write", "edit"} and isinstance(raw_input.get("content"), str):
                    content = raw_input["content"]
                    detail += f"\n({len(content)} chars)"
                return detail
    if lowered == "grep":
        parts = []
        if pattern := raw_input.get("pattern"):
            parts.append(f"pattern: {pattern}")
        if path := raw_input.get("path"):
            parts.append(f"path: {path}")
        if parts:
            return "\n".join(parts)
    if lowered == "glob":
        if pattern := raw_input.get("pattern"):
            return str(pattern)

    try:
        return truncate_text(json.dumps(raw_input, ensure_ascii=False, indent=2))
    except (TypeError, ValueError):
        return truncate_text(str(raw_input))


def tool_result_text(content: Any) -> str:
    if content is None:
        return ""
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        parts: list[str] = []
        for item in content:
            if isinstance(item, dict):
                text = item.get("text")
                if isinstance(text, str):
                    parts.append(text)
                else:
                    parts.append(json.dumps(item, ensure_ascii=False))
            else:
                parts.append(str(item))
        return "\n".join(parts)
    return str(content)


def truncate_text(text: str) -> str:
    lines = text.splitlines()
    if len(lines) > TOOL_RESULT_MAX_LINES:
        kept = lines[:TOOL_RESULT_MAX_LINES]
        omitted = len(lines) - TOOL_RESULT_MAX_LINES
        text = "\n".join(kept) + f"\n… ({omitted} more lines)"
    if len(text) > TOOL_RESULT_MAX_CHARS:
        text = text[: TOOL_RESULT_MAX_CHARS - 1].rstrip() + "…"
    return text


def indent(text: str, prefix: str = "  ") -> str:
    return "\n".join(prefix + line if line else prefix.rstrip() for line in text.splitlines())


if __name__ == "__main__":
    raise SystemExit(main())
