from dataclasses import dataclass
from typing import Any

from pydantic_evals.evaluators import EvaluationReason, Evaluator, EvaluatorContext

from ai.models import CanvasAnswer
from evals.evaluators.workflow_utils import process_operations

@dataclass
class CanvasHasTrigger(Evaluator):
  trigger: str

  def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> EvaluationReason:
    wf = process_operations(ctx.output.proposal.operations)
    
    if self.trigger in wf.nodes:
      return EvaluationReason(value=True, reason=f"Trigger {self.trigger} found in workflow")
    else:
      return EvaluationReason(value=False, reason=f"Trigger {self.trigger} not found in workflow")