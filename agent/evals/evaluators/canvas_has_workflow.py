from __future__ import annotations

from collections import deque
from typing import Any

from pydantic_evals.evaluators import EvaluationReason, Evaluator, EvaluatorContext

from ai.models import CanvasAnswer, CanvasChange, CanvasChangeType


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

        node_names, graph = _build_graph(ctx.output.proposal.changeset.changes or [])
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


def _build_graph(changes: list[CanvasChange]) -> tuple[dict[str, str], dict[str, set[str]]]:
    node_names: dict[str, str] = {}
    graph: dict[str, set[str]] = {}

    for change in changes:
        if change.type != CanvasChangeType.ADD_NODE or change.node is None:
            continue

        node_id = change.node.id
        block = change.node.block
        if not isinstance(node_id, str) or not node_id:
            continue
        if not isinstance(block, str) or not block:
            continue

        node_names[node_id] = block
        graph.setdefault(node_id, set())

    for change in changes:
        if change.type != CanvasChangeType.ADD_EDGE or change.edge is None:
            continue

        source_id = change.edge.source_id
        target_id = change.edge.target_id
        if (
            not isinstance(source_id, str)
            or not source_id
            or not isinstance(target_id, str)
            or not target_id
        ):
            continue
        if source_id not in node_names or target_id not in node_names:
            continue

        graph.setdefault(source_id, set()).add(target_id)

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
