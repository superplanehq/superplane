from __future__ import annotations

import asyncio
import os
import evals.evaluators as evals
import time
import textwrap
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from pydantic_ai.messages import (
    FinalResultEvent,
    FunctionToolCallEvent,
    FunctionToolResultEvent,
)
from pydantic_ai.usage import RunUsage
from pydantic_ai.run import AgentRunResultEvent
from pydantic_evals import Case, Dataset

from ai.agent import AgentDeps, build_agent, build_prompt
from ai.jsonutil import to_jsonable
from ai.models import CanvasAnswer, CanvasQuestionRequest
from ai.superplane_client import SuperplaneClient, SuperplaneClientConfig
from evals.report import ReportBuilder

dataset = Dataset(
    cases=[
        Case(
            name="manual_run_then_two_noops",
            inputs=(
                "Build me a basic workflow that starts with a manual run and runs two noop actions"
            ),
            evaluators=[
              evals.CanvasHasTrigger("start"),
              evals.CanvasHasNode("noop", count=2),
              evals.CanvasTotalNodeCount(count=3),
            ],
        ),

        Case(
            name="github_and_slack",
            inputs=(
                "Listen to pull-request comments and send a slack message when "
                "a comment is made"
            ),
            evaluators=[
                evals.CanvasHasTrigger("github.onPRComment"),
                evals.CanvasHasNode("slack.sendTextMessage", count=1),
                evals.CanvasTotalNodeCount(count=2),
            ],
        ),
        Case(
            name="github_issue_opened_to_discord",
            inputs=(
                "When a GitHub issue is opened, post a Discord message "
                "that includes the issue title"
            ),
            evaluators=[
                evals.CanvasHasTrigger("github.onIssue"),
                evals.CanvasHasNode("discord.sendTextMessage"),
                evals.CanvasTotalNodeCount(count=2),
                evals.NoDollarDataAsRoot(),
            ],
        ),
        Case(
            name="pr_comment_filter_slack_message_chain",
            inputs=(
                "When a GitHub PR receives a comment, run the filter component then Slack "
                "send text message; the Slack message body should contain the name of the PR and the time the filter node was executed"
            ),
            evaluators=[
                evals.CanvasHasTrigger("github.onPRComment"),
                evals.CanvasHasNode("filter"),
                evals.CanvasHasNode("slack.sendTextMessage"),
                evals.CanvasTotalNodeCount(count=3),
                evals.BracketSelectorsMatchCanvasNames(
                    scan_scope="all",
                    require_at_least_one_selector=True,
                    target_block_name="slack.sendTextMessage",
                ),
            ],
        ),
        Case(
            name="ephemeral_pr_preview_machines",
            inputs=(
                "Build a workflow that creates ephemeral preview machines for pull requests. "
                "On PR open, create infra and post the preview URL to the PR. "
                "On PR close or after 48 hours, tear it down."
            ),
            evaluators=[
              evals.CanvasHasTrigger("github.onPullRequest"),
              evals.CanvasHasNode("daytona.createRepositorySandbox"),
              evals.CanvasHasNode("wait"),
              evals.CanvasHasNode("daytona.deleteSandbox", count=2),
              evals.CanvasHasWorkflow("github.onPullRequest", "...", "daytona.createRepositorySandbox", "...", "wait", "...", "daytona.deleteSandbox"),
              evals.CanvasHasWorkflow("github.onPullRequest", "...", "readMemory", "...", "daytona.deleteSandbox"),
            ],
        ),
        Case(
            name="agent_labeled_issue_auto_resolve",
            inputs="Build a workflow that auto-resolves GitHub issues",
            evaluators=[
              evals.CanvasHasTrigger("github.onIssue"),
              evals.CanvasHasWorkflow("github.onIssue", "...", "github.createIssueComment", "...", "daytona.executeCode", "...", "github.createIssueComment"),
            ],
        ),
    ],
)


