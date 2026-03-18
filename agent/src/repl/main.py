import argparse
import json
import os
import re
import sys
import textwrap
import time
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

from ai.models import CanvasQuestionRequest
from ai.repl_web import ReplWebServer, ReplWebServerConfig

RELOAD_EXIT_CODE = 3


def use_color() -> bool:
    return sys.stdout.isatty() and not os.getenv("NO_COLOR")


def color(text: str, ansi_code: str) -> str:
    if not use_color():
        return text
    return f"\033[{ansi_code}m{text}\033[0m"


def require_setting(value: str | None, env_name: str) -> str:
    resolved = (value or os.getenv(env_name, "")).strip()
    if resolved:
        return resolved
    raise ValueError(f"Missing required setting: {env_name}")


def normalize_optional_setting(value: str | None) -> str | None:
    if value is None:
        return None
    normalized = value.strip()
    return normalized or None


def require_canvas_id(value: str | None) -> str:
    resolved = (
        value or os.getenv("SUPERPLANE_CANVAS_ID", "") or os.getenv("CANVAS_ID", "")
    ).strip()
    if resolved:
        return resolved
    raise ValueError("Missing required setting: SUPERPLANE_CANVAS_ID (or CANVAS_ID).")


def elapsed_since(started_at: float) -> str:
    return f"{(time.perf_counter() - started_at):7.3f}s"


def _type_out(text: str, chars_per_second: float | None = None) -> None:
    if not text:
        return
    if not sys.stdout.isatty():
        print(text, end="", flush=True)
        return

    configured_rate = chars_per_second
    if configured_rate is None:
        configured_rate = float(os.getenv("AI_REPL_TYPEWRITER_CPS", "500"))
    if configured_rate <= 0:
        print(text, end="", flush=True)
        return

    delay_seconds = 1.0 / configured_rate
    for char in text:
        print(char, end="", flush=True)
        if char not in {" ", "\n", "\t"}:
            time.sleep(delay_seconds)


def _render_answer(answer: str, started_at: float, max_width: int = 140) -> str:
    timestamp_width = len(elapsed_since(started_at))
    status_column = timestamp_width + 1  # account for one space after timestamp
    base_indent = " " * status_column
    available_width = max(20, max_width - status_column)

    bullet_pattern = re.compile(r"^([*-]\s+|\d+[.)]\s+)(.*)$")
    wrapped_lines: list[str] = []

    for raw_line in answer.splitlines():
        stripped = raw_line.strip()
        if not stripped:
            wrapped_lines.append("")
            continue

        bullet_match = bullet_pattern.match(stripped)
        if bullet_match:
            bullet_prefix = bullet_match.group(1)
            content = bullet_match.group(2).strip()
            wrapped_lines.extend(
                textwrap.wrap(
                    content,
                    width=available_width,
                    initial_indent=base_indent + bullet_prefix,
                    subsequent_indent=base_indent + (" " * len(bullet_prefix)),
                    break_long_words=False,
                    break_on_hyphens=False,
                )
            )
            continue

        wrapped_lines.extend(
            textwrap.wrap(
                stripped,
                width=available_width,
                initial_indent=base_indent,
                subsequent_indent=base_indent,
                break_long_words=False,
                break_on_hyphens=False,
            )
        )

    return "\n".join(wrapped_lines)


def _parse_stream_event(raw_line: bytes) -> dict[str, Any] | None:
    line = raw_line.decode("utf-8").strip()
    if not line or not line.startswith("data:"):
        return None
    payload = line[len("data:") :].strip()
    if not payload:
        return None
    parsed = json.loads(payload)
    if not isinstance(parsed, dict):
        return None
    return parsed


