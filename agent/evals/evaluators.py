from collections.abc import Iterator
from dataclasses import dataclass
from typing import Any

from pydantic_evals.evaluators import EvaluationReason, Evaluator, EvaluatorContext

from ai.models import CanvasAnswer, CanvasOperation

_FORBIDDEN_DOLLAR_DATA = "$.data."

@dataclass
class WorkflowShape(Evaluator):
  nodes: list[str]
  edges: list[tuple[str, str]]

  def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> EvaluationReason:
    wf = process_operations(ctx.output.proposal.operations)

    # Check if all nodes are present
    for node in self.nodes:
      if node not in wf.nodes:
        return EvaluationReason(value=False, reason=f"Node {node} not found in workflow")

    # Check if all edges are present
    for edge in wf.edges:
      if edge not in self.edges:
        return EvaluationReason(value=False, reason=f"Edge {edge} not found in workflow")

    # Check if the number of nodes and edges match
    if len(wf.nodes) != len(self.nodes):
      return EvaluationReason(
          value=False,
          reason=f"Workflow has {len(wf.nodes)} nodes, expected {len(self.nodes)}",
      )

    # Check if the number of edges match
    if len(wf.edges) != len(self.edges):
      return EvaluationReason(
          value=False,
          reason=f"Workflow has {len(wf.edges)} edges, expected {len(self.edges)}",
      )

    # Everything matches, return success
    return EvaluationReason(value=True, reason="Workflow shape matches")


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


def iter_config_strings_from_operations(
    operations: list[CanvasOperation],
) -> Iterator[str]:
    for op in operations:
        if op.type == "add_node":
            yield from _iter_strings_in_value(op.configuration)
        elif op.type == "update_node_config":
            yield from _iter_strings_in_value(op.configuration)


def _iter_strings_in_value(value: Any) -> Iterator[str]:
    if isinstance(value, str):
        yield value
    elif isinstance(value, dict):
        for nested in value.values():
            yield from _iter_strings_in_value(nested)
    elif isinstance(value, list):
        for item in value:
            yield from _iter_strings_in_value(item)


# Helper functions

class CanvasShape:
  nodes: list[str]
  edges: list[tuple[str, str]]

  def __init__(self, nodes: list[str], edges: list[tuple[str, str]]):
    self.nodes = nodes
    self.edges = edges

  def __eq__(self, other: 'CanvasShape') -> bool:
    return self.nodes == other.nodes and self.edges == other.edges

  def __str__(self) -> str:
    return f"CanvasShape(nodes={self.nodes}, edges={self.edges})"

def process_operations(operations: list[CanvasOperation]) -> CanvasShape:
  added_nodes: dict[str, CanvasOperation] = {}
  nodes: list[str] = []
  edges: list[tuple[str, str]] = []

  for op in operations:
    if op.type == "add_node":
      added_nodes[op.node_key] = op
      nodes.append(op.block_name)

  for op in operations:
    if op.type == "connect_nodes":
      n1 = added_nodes[op.source.node_key]
      n2 = added_nodes[op.target.node_key]

      edges.append((n1.block_name, n2.block_name))

  return CanvasShape(nodes, edges)