class InteractionLogger:
    def __init__(self, run_id: str, case_names: list[str]) -> None:
        self._lock = asyncio.Lock()
        self._case_started_at_monotonic: dict[str, float] = {}
        output_dir = Path("/app/tmp/agent/evals")
        display_output_dir = Path("tmp/agent/evals")
        output_dir.mkdir(parents=True, exist_ok=True)
        run_suffix = run_id.split("_")[-1] if "_" in run_id else run_id

        self._files_by_case_name: dict[str, Any] = {}
        self._display_path_by_case_name: dict[str, str] = {}
        for index, case_name in enumerate(case_names, start=1):
            log_id = f"{run_suffix}-{index:02d}"
            filename = f"{log_id}.log"
            output_path = output_dir / filename
            file = output_path.open("w", encoding="utf-8")
            file.write(f"case={case_name}\n")
            file.flush()
            self._files_by_case_name[case_name] = file
            self._display_path_by_case_name[case_name] = str(display_output_dir / filename)

    @property
    def display_paths_by_case_name(self) -> dict[str, str]:
        return dict(self._display_path_by_case_name)

    async def log_case(self, case_name: str, line: str) -> None:
        file = self._files_by_case_name.get(case_name)
        if file is None:
            raise RuntimeError(f"No interaction log file configured for case {case_name!r}")
        lines = line.splitlines() or [line]
        async with self._lock:
            now = time.perf_counter()
            started_at = self._case_started_at_monotonic.get(case_name)
            if started_at is None:
                self._case_started_at_monotonic[case_name] = now
                elapsed_seconds = 0.0
            else:
                elapsed_seconds = now - started_at

            elapsed_timestamp = _format_elapsed(elapsed_seconds)
            file.write(f"[{elapsed_timestamp}] {lines[0]}\n")
            for continuation in lines[1:]:
                file.write(f"{continuation}\n")
            file.flush()

    def close(self) -> None:
        for file in self._files_by_case_name.values():
            file.close()


def load_env() -> dict[str, str]:
    return {
        "model": os.getenv("AI_MODEL", "").strip(),
        "base_url": "http://app:8000",
        "api_token": os.getenv("SUPERPLANE_API_TOKEN", "").strip(),
        "organization_id": os.getenv("EVAL_ORG_ID", "").strip(),
        "canvas_id": os.getenv("EVAL_CANVAS_ID", "").strip(),
    }

