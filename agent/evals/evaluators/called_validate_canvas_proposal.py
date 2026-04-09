from dataclasses import dataclass
from typing import Any

from pydantic_evals.evaluators import EvaluationReason, Evaluator, EvaluatorContext

from ai.models import CanvasAnswer
from evals.run_tool_registry import count_tool_calls

TOOL_NAME = "validate_canvas_proposal"


@dataclass
class CalledValidateCanvasProposal(Evaluator):
    """Requires the agent to call validate_canvas_proposal at least ``min_calls`` time(s)."""

    min_calls: int = 1

    def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> EvaluationReason:
        if self.min_calls < 1:
            return EvaluationReason(
                value=False,
                reason=f"Invalid min_calls={self.min_calls}; expected >= 1",
            )
        observed = count_tool_calls(ctx.inputs, TOOL_NAME)
        if observed >= self.min_calls:
            return EvaluationReason(
                value=True,
                reason=f"{TOOL_NAME} called {observed} time(s) (min {self.min_calls})",
            )
        return EvaluationReason(
            value=False,
            reason=(
                f"{TOOL_NAME} called {observed} time(s), expected at least {self.min_calls}"
            ),
        )
