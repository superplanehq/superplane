from __future__ import annotations

import asyncio
import os
import time
from datetime import UTC, datetime

from pydantic_ai.usage import RunUsage
from pydantic_evals import Dataset

from ai.agent import AgentDeps, build_agent
from ai.superplane_client import SuperplaneClient, SuperplaneClientConfig
from evals.case_filter import case_filter, select_cases
from evals.case_logger import CaseLogger
from evals.case_task import build_case_name_index, build_case_task, read_agent_system_prompt
from evals.cases import dataset
from evals.report import ReportBuilder
from evals.run_tool_registry import clear_tool_call_registry


def load_env() -> dict[str, str]:
    return {
        "model": os.getenv("AI_MODEL", "").strip(),
        "base_url": "http://app:8000",
        "api_token": os.getenv("SUPERPLANE_API_TOKEN", "").strip(),
        "organization_id": os.getenv("EVAL_ORG_ID", "").strip(),
        "canvas_id": os.getenv("EVAL_CANVAS_ID", "").strip(),
    }


async def runner(*, selected_case_names: list[str] | None) -> None:
    env = load_env()
    full_cases = list(dataset.cases)
    cases = select_cases(full_cases, selected_case_names)
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
    question_to_case_name, case_names = build_case_name_index(cases, full_cases)

    case_logger = CaseLogger(
        run_id=datetime.now(UTC).strftime("%Y%m%dT%H%M%S_%fZ"),
        case_names=case_names,
    )

    task = build_case_task(
        agent=agent,
        deps=deps,
        question_to_case_name=question_to_case_name,
        system_prompt_text=read_agent_system_prompt(agent),
        case_logger=case_logger,
        run_usages=run_usages,
        usage_lock=usage_lock,
    )

    wall_start = time.perf_counter()
    clear_tool_call_registry()
    try:
        report = await eval_dataset.evaluate(task, progress=True)
        evaluate_wall_seconds = time.perf_counter() - wall_start
        ReportBuilder(
            report,
            model=env["model"],
            run_usages=run_usages,
            evaluate_wall_seconds=evaluate_wall_seconds,
            case_names=case_names,
            interaction_log_paths_by_case_name=case_logger.display_paths_by_case_name,
        ).render()
    finally:
        clear_tool_call_registry()
        case_logger.close()


def main(argv: list[str] | None = None) -> None:
    selected_case_names = case_filter(argv)
    asyncio.run(runner(selected_case_names=selected_case_names))


if __name__ == "__main__":
    main()