async def runner() -> None:
    env = load_env()
    cases = list(dataset.cases)
    eval_dataset = Dataset(cases=cases)

    deps = AgentDeps(
        client=SuperplaneClient(
            SuperplaneClientConfig(
                base_url=env["base_url"],
                api_token=env["api_token"],
                organization_id=env["organization_id"],
            )
        ),
        canvas_id=env["canvas_id"],
    )
    agent = build_agent(env["model"])
    # Keyed by case ``inputs`` string so usage matches when cases run concurrently.
    # Duplicate prompts across cases are not supported (detected under a lock).
    run_usages: dict[str, RunUsage] = {}
    usage_lock = asyncio.Lock()
    question_to_case_name: dict[str, str] = {}
    case_names: list[str] = []
    for index, case in enumerate(cases):
        if not isinstance(case.inputs, str):
            raise RuntimeError(
                f"Case {getattr(case, 'name', f'case_{index}')!r} has non-string input; "
                "eval logging requires string case inputs."
            )
        if case.inputs in question_to_case_name:
            raise RuntimeError(
                "Duplicate eval case inputs are not supported for logging correlation "
                f"(collision on {case.inputs[:120]!r}...)"
            )
        case_name = getattr(case, "name", f"case_{index}")
        question_to_case_name[case.inputs] = case_name
        case_names.append(case_name)

    interaction_logger = InteractionLogger(
        run_id=datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%S_%fZ"),
        case_names=case_names,
    )

    raw_system_prompts = getattr(agent, "_system_prompts", ())
    system_prompt_text = "\n\n".join(
        prompt for prompt in raw_system_prompts if isinstance(prompt, str) and prompt
    )

    def _wrap_text(value: str, *, indent: int, width: int = 120) -> str:
        if not value:
            return ""
        return textwrap.fill(
            value,
            width=width,
            initial_indent=" " * indent,
            subsequent_indent=" " * indent,
            break_long_words=False,
            break_on_hyphens=False,
        )

    def _to_compact_text(value: Any) -> str:
        jsonable = to_jsonable(value)
        if isinstance(jsonable, str):
            return jsonable
        # Compact but readable single-line representation before wrapping.
        return str(jsonable)

    def _event_lines(event: Any) -> list[str]:
        if isinstance(event, FunctionToolCallEvent):
            lines = [f"TOOL_CALL name={event.part.tool_name} id={event.part.tool_call_id or '-'}"]
            args_text = _to_compact_text(event.part.args)
            if args_text:
                wrapped_args = _wrap_text(f"args: {args_text}", indent=10)
                if wrapped_args:
                    lines = [f"{lines[0]}\n{wrapped_args}"]
            return lines

        if isinstance(event, FunctionToolResultEvent):
            lines = [f"TOOL_RESULT name={event.result.tool_name} id={event.result.tool_call_id or '-'}"]
            output_text = _to_compact_text(event.result.content)
            if output_text:
                wrapped_output = _wrap_text(output_text, indent=10)
                if wrapped_output:
                    lines = [f"{lines[0]}\n{wrapped_output}"]
            return lines

        if isinstance(event, FinalResultEvent):
            return [
                f"FINAL_RESULT tool={event.tool_name or '-'} id={event.tool_call_id or '-'}",
            ]

        return [f"EVENT {type(event).__name__}"]

    async def task(question: str) -> CanvasAnswer:
        case_name = question_to_case_name.get(question, "unknown_case")
        if system_prompt_text:
            wrapped_system_prompt = _wrap_text(system_prompt_text, indent=10)
            if wrapped_system_prompt:
                await interaction_logger.log_case(
                    case_name,
                    f"SYSTEM_PROMPT\n{wrapped_system_prompt}",
                )
        await interaction_logger.log_case(case_name, f"CASE_STARTED question={question}")
        payload = CanvasQuestionRequest(question=question, canvas_id=deps.canvas_id)
        result: Any | None = None
        try:
            async for event in agent.run_stream_events(
                user_prompt=build_prompt(payload),
                deps=deps,
            ):
                for event_line in _event_lines(event):
                    await interaction_logger.log_case(case_name, event_line)
                if isinstance(event, AgentRunResultEvent):
                    result = event.result
        except Exception as error:
            await interaction_logger.log_case(
                case_name,
                f"CASE_FAILED error={error}",
            )
            raise
        if result is None:
            raise RuntimeError(f"Eval case {case_name!r} did not produce a final result event.")

        run_usage = result.usage()
        async with usage_lock:
            if question in run_usages:
                raise RuntimeError(
                    "Duplicate eval case inputs are not supported for usage correlation "
                    f"(collision on {question[:120]!r}…)"
                )
            run_usages[question] = run_usage
        await interaction_logger.log_case(
            case_name,
            (
                "CASE_COMPLETED "
                f"tool_calls={run_usage.tool_calls} "
                f"input_tokens={run_usage.input_tokens} "
                f"output_tokens={run_usage.output_tokens}"
            ),
        )
        return result.output

    wall_start = time.perf_counter()
    try:
        report = await eval_dataset.evaluate(task, progress=True)
        evaluate_wall_seconds = time.perf_counter() - wall_start
        ReportBuilder(
            report,
            model=env["model"],
            run_usages=run_usages,
            evaluate_wall_seconds=evaluate_wall_seconds,
            interaction_log_paths_by_case_name=interaction_logger.display_paths_by_case_name,
        ).render()
    finally:
        interaction_logger.close()

def main() -> None:
    asyncio.run(runner())

def _format_elapsed(total_seconds: float) -> str:
    if total_seconds < 0:
        total_seconds = 0.0
    whole_seconds = int(total_seconds)
    millis = int(round((total_seconds - whole_seconds) * 1000))
    if millis == 1000:
        whole_seconds += 1
        millis = 0
    minutes, seconds = divmod(whole_seconds, 60)
    hours, minutes = divmod(minutes, 60)
    return f"{hours:02d}:{minutes:02d}:{seconds:02d}.{millis:03d}"


if __name__ == "__main__":
    main()
