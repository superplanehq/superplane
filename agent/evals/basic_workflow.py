"""Eval: manual-run trigger + two noops in a chain. See https://ai.pydantic.dev/evals/

Run: ``python -m evals.manual_run_two_noop_live`` (env for model/canvas id).
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

from pydantic_evals import Case, Dataset
from pydantic_evals.evaluators import Evaluator, EvaluatorContext
from pydantic_evals.reporting import EvaluationReport

from ai.agent import AgentDeps, build_agent
from ai.models import (
    AddNodeOperation,
    CanvasAnswer,
    CanvasOperationNodeRef,
    CanvasProposal,
    ConnectNodesOperation,
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

# ----

def list_nodes(ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> list[str]:
  return [op.node_key for op in ctx.output.proposal.operations if isinstance(op, AddNodeOperation)]

def count_nodes(ctx: EvaluatorContext[str, CanvasAnswer, Any], node_type: str) -> int:
  return len([op for op in ctx.output.proposal.operations if isinstance(op, node_type)])

def list_triggers(ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> list[str]:
  return [op.trigger_name for op in ctx.output.proposal.operations if isinstance(op, TriggerOperation)]

def assert_workflow_shape(ctx: EvaluatorContext[str, CanvasAnswer, Any], shape: str) -> bool:
  false