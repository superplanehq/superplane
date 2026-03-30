from __future__ import annotations

import asyncio
import os
import time
from datetime import datetime, timezone

from pydantic_ai.usage import RunUsage
from pydantic_evals import Dataset

from ai.agent import AgentDeps, build_agent
from ai.superplane_client import SuperplaneClient, SuperplaneClientConfig
from evals.case_task import build_case_name_index, build_case_task, read_agent_system_prompt
from evals.case_logger import CaseLogger
from evals.cases import dataset
from evals.report import ReportBuilder


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
    question_to_case_name, case_names = build_case_name_index(cases)

    case_logger = CaseLogger(
        run_id=datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%S_%fZ"),
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
    try:
        report = await eval_dataset.evaluate(task, progress=True)
        evaluate_wall_seconds = time.perf_counter() - wall_start
        ReportBuilder(
            report,
            model=env["model"],
            run_usages=run_usages,
            evaluate_wall_seconds=evaluate_wall_seconds,
            interaction_log_paths_by_case_name=case_logger.display_paths_by_case_name,
        ).render()
    finally:
        case_logger.close()

def main() -> None:
    asyncio.run(runner())

if __name__ == "__main__":
    main()
