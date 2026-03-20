from __future__ import annotations

import asyncio

from ai.agent import AgentDeps
from evals.basic_workflow import evaluate_manual_run_two_noop_live
from evals.stub_superplane_client import StubSuperplaneClient

async def _async_main(*, model: str, canvas_id: str) -> None:
    deps = AgentDeps(client=StubSuperplaneClient(), canvas_id=canvas_id)
    report = await evaluate_manual_run_two_noop_live(model=model, deps=deps, progress=True)
    report.print()

def main() -> None:
    asyncio.run(_async_main(model="gpt-4o-mini", canvas_id="eval-stub-canvas"))

if __name__ == "__main__":
    main()