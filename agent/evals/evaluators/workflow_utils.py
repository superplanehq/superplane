from collections.abc import Iterator
from typing import Any

from ai.models import CanvasChange, CanvasChangeType


class CanvasShape:
    nodes: list[str]
    edges: list[tuple[str, str]]

    def __init__(self, nodes: list[str], edges: list[tuple[str, str]]):
        self.nodes = nodes
        self.edges = edges

    def __eq__(self, other: object) -> bool:
        if not isinstance(other, CanvasShape):
            return NotImplemented
        return self.nodes == other.nodes and self.edges == other.edges

    def __str__(self) -> str:
        return f"CanvasShape(nodes={self.nodes}, edges={self.edges})"


def process_changes(changes: list[CanvasChange]) -> CanvasShape:
    added_nodes: dict[str, str] = {}
    nodes: list[str] = []
    edges: list[tuple[str, str]] = []

    for change in changes:
        if change.type != CanvasChangeType.ADD_NODE or change.node is None:
            continue

        node_id = change.node.id
        block = change.node.block
        if not isinstance(node_id, str) or not node_id:
            continue
        if not isinstance(block, str) or not block:
            continue

        added_nodes[node_id] = block
        nodes.append(block)

    for change in changes:
        if change.type != CanvasChangeType.ADD_EDGE or change.edge is None:
            continue

        source_id = change.edge.source_id
        target_id = change.edge.target_id
        if not isinstance(source_id, str) or not source_id:
            continue
        if not isinstance(target_id, str) or not target_id:
            continue

        n1 = added_nodes.get(source_id)
        n2 = added_nodes.get(target_id)
        if n1 is None or n2 is None:
            continue

        edges.append((n1, n2))

    return CanvasShape(nodes, edges)


def process_operations(changes: list[CanvasChange]) -> CanvasShape:
    return process_changes(changes)


def iter_config_strings_from_changes(
    changes: list[CanvasChange],
) -> Iterator[str]:
    for change in changes:
        if change.node is None:
            continue

        if change.type not in (CanvasChangeType.ADD_NODE, CanvasChangeType.UPDATE_NODE):
            continue

        if isinstance(change.node.configuration, dict):
            yield from _iter_strings_in_value(change.node.configuration)


def iter_config_strings_from_operations(
    changes: list[CanvasChange],
) -> Iterator[str]:
    yield from iter_config_strings_from_changes(changes)


def _iter_strings_in_value(value: Any) -> Iterator[str]:
    if isinstance(value, str):
        yield value
    elif isinstance(value, dict):
        for nested in value.values():
            yield from _iter_strings_in_value(nested)
    elif isinstance(value, list):
        for item in value:
            yield from _iter_strings_in_value(item)
