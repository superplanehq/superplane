"""Helpers for Pydantic Evals on ``CanvasAnswer`` proposals."""

from __future__ import annotations

from collections import Counter
from typing import Any

from pydantic_evals.evaluators import EvaluatorContext

from ai.models import (
    AddNodeOperation,
    CanvasAnswer,
    CanvasOperationNodeRef,
    CanvasProposal,
    ConnectNodesOperation,
)

# Trigger block names we treat as triggers in the stub catalog (extend if evals add triggers).
_TRIGGER_BLOCKS = frozenset({"start"})


def _proposal(ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> CanvasProposal | None:
    return ctx.output.proposal


def list_triggers(ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> list[str]:
    p = _proposal(ctx)
    if p is None:
        return []
    return [
        op.block_name
        for op in p.operations
        if isinstance(op, AddNodeOperation) and op.block_name in _TRIGGER_BLOCKS
    ]


def count_nodes(ctx: EvaluatorContext[str, CanvasAnswer, Any], block_name: str) -> int:
    p = _proposal(ctx)
    if p is None:
        return 0
    return sum(
        1 for op in p.operations if isinstance(op, AddNodeOperation) and op.block_name == block_name
    )


def _ref_key(ref: CanvasOperationNodeRef, key_to_block: dict[str, str]) -> str | None:
    if ref.node_key is not None and ref.node_key in key_to_block:
        return ref.node_key
    if ref.node_id is not None and ref.node_id in key_to_block:
        return ref.node_id
    if ref.node_name is not None and ref.node_name in key_to_block:
        return ref.node_name
    return None


def _linear_path_matches(
    key_to_block: dict[str, str],
    edges: list[tuple[str, str]],
    expected_blocks: list[str],
) -> bool:
    keys = set(key_to_block)
    if len(keys) != len(expected_blocks) or len(edges) != len(expected_blocks) - 1:
        return False
    inc = {k: 0 for k in keys}
    out = {k: 0 for k in keys}
    for s, t in edges:
        if s not in keys or t not in keys:
            return False
        out[s] += 1
        inc[t] += 1
    heads = [k for k in keys if inc[k] == 0]
    tails = [k for k in keys if out[k] == 0]
    if len(heads) != 1 or len(tails) != 1:
        return False
    cur = heads[0]
    seen: list[str] = []
    visited: set[str] = set()
    while cur not in visited:
        visited.add(cur)
        seen.append(key_to_block[cur])
        if cur == tails[0]:
            break
        outs = [t for s, t in edges if s == cur]
        if len(outs) != 1:
            visited.add("__fail")
            break
        cur = outs[0]
    return visited == keys and seen == expected_blocks


def assert_workflow_shape(ctx: EvaluatorContext[str, CanvasAnswer, Any], shape: str) -> bool:
    """Require a single node chain whose blocks match ``shape`` (e.g. ``a -> b -> c``)."""
    expected = [p.strip() for p in shape.split("->")]
    if not expected or any(not p for p in expected):
        return False
    p = _proposal(ctx)
    if p is None:
        return False
    ops = p.operations
    adds = [op for op in ops if isinstance(op, AddNodeOperation)]
    conns = [op for op in ops if isinstance(op, ConnectNodesOperation)]
    allowed = {AddNodeOperation, ConnectNodesOperation}
    if not all(type(op) in allowed for op in ops):
        return False

    key_to_block: dict[str, str] = {}
    for i, op in enumerate(adds):
        key = op.node_key or f"__auto_{i}"
        if key in key_to_block:
            return False
        key_to_block[key] = op.block_name

    if len(adds) != len(expected):
        return False
    if Counter(key_to_block.values()) != Counter(expected):
        return False
    resolved: list[tuple[str, str]] = []
    for c in conns:
        s, t = _ref_key(c.source, key_to_block), _ref_key(c.target, key_to_block)
        if s is None or t is None:
            return False
        resolved.append((s, t))
    return _linear_path_matches(key_to_block, resolved, expected)