def _stream_repl_answer(
    repl_web_url: str,
    payload: CanvasQuestionRequest,
    model: str,
    base_url: str | None = None,
    token: str | None = None,
    org_id: str | None = None,
) -> str:
    started_at = time.perf_counter()
    request_payload = payload.model_dump(mode="json")
    request_payload["model"] = model
    request_payload["base_url"] = base_url
    request_payload["token"] = token
    request_payload["org_id"] = org_id
    request_body = json.dumps(request_payload).encode("utf-8")
    request = Request(
        url=f"{repl_web_url.rstrip('/')}/v1/repl/stream",
        data=request_body,
        method="POST",
        headers={
            "content-type": "application/json",
            "accept": "text/event-stream",
        },
    )

    chunks: list[str] = []
    run_failed_error: str | None = None
    first_model_delta_elapsed_ms: float | None = None
    tool_elapsed_total_ms = 0.0
    final_answer_announced = False
    try:
        with urlopen(request, timeout=30) as response:
            while True:
                raw_line = response.readline()
                if not raw_line:
                    break
                event = _parse_stream_event(raw_line)
                if not event:
                    continue
                event_type = event.get("type")
                if event_type == "run_started":
                    print(
                        f"{color(elapsed_since(started_at), '90')} {color('Started', '34')}",
                        flush=True,
                    )
                    continue
                if event_type == "tool_started":
                    tool_name = event.get("tool_name", "unknown")
                    print(
                        f"{color(elapsed_since(started_at), '90')} "
                        f"{color('[tool]', '36')} {tool_name} started",
                        flush=True,
                    )
                    continue
                if event_type == "tool_finished":
                    tool_name = event.get("tool_name", "unknown")
                    elapsed_ms = event.get("elapsed_ms")
                    if isinstance(elapsed_ms, int | float):
                        tool_elapsed_total_ms += float(elapsed_ms)
                        print(
                            f"{color(elapsed_since(started_at), '90')} "
                            f"{color('[tool]', '36')} "
                            f"{tool_name} completed ({elapsed_ms:.1f}ms)",
                            flush=True,
                        )
                    else:
                        print(
                            f"{color(elapsed_since(started_at), '90')} "
                            f"{color('[tool]', '36')} {tool_name} completed",
                            flush=True,
                        )
                    continue
                if event_type == "model_delta":
                    content = event.get("content")
                    if content:
                        if not isinstance(content, str):
                            continue
                        if not final_answer_announced:
                            print(
                                f"{color(elapsed_since(started_at), '90')} "
                                f"{color('[status]', '33')} Final answer ready.",
                                flush=True,
                            )
                            final_answer_announced = True
                        if first_model_delta_elapsed_ms is None:
                            first_model_delta_elapsed_ms = (time.perf_counter() - started_at) * 1000
                        chunks.append(content)
                        _type_out(content)
                    continue
                if event_type == "final_answer":
                    output = event.get("output")
                    if not final_answer_announced:
                        print(
                            f"{color(elapsed_since(started_at), '90')} "
                            f"{color('[status]', '33')} Final answer ready.",
                            flush=True,
                        )
                        final_answer_announced = True
                    if isinstance(output, str):
                        if not chunks:
                            chunks.append(output)
                            _type_out(output)
                        continue
                    if isinstance(output, dict):
                        answer = output.get("answer")
                        if isinstance(answer, str):
                            if not chunks:
                                chunks.append(answer)
                                _type_out(answer)
                    continue
                if event_type == "run_failed":
                    error_message = event.get("error")
                    if isinstance(error_message, str):
                        run_failed_error = error_message
                    else:
                        run_failed_error = "Unknown run error."
                    continue
                if event_type == "run_completed":
                    continue
                if event_type == "done":
                    break
    except HTTPError as error:
        response_text = ""
        try:
            response_text = error.read().decode("utf-8")
        except Exception:
            response_text = ""
        details = f" HTTP {error.code}"
        if response_text:
            details += f": {response_text}"
        raise RuntimeError(f"Test REPL web request failed.{details}") from error
    except URLError as error:
        raise RuntimeError(
            "Failed to reach the REPL web application. "
            "Check --repl-web-url and make sure the server is running."
        ) from error

    if run_failed_error is not None:
        print(
            f"{color(elapsed_since(started_at), '90')} "
            f"{color('Failed', '31')} ({run_failed_error})",
            flush=True,
        )
        raise RuntimeError(run_failed_error)

    print()
    return "".join(chunks).strip()


