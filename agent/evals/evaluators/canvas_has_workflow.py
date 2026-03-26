from __future__ import annotations

from collections import deque
from typing import Any

from pydantic_evals.evaluators import EvaluationReason, Evaluator, EvaluatorContext

from ai.models import CanvasAnswer


class CanvasHasWorkflow(Evaluator):
    """Verify that a directed workflow path matches the provided sequence.

    Use "..." to allow zero or more intermediate nodes between two steps.
    Example:
      CanvasHasWorkflow("github.onPullRequest", "...", "wait", "...", "daytona.deleteSandbox")
    """

    def __init__(self, *steps: str):
        self.steps = steps

    def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> EvaluationReason:
        if ctx.output.proposal is None:
            return EvaluationReason(value=False, reason="No proposal to check")

        node_names, graph = _build_graph(ctx.output.proposal.operations)
        if not node_names:
            return EvaluationReason(value=False, reason="No add_node operations")

        normalized_steps = _normalize_steps(self.steps)
        if not normalized_steps:
            return EvaluationReason(value=False, reason="Workflow sequence is empty")

        first_name = normalized_steps[0]
        starting_nodes = [node_key for node_key, name in node_names.items() if name == first_name]
        if not starting_nodes:
            return EvaluationReason(
                value=False,
                reason=f"Workflow start node {first_name!r} not found",
            )

        cache: dict[tuple[str, int], bool] = {}
        for node_key in starting_nodes:
            if _matches_from(node_key, 0, normalized_steps, node_names, graph, cache):
                return EvaluationReason(
                    value=True,
                    reason=f"Workflow path found for sequence: {normalized_steps}",
                )

        return EvaluationReason(
            value=False,
            reason=f"No connected path matches sequence: {normalized_steps}",
        )


def _build_graph(operations: list[Any]) -> tuple[dict[str, str], dict[str, set[str]]]:
    node_names: dict[str, str] = {}
    graph: dict[str, set[str]] = {}
    add_node_ops: list[Any] = []

    for op in operations:
        if op.type != "add_node":
            continue
        node_names[op.node_key] = op.block_name
        graph.setdefault(op.node_key, set())
        add_node_ops.append(op)

    # Most proposals encode connectivity via add_node.source rather than connect_nodes.
    for op in add_node_ops:
        if op.source is None:
            continue
        source_key = op.source.node_key
        target_key = op.node_key
        if source_key in node_names and target_key in node_names:
            graph.setdefault(source_key, set()).add(target_key)

    for op in operations:
        if op.type != "connect_nodes":
            continue

        source_key = op.source.node_key
        target_key = op.target.node_key
        if source_key not in node_names or target_key not in node_names:
            continue

        graph.setdefault(source_key, set()).add(target_key)

    return node_names, graph


def _normalize_steps(steps: tuple[str, ...]) -> list[str]:
    cleaned = [step.strip() for step in steps if step.strip()]
    if not cleaned:
        return []

    normalized: list[str] = []
    for step in cleaned:
        if step == "..." and normalized and normalized[-1] == "...":
            continue
        normalized.append(step)

    while normalized and normalized[0] == "...":
        normalized.pop(0)
    while normalized and normalized[-1] == "...":
        normalized.pop()

    return normalized


def _matches_from(
    current_node: str,
    step_index: int,
    steps: list[str],
    node_names: dict[str, str],
    graph: dict[str, set[str]],
    cache: dict[tuple[str, int], bool],
) -> bool:
    cache_key = (current_node, step_index)
    if cache_key in cache:
        return cache[cache_key]

    if node_names[current_node] != steps[step_index]:
        cache[cache_key] = False
        return False

    if step_index == len(steps) - 1:
        cache[cache_key] = True
        return True

    next_step = steps[step_index + 1]

    if next_step == "...":
        target_name = steps[step_index + 2]
        for reachable in _reachable_named_nodes(current_node, target_name, node_names, graph):
            if _matches_from(reachable, step_index + 2, steps, node_names, graph, cache):
                cache[cache_key] = True
                return True
        cache[cache_key] = False
        return False

    for neighbor in graph.get(current_node, set()):
        if node_names[neighbor] != next_step:
            continue
        if _matches_from(neighbor, step_index + 1, steps, node_names, graph, cache):
            cache[cache_key] = True
            return True

    cache[cache_key] = False
    return False


def _reachable_named_nodes(
    start: str,
    target_name: str,
    node_names: dict[str, str],
    graph: dict[str, set[str]],
) -> list[str]:
    queue: deque[str] = deque(graph.get(start, set()))
    visited: set[str] = set()
    matches: list[str] = []

    while queue:
        node = queue.popleft()
        if node in visited:
            continue
        visited.add(node)

        if node_names[node] == target_name:
            matches.append(node)

        for next_node in graph.get(node, set()):
            if next_node not in visited:
                queue.append(next_node)

    return matches
