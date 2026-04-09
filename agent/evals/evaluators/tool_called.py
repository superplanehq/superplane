from dataclasses import dataclass
from typing import Any

from pydantic_evals.evaluators import EvaluationReason, Evaluator, EvaluatorContext

from ai.models import CanvasAnswer
from evals.run_tool_registry import count_tool_calls


@dataclass
class ToolCalled(Evaluator):
    """Assert the agent invoked a tool at least ``min_calls`` time(s) during the case run."""

    tool_name: str
    min_calls: int = 1

    def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> EvaluationReason:
        if self.min_calls < 1:
            return EvaluationReason(
                value=False,
                reason=f"Invalid min_calls={self.min_calls}; expected >= 1",
            )
        if not self.tool_name.strip():
            return EvaluationReason(value=False, reason="tool_name must be non-empty")

        observed = count_tool_calls(ctx.inputs, self.tool_name)
        if observed >= self.min_calls:
            return EvaluationReason(
                value=True,
                reason=f"{self.tool_name!r} called {observed} time(s) (min {self.min_calls})",
            )
        return EvaluationReason(
            value=False,
            reason=(
                f"{self.tool_name!r} called {observed} time(s), expected at least {self.min_calls}"
            ),
        )