def main() -> None:
    parser = argparse.ArgumentParser(description="Canvas Q&A CLI.")
    parser.add_argument("--healthcheck", action="store_true", help="Return health status.")
    parser.add_argument("--question", help="Question about a canvas.")
    parser.add_argument(
        "--interactive",
        action="store_true",
        help="Start interactive console for multiple questions.",
    )
    parser.add_argument(
        "--canvas-id",
        help="Canvas ID to inspect.",
    )
    parser.add_argument("--base-url", help="Superplane base URL.")
    parser.add_argument("--token", help="Superplane API token.")
    parser.add_argument("--org-id", help="Superplane organization ID.")
    parser.add_argument(
        "--serve-repl-web",
        action="store_true",
        help="Start REPL web server and block.",
    )
    parser.add_argument(
        "--serve-test-repl-web",
        action="store_true",
        help="Deprecated alias for --serve-repl-web.",
    )
    parser.add_argument(
        "--test-repl-web-host",
        default=os.getenv("AI_TEST_REPL_WEB_HOST", "127.0.0.1"),
        help="Host for test REPL web server.",
    )
    parser.add_argument(
        "--test-repl-web-port",
        type=int,
        default=int(os.getenv("AI_TEST_REPL_WEB_PORT", "8090")),
        help="Port for test REPL web server.",
    )
    parser.add_argument(
        "--repl-web-url",
        default=os.getenv("AI_REPL_WEB_URL"),
        help="Base URL for REPL web application.",
    )
    parser.add_argument(
        "--start-repl-web",
        action="store_true",
        help="Auto-start REPL web server before handling requests.",
    )
    parser.add_argument(
        "--start-test-repl-web",
        action="store_true",
        help="Deprecated alias for --start-repl-web.",
    )
    parser.add_argument(
        "--model",
        default=os.getenv("AI_MODEL", "test"),
        help="PydanticAI model identifier.",
    )
    args = parser.parse_args()
    canvas_id_arg = normalize_optional_setting(args.canvas_id)

    if args.healthcheck:
        print("ok")
        return

    if args.serve_repl_web or args.serve_test_repl_web:
        server = ReplWebServer(
            ReplWebServerConfig(host=args.test_repl_web_host, port=args.test_repl_web_port)
        )
        print(f"Serving REPL web app at {server.base_url}", flush=True)
        try:
            server.serve_forever()
        except KeyboardInterrupt:
            return

    if not args.question and not args.interactive:
        raise ValueError("Provide --question or --interactive.")

    repl_web_url = normalize_optional_setting(args.repl_web_url)
    server: ReplWebServer | None = None
    should_start_server = args.start_repl_web or args.start_test_repl_web or repl_web_url is None
    if should_start_server:
        server = ReplWebServer(
            ReplWebServerConfig(
                host=args.test_repl_web_host,
                port=args.test_repl_web_port,
            )
        )
        server.start()
        repl_web_url = server.base_url
        print(f"REPL web app started at {repl_web_url}")
    if repl_web_url is None:
        raise ValueError(
            "Missing REPL web URL. Set --repl-web-url or AI_REPL_WEB_URL, or pass --start-repl-web."
        )

    base_url: str | None = None
    token: str | None = None
    org_id: str | None = None
    canvas_id = canvas_id_arg
    if args.model != "test":
        base_url = require_setting(args.base_url, "SUPERPLANE_BASE_URL")
        token = require_setting(args.token, "SUPERPLANE_API_TOKEN")
        org_id = require_setting(args.org_id, "SUPERPLANE_ORG_ID")
        canvas_id = require_canvas_id(canvas_id_arg)

    console_label = (
        "Canvas Q&A Console (test model)." if args.model == "test" else "Canvas Q&A Console."
    )
    if args.interactive:
        print(f"{console_label} Type 'exit' to quit.")
        try:
            while True:
                question = input("> ").strip()
                if question.lower() == "/reload":
                    print(
                        f"{color(elapsed_since(time.perf_counter()), '90')} "
                        f"{color('[reload]', '35')} restarting..."
                    )
                    raise SystemExit(RELOAD_EXIT_CODE)
                if question.lower() in {"exit", "quit"}:
                    break
                if not question:
                    continue
                payload = CanvasQuestionRequest(question=question, canvas_id=canvas_id)
                _stream_repl_answer(
                    repl_web_url=repl_web_url,
                    payload=payload,
                    model=args.model,
                    base_url=base_url,
                    token=token,
                    org_id=org_id,
                )
        finally:
            if server is not None:
                server.stop()
        return

    payload = CanvasQuestionRequest(question=args.question, canvas_id=canvas_id)
    try:
        _stream_repl_answer(
            repl_web_url=repl_web_url,
            payload=payload,
            model=args.model,
            base_url=base_url,
            token=token,
            org_id=org_id,
        )
    except Exception as error:
        raise SystemExit(f"Error: {error}") from error
    finally:
        if server is not None:
            server.stop()


if __name__ == "__main__":
    main()
