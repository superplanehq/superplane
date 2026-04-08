from __future__ import annotations

import asyncio
import textwrap
from typing import Any, cast

from pydantic_ai.messages import (
    FinalResultEvent,
    FunctionToolCallEvent,
    FunctionToolResultEvent,
)
from pydantic_ai.run import AgentRunResultEvent
from pydantic_ai.usage import RunUsage

from ai.agent import build_prompt
from ai.jsonutil import to_jsonable
from ai.models import CanvasAnswer, CanvasQuestionRequest
from evals.case_logger import CaseLogger
from evals.case_names import eval_case_name


def build_case_name_index(
    selected_cases: list[Any],
    full_dataset: list[Any],
) -> tuple[dict[str, str], list[str]]:
    index_by_id = {id(c): i for i, c in enumerate(full_dataset)}
    question_to_case_name: dict[str, str] = {}
    case_names: list[str] = []
    for case in selected_cases:
        idx = index_by_id.get(id(case))
        if idx is None:
            raise RuntimeError(
                "Eval case not in full_dataset; pass the same Case instances as in cases.py."
            )
        if not isinstance(case.inputs, str):
            raise RuntimeError(
                f"Case {eval_case_name(case, idx)!r} has non-string input; "
                "eval logging requires string case inputs."
            )
        if case.inputs in question_to_case_name:
            raise RuntimeError(
                "Duplicate eval case inputs are not supported for usage correlation "
                f"(collision on {case.inputs[:120]!r}...)"
            )
        case_name = eval_case_name(case, idx)
        question_to_case_name[case.inputs] = case_name
        case_names.append(case_name)
    return question_to_case_name, case_names


def read_agent_system_prompt(agent: Any) -> str:
    raw_system_prompts = getattr(agent, "_system_prompts", ())
    return "\n\n".join(
        prompt for prompt in raw_system_prompts if isinstance(prompt, str) and prompt
    )


def build_case_task(
    *,
    agent: Any,
    deps: Any,
    question_to_case_name: dict[str, str],
    system_prompt_text: str,
    case_logger: CaseLogger,
    run_usages: dict[str, RunUsage],
    usage_lock: asyncio.Lock,
) -> Any:
    async def task(question: str) -> CanvasAnswer:
        case_name = question_to_case_name.get(question, "unknown_case")
        if system_prompt_text:
            wrapped_system_prompt = _wrap_text(system_prompt_text, indent=10)
            if wrapped_system_prompt:
                await case_logger.log_case(case_name, f"SYSTEM_PROMPT\n{wrapped_system_prompt}")
        await case_logger.log_case(case_name, f"CASE_STARTED question={question}")
        payload = CanvasQuestionRequest(question=question, canvas_id=deps.canvas_id)
        result: Any | None = None
        try:
            async for event in agent.run_stream_events(
                user_prompt=build_prompt(payload),
                deps=deps,
            ):
                for event_line in _event_lines(event):
                    await case_logger.log_case(case_name, event_line)
                if isinstance(event, AgentRunResultEvent):
                    result = event.result
        except Exception as error:
            await case_logger.log_case(case_name, f"CASE_FAILED error={error}")
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
        await case_logger.log_case(
            case_name,
            (
                "CASE_COMPLETED "
                f"tool_calls={run_usage.tool_calls} "
                f"input_tokens={run_usage.input_tokens} "
                f"output_tokens={run_usage.output_tokens}"
            ),
        )
        return cast(CanvasAnswer, result.output)

    return task


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
    return str(jsonable)


def _event_lines(event: Any) -> list[str]:
    if isinstance(event, FunctionToolCallEvent):
        line = f"TOOL_CALL name={event.part.tool_name} id={event.part.tool_call_id or '-'}"
        args_text = _to_compact_text(event.part.args)
        if args_text:
            wrapped_args = _wrap_text(f"args: {args_text}", indent=10)
            if wrapped_args:
                return [f"{line}\n{wrapped_args}"]
        return [line]

    if isinstance(event, FunctionToolResultEvent):
        line = f"TOOL_RESULT name={event.result.tool_name} id={event.result.tool_call_id or '-'}"
        output_text = _to_compact_text(event.result.content)
        if output_text:
            wrapped_output = _wrap_text(output_text, indent=10)
            if wrapped_output:
                return [f"{line}\n{wrapped_output}"]
        return [line]

    if isinstance(event, FinalResultEvent):
        return [f"FINAL_RESULT tool={event.tool_name or '-'} id={event.tool_call_id or '-'}"]

    return [f"EVENT {type(event).__name__}"]
