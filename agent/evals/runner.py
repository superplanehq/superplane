from __future__ import annotations

import asyncio

from ai.agent import AgentDeps
from pydantic_evals import Dataset, Case, Evaluator
from pydantic_evals.evaluators import Evaluator
from pydantic_evals.reporting import EvaluationReport
from evals.stub_superplane_client import StubSuperplaneClient

from evals.utils.workflow_assertions import (
  list_nodes,
  count_nodes, 
  list_triggers, 
  assert_workflow_shape
)

dataset = Dataset(
    cases=[
        Case(
          name="manual_run_then_two_noops", 
          inputs="Build me a basic workflow that starts with a manual run, and runs two noop actions"
        ),
    ],
    evaluators=[
      Evaluator(name="StartsWithManualRun", evaluate=lambda ctx: list_triggers(ctx)[0] == "start"),
      Evaluator(name="HasTwoNoops", evaluate=lambda ctx: count_nodes(ctx, "noop") == 2)
      Evaluator(name="Linear", evaluate=lambda ctx: assert_workflow_shape(ctx, "start -> noop -> noop"),
    ],
)

def main() -> None:
    model = os.getenv("AI_MODEL")
    asyncio.run(runner(model))

async def runner(model: str) -> None:
    deps = AgentDeps(client=StubSuperplaneClient(), canvas_id="eval-stub-canvas")

    report = await evaluate_dataset(
      model=model, 
      deps=deps, 
      progress=True, 
      dataset=dataset
    )

    report.print()

if __name__ == "__main__":
    main()