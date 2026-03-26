import re
from collections.abc import Iterator
from dataclasses import dataclass
from typing import Any, Literal

from pydantic_evals.evaluators import EvaluationReason, Evaluator, EvaluatorContext

from ai.models import AddNodeOperation, CanvasAnswer, CanvasOperation

_BRACKET_SINGLE_QUOTED = re.compile(r"\$\[\s*'([^']*)'\s*\]")
_BRACKET_DOUBLE_QUOTED = re.compile(r'\$\[\s*"([^"]*)"\s*\]')

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


@dataclass
class BracketSelectorsMatchCanvasNames(Evaluator):
    """$['…'] / $["…"] keys must match an add_node canvas label (nodeName or blockName)."""

    scan_scope: Literal["all", "last_add_node"] = "all"
    require_at_least_one_selector: bool = True
    target_block_name: str | None = None

    def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> EvaluationReason:
        if ctx.output.proposal is None:
            return EvaluationReason(value=False, reason="No proposal to check")

        ops = ctx.output.proposal.operations
        add_ops = [op for op in ops if op.type == "add_node"]
        if not add_ops:
            return EvaluationReason(value=False, reason="No add_node operations")

        allowed_names = _effective_canvas_labels(add_ops)
        texts = _collect_config_texts_for_bracket_scan(
            add_ops,
            scan_scope=self.scan_scope,
            target_block_name=self.target_block_name,
        )
        if not texts and self.scan_scope == "last_add_node":
            return EvaluationReason(value=False, reason="No configuration on last add_node to scan")

        combined = "\n".join(texts)
        keys = _extract_bracket_keys(combined)
        if self.require_at_least_one_selector and not keys:
            return EvaluationReason(
                value=False,
                reason="Expected $['…'] or $[\"…\"] selector in scanned configs",
            )

        for key in keys:
            if key not in allowed_names:
                return EvaluationReason(
                    value=False,
                    reason=(
                        f"Bracket key {key!r} is not a canvas node name "
                        f"(allowed: {sorted(allowed_names)})"
                    ),
                )

        return EvaluationReason(
            value=True,
            reason="All $['…'] / $[\"…\"] keys match canvas node names",
        )


def _effective_canvas_labels(add_ops: list[AddNodeOperation]) -> set[str]:
    labels: set[str] = set()
    for op in add_ops:
        name = (op.node_name or "").strip()
        labels.add(name if name else op.block_name)
    return labels


def _collect_config_texts_for_bracket_scan(
    add_ops: list[AddNodeOperation],
    *,
    scan_scope: Literal["all", "last_add_node"],
    target_block_name: str | None,
) -> list[str]:
    candidates: list[AddNodeOperation]
    if target_block_name is not None:
        candidates = [op for op in add_ops if op.block_name == target_block_name]
        if not candidates:
            return []
    elif scan_scope == "last_add_node":
        candidates = [add_ops[-1]]
    else:
        candidates = list(add_ops)

    texts: list[str] = []
    for op in candidates:
        texts.extend(_iter_strings_in_value(op.configuration))
    return texts


def _extract_bracket_keys(text: str) -> list[str]:
    keys = _BRACKET_SINGLE_QUOTED.findall(text) + _BRACKET_DOUBLE_QUOTED.findall(text)
    return [k for k in keys if k]


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