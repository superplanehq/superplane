"""Eval dataset: manual-run trigger + two noop components in a linear chain.

Follows the Pydantic Evals pattern (Dataset, Case, custom Evaluator); see
https://ai.pydantic.dev/evals/

Live agent example::

    async def task(prompt: str) -> CanvasAnswer:
        agent = build_agent(model="gpt-4o-mini")
        result = await agent.run(prompt, deps=deps)
        return result.output

    report = await build_manual_run_two_noop_dataset().evaluate(task)
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

from pydantic_evals import Case, Dataset
from pydantic_evals.evaluators import Evaluator, EvaluatorContext
from pydantic_evals.reporting import EvaluationReport

from ai.models import (
    AddNodeOperation,
    CanvasAnswer,
    CanvasOperationNodeRef,
    CanvasProposal,
    ConnectNodesOperation,
)

# Catalog block names (see pkg/triggers/start and pkg/components/noop).
MANUAL_RUN_TWO_NOOP_PROMPT = (
    "Build me a basic workflow that starts with a manual run, and runs two noop actions"
)

EXPECTED_TRIGGER_BLOCK = "start"
EXPECTED_NOOP_BLOCK = "noop"


def _ref_resolves_to_key(ref: CanvasOperationNodeRef, key_to_block: dict[str, str]) -> str | None:
    if ref.node_key is not None and ref.node_key in key_to_block:
        return ref.node_key
    if ref.node_id is not None and ref.node_id in key_to_block:
        return ref.node_id
    if ref.node_name is not None and ref.node_name in key_to_block:
        return ref.node_name
    return None


def _linear_chain_role_ok(
    key_to_block: dict[str, str],
    edge_pairs: list[tuple[str, str]],
) -> bool:
    keys = set(key_to_block)
    if len(keys) != 3 or len(edge_pairs) != 2:
        return False
    incoming = {k: 0 for k in keys}
    outgoing = {k: 0 for k in keys}
    for src, tgt in edge_pairs:
        if src not in keys or tgt not in keys:
            return False
        outgoing[src] += 1
        incoming[tgt] += 1
    chain_starts = [k for k in keys if incoming[k] == 0]
    chain_ends = [k for k in keys if outgoing[k] == 0]
    chain_mids = [k for k in keys if incoming[k] == 1 and outgoing[k] == 1]
    if len(chain_starts) != 1 or len(chain_ends) != 1 or len(chain_mids) != 1:
        return False
    if key_to_block[chain_starts[0]] != EXPECTED_TRIGGER_BLOCK:
        return False
    if key_to_block[chain_mids[0]] != EXPECTED_NOOP_BLOCK:
        return False
    if key_to_block[chain_ends[0]] != EXPECTED_NOOP_BLOCK:
        return False
    return True


def score_manual_run_two_noop_proposal(proposal: CanvasProposal) -> dict[str, float]:
    """Score how well a canvas proposal matches manual run → noop → noop.

    Returns sub-metrics in 0..1 plus ``combined`` (average).
    """
    ops = proposal.operations
    adds = [op for op in ops if isinstance(op, AddNodeOperation)]
    conns = [op for op in ops if isinstance(op, ConnectNodesOperation)]
    allowed = {AddNodeOperation, ConnectNodesOperation}
    only_nodes_and_edges = all(type(op) in allowed for op in ops)

    key_to_block: dict[str, str] = {}
    for i, op in enumerate(adds):
        key = op.node_key or f"__auto_{i}"
        if key in key_to_block:
            return {"nodes": 0.0, "connections": 0.0, "combined": 0.0}
        key_to_block[key] = op.block_name

    blocks = list(key_to_block.values())
    nodes_ok = (
        only_nodes_and_edges
        and len(adds) == 3
        and blocks.count(EXPECTED_TRIGGER_BLOCK) == 1
        and blocks.count(EXPECTED_NOOP_BLOCK) == 2
    )
    nodes_score = 1.0 if nodes_ok else 0.0

    conn_ok = False
    if len(conns) == 2 and len(key_to_block) == 3:
        resolved: list[tuple[str, str]] = []
        for conn in conns:
            src = _ref_resolves_to_key(conn.source, key_to_block)
            tgt = _ref_resolves_to_key(conn.target, key_to_block)
            if src is None or tgt is None:
                resolved = []
                break
            resolved.append((src, tgt))
        conn_ok = len(resolved) == 2 and _linear_chain_role_ok(key_to_block, resolved)

    connections_score = 1.0 if conn_ok else 0.0
    combined = (nodes_score + connections_score) / 2.0
    return {
        "nodes": nodes_score,
        "connections": connections_score,
        "combined": combined,
    }


@dataclass
class ManualRunTwoNoopTopologyEvaluator(Evaluator[str, CanvasAnswer, Any]):
    """Checks proposal ops: one ``start`` trigger, two ``noop`` components, linear edges."""

    evaluation_name = "manual_run_two_noop_topology"

    def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> dict[str, float]:
        proposal = ctx.output.proposal
        if proposal is None:
            return {"nodes": 0.0, "connections": 0.0, "combined": 0.0}
        return score_manual_run_two_noop_proposal(proposal)


def build_manual_run_two_noop_dataset() -> Dataset[str, CanvasAnswer, Any]:
    return Dataset(
        cases=[
            Case(
                name="manual_run_then_two_noops",
                inputs=MANUAL_RUN_TWO_NOOP_PROMPT,
            ),
        ],
        evaluators=[ManualRunTwoNoopTopologyEvaluator()],
    )


def run_manual_run_two_noop_experiment_sync(
    task_answer: CanvasAnswer,
) -> EvaluationReport[str, CanvasAnswer, Any]:
    """Run the dataset against a prebuilt answer (sync helper for tests or notebooks)."""

    def task(_prompt: str) -> CanvasAnswer:
        return task_answer

    return build_manual_run_two_noop_dataset().evaluate_sync(task)
