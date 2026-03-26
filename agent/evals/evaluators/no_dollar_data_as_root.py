from dataclasses import dataclass
from typing import Any

from pydantic_evals.evaluators import EvaluationReason, Evaluator, EvaluatorContext

from ai.models import CanvasAnswer
from evals.evaluators.workflow_utils import iter_config_strings_from_operations

_FORBIDDEN_DOLLAR_DATA = "$.data."

@dataclass
class NoDollarDataAsRoot(Evaluator):
    """Reject proposals that treat $.data. as run-start payload (use root().data...)."""

    def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> EvaluationReason:
        if ctx.output.proposal is None:
            return EvaluationReason(value=True, reason="No proposal to check")

        for text in iter_config_strings_from_operations(ctx.output.proposal.operations):
            if _FORBIDDEN_DOLLAR_DATA in text:
                snippet = text if len(text) <= 120 else text[:117] + "..."
                msg = "Forbidden $.data. in configuration; use root().data... for run-start fields"
                return EvaluationReason(
                    value=False,
                    reason=f"{msg}; example: {snippet!r}",
                )

        return EvaluationReason(value=True, reason="No forbidden $.data. in configuration")