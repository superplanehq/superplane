from dataclasses import dataclass
from typing import Any

from pydantic_evals.evaluators import EvaluationReason, Evaluator, EvaluatorContext

from ai.models import CanvasAnswer
from evals.evaluators.workflow_utils import process_operations

@dataclass
class CanvasTotalNodeCount(Evaluator):
  count: int

  def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> EvaluationReason:
    wf = process_operations(ctx.output.proposal.operations)
    count = len(wf.nodes)

    if count == self.count:
      return EvaluationReason(value=True, reason=f"Workflow has {count} nodes, expected {self.count} nodes")
    elif count > self.count:
      return EvaluationReason(value=False, reason=f"Workflow has {count} nodes, expected {self.count} nodes")
    elif count == 0:
      return EvaluationReason(value=False, reason=f"Workflow has no nodes")
    else:
      return EvaluationReason(value=False, reason=f"Workflow has {count} nodes, expected {self.count} nodes")