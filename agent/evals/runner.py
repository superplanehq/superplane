from __future__ import annotations

import argparse
import asyncio
import os
import sys
import time
from collections.abc import Collection, Sequence
from datetime import UTC, datetime
from typing import Any

from pydantic_ai.usage import RunUsage
from pydantic_evals import Dataset

from ai.agent import AgentDeps, build_agent
from ai.superplane_client import SuperplaneClient, SuperplaneClientConfig
from evals.case_logger import CaseLogger
from evals.case_task import build_case_name_index, build_case_task, read_agent_system_prompt
from evals.cases import dataset
from evals.report import ReportBuilder


def _case_name(case: Any, index: int) -> str:
    return getattr(case, "name", f"case_{index}")


def split_case_names(value: str | None) -> list[str] | None:
    if value is None:
        return None
    names = [part.strip() for part in value.split(",")]
    names = [n for n in names if n]
    return names if names else None


def select_cases(all_cases: Sequence[Any], selected: Collection[str] | None) -> list[Any]:
    if not selected:
        return list(all_cases)
    wanted = frozenset(selected)
    known = {_case_name(c, i) for i, c in enumerate(all_cases)}
    unknown = sorted(wanted - known)
    if unknown:
        available = "\n  ".join(sorted(known))
        sys.stderr.write(
            f"Unknown eval case name(s): {', '.join(unknown)}\nValid names:\n  {available}\n"
        )
        raise SystemExit(2)
    return [c for i, c in enumerate(all_cases) if _case_name(c, i) in wanted]


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
    cases = select_cases(list(dataset.cases), selected_case_names)
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


def _parse_args(argv: list[str] | None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run SuperPlane canvas agent evals.")
    parser.add_argument(
        "--cases",
        metavar="NAMES",
        help="Comma-separated eval case names; overrides CASES when set.",
    )
    parser.add_argument(
        "--list-cases",
        action="store_true",
        help="Print eval case names and exit.",
    )
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> None:
    args = _parse_args(argv)
    if args.list_cases:
        for index, case in enumerate(dataset.cases):
            print(_case_name(case, index))
        return
    selected = split_case_names(args.cases)
    if selected is None:
        selected = split_case_names(os.getenv("CASES"))
    asyncio.run(runner(selected_case_names=selected))


if __name__ == "__main__":
    main()
