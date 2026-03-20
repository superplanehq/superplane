from __future__ import annotations

import asyncio
import os
from dataclasses import dataclass
from typing import Any

from pydantic_evals import Case, Dataset

from ai.agent import AgentDeps, build_agent, build_prompt
from ai.models import CanvasAnswer, CanvasQuestionRequest
from evals.utils.stub_superplane_client import StubSuperplaneClient

from evals.evaluators import WorkflowShape

dataset = Dataset(
    cases=[
        Case(
            name="manual_run_then_two_noops",
            inputs="Build me a basic workflow that starts with a manual run and runs two noop actions",
            evaluators=[
                WorkflowShape(
                  nodes=["Manual Run", "Noop 1", "Noop 2"],
                  edges=[("Manual Run", "Noop 1"), ("Noop 1", "Noop 2")],
                )
            ],
        ),
    ],
)

async def runner(model: str) -> None:
    deps = AgentDeps(client=StubSuperplaneClient(), canvas_id="eval-stub-canvas")
    agent = build_agent(model)

    async def task(question: str) -> CanvasAnswer:
        payload = CanvasQuestionRequest(question=question, canvas_id=deps.canvas_id)
        result = await agent.run(build_prompt(payload), deps=deps)
        return result.output

    report = await dataset.evaluate(task, progress=True)
    report.print(include_output=True, include_input=True)

def main() -> None:
    model = os.getenv("AI_MODEL", "test")
    asyncio.run(runner(model))


if __name__ == "__main__":
    main()
