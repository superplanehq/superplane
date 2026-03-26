from typing import Any, Iterator

from ai.models import CanvasOperation

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