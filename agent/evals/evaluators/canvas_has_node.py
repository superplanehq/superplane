from dataclasses import dataclass
from typing import Any

from pydantic_evals.evaluators import EvaluationReason, Evaluator, EvaluatorContext

from ai.models import CanvasAnswer
from evals.evaluators.workflow_utils import process_operations

@dataclass
class CanvasHasNode(Evaluator):
  node: str
  count: int = 1

  def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> EvaluationReason:
    wf = process_operations(ctx.output.proposal.operations)
    count = wf.nodes.count(self.node)

    if count == self.count:
      return EvaluationReason(value=True, reason=f"Node {self.node} found in workflow {count} times")
    elif count > self.count:
      return EvaluationReason(value=False, reason=f"Node {self.node} found in workflow {count} times, expected {self.count} times")
    elif count == 0:
      return EvaluationReason(value=False, reason=f"Node {self.node} not found in workflow")
    else:
      return EvaluationReason(value=False, reason=f"Node {self.node} found in workflow {count} times, expected {self.count} times")