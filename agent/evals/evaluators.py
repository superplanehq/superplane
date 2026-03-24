from dataclasses import dataclass
from pydantic_evals.evaluators import Evaluator, EvaluatorContext, EvaluationReason
from ai.models import CanvasAnswer, CanvasOperation
from typing import Any

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
      return EvaluationReason(value=False, reason=f"Workflow has {len(wf.nodes)} nodes, expected {len(self.nodes)}")

    # Check if the number of edges match
    if len(wf.edges) != len(self.edges):
      return EvaluationReason(value=False, reason=f"Workflow has {len(wf.edges)} edges, expected {len(self.edges)}")

    # Everything matches, return success
    return EvaluationReason(value=True, reason="Workflow shape matches")

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