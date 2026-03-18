import argparse
import os
import re
import sys
import textwrap
import time

from ai.agent import AgentDeps, build_agent
from ai.models import CanvasQuestionRequest
from ai.superplane_client import SuperplaneClient, SuperplaneClientConfig

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
        value
        or os.getenv("SUPERPLANE_CANVAS_ID", "")
        or os.getenv("CANVAS_ID", "")
    ).strip()
    if resolved:
        return resolved
    raise ValueError("Missing required setting: SUPERPLANE_CANVAS_ID (or CANVAS_ID).")


def elapsed_since(started_at: float) -> str:
    return f"{(time.perf_counter() - started_at):7.3f}s"


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
        "--model",
        default=os.getenv("AI_MODEL", "test"),
        help="PydanticAI model identifier.",
    )
    args = parser.parse_args()
    canvas_id_arg = normalize_optional_setting(args.canvas_id)

    if args.healthcheck:
        print("ok")
        return

    if not args.question and not args.interactive:
        raise ValueError("Provide --question or --interactive.")

    if args.model == "test":
        if args.interactive:
            print("Canvas Q&A Console (test model). Type 'exit' to quit.")
            while True:
                question = input("> ").strip()
                if question.lower() == "/reload":
                    print("[reload] restarting...")
                    raise SystemExit(RELOAD_EXIT_CODE)
                if question.lower() in {"exit", "quit"}:
                    break
                if not question:
                    continue
                payload = CanvasQuestionRequest(question=question, canvas_id=canvas_id_arg)
                print(payload.question)
        else:
            payload = CanvasQuestionRequest(question=args.question, canvas_id=canvas_id_arg)
            print(payload.question)
        return

    base_url = require_setting(args.base_url, "SUPERPLANE_BASE_URL")
    token = require_setting(args.token, "SUPERPLANE_API_TOKEN")
    org_id = require_setting(args.org_id, "SUPERPLANE_ORG_ID")
    canvas_id = require_canvas_id(canvas_id_arg)

    client = SuperplaneClient(
        SuperplaneClientConfig(
            base_url=base_url,
            api_token=token,
            organization_id=org_id,
        )
    )

    agent = build_agent(model=args.model)
    deps = AgentDeps(
        client=client,
        default_canvas_id=canvas_id,
        show_tool_calls=True,
    )

    if args.interactive:
        print("Canvas Q&A Console. Type 'exit' to quit.")
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
            started_at = time.perf_counter()
            deps.question_started_at = started_at
            deps.waiting_message_printed = False
            deps.allow_canvas_details = False
            print(
                f"{color(elapsed_since(started_at), '90')} {color('Started', '34')}",
                flush=True,
            )
            try:
                result = agent.run_sync(user_prompt=payload.question, deps=deps)
            except Exception as error:
                elapsed_ms = (time.perf_counter() - started_at) * 1000
                print(f"Error: {error}")
                print(
                    f"{color(elapsed_since(started_at), '90')} "
                    f"{color('Failed', '31')} (answer_elapsed_ms={elapsed_ms:.1f})"
                )
                continue
            elapsed_ms = (time.perf_counter() - started_at) * 1000
            print(
                f"{color(elapsed_since(started_at), '90')} "
                f"{color('[status]', '33')} Final answer ready."
            )
            print(_render_answer(result.output.answer, started_at))
            print(
                f"{color(elapsed_since(started_at), '90')} "
                f"{color('Completed', '32')} (answer_elapsed_ms={elapsed_ms:.1f})"
            )
        return

    payload = CanvasQuestionRequest(question=args.question, canvas_id=canvas_id)
    started_at = time.perf_counter()
    deps.question_started_at = started_at
    deps.waiting_message_printed = False
    deps.allow_canvas_details = False
    print(
        f"{color(elapsed_since(started_at), '90')} {color('Started', '34')}",
        flush=True,
    )
    try:
        result = agent.run_sync(user_prompt=payload.question, deps=deps)
    except Exception as error:
        raise SystemExit(f"Error: {error}") from error
    elapsed_ms = (time.perf_counter() - started_at) * 1000
    print(
        f"{color(elapsed_since(started_at), '90')} "
        f"{color('[status]', '33')} Final answer ready."
    )
    print(_render_answer(result.output.answer, started_at))
    print(
        f"{color(elapsed_since(started_at), '90')} "
        f"{color('Completed', '32')} (answer_elapsed_ms={elapsed_ms:.1f})"
    )


if __name__ == "__main__":
    main()

