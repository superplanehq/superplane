from __future__ import annotations

from typing import Any

from pydantic_evals.evaluators import EvaluatorContext

from ai.models import CanvasAnswer, AddNodeOperation, TriggerOperation

def list_nodes(ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> list[str]:
  return [op.node_key for op in ctx.output.proposal.operations if isinstance(op, AddNodeOperation)]

def count_nodes(ctx: EvaluatorContext[str, CanvasAnswer, Any], node_type: str) -> int:
  return len([op for op in ctx.output.proposal.operations if isinstance(op, node_type)])

def list_triggers(ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> list[str]:
  return [op.trigger_name for op in ctx.output.proposal.operations if isinstance(op, TriggerOperation)]

def assert_workflow_shape(ctx: EvaluatorContext[str, CanvasAnswer, Any], shape: str) -> bool:
  return False