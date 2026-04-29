from dataclasses import dataclass
from typing import Any

from pydantic_evals.evaluators import EvaluationReason, Evaluator, EvaluatorContext

from ai.models import CanvasAnswer, CanvasChangeType


@dataclass
class CanvasEdgeUsesChannel(Evaluator):
    """Assert the changeset contains at least ``count`` edge(s) with the given channel name."""

    channel: str
    count: int = 1

    def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> EvaluationReason:
        if self.count < 1:
            return EvaluationReason(
                value=False,
                reason=f"Invalid count={self.count}; expected >= 1",
            )
        if not self.channel.strip():
            return EvaluationReason(value=False, reason="channel must be non-empty")

        proposal = ctx.output.proposal
        if proposal is None:
            return EvaluationReason(value=False, reason="no proposal in answer")

        changes = proposal.changeset.changes
        observed = sum(
            1
            for change in changes
            if change.type == CanvasChangeType.ADD_EDGE
            and change.edge is not None
            and change.edge.channel == self.channel
        )

        if observed >= self.count:
            return EvaluationReason(
                value=True,
                reason=f"channel {self.channel!r} used on {observed} edge(s) (min {self.count})",
            )
        return EvaluationReason(
            value=False,
            reason=f"channel {self.channel!r} found on {observed} edge(s), expected at least {self.count}",
        )
