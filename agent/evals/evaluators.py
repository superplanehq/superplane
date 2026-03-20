from dataclasses import dataclass
from pydantic_evals.evaluators import Evaluator, EvaluatorContext
from ai.models import CanvasAnswer, CanvasOperation
from typing import Any

@dataclass
class WorkflowShape(Evaluator):
  nodes: list[str]
  edges: list[tuple[str, str]]

  def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> bool:
    wf = process_operations(ctx.output.proposal.operations)
    return wf.nodes == self.nodes and wf.edges == self.edges

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
      nodes.append(op.node_name)

  for op in operations:
    if op.type == "connect_nodes":
      n1 = added_nodes[op.source.node_key]
      n2 = added_nodes[op.target.node_key]

      edges.append((n1.node_name, n2.node_name))

  return CanvasShape(nodes, edges)