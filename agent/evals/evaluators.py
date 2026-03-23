from dataclasses import dataclass
from typing import Any

from pydantic_evals.evaluators import EvaluationReason, Evaluator, EvaluatorContext

from ai.models import CanvasAnswer, CanvasOperation


@dataclass
class WorkflowShape(Evaluator):
    nodes: list[str]
    edges: list[tuple[str, str]]

    def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> EvaluationReason:
        proposal = ctx.output.proposal
        if proposal is None:
            return EvaluationReason(value=False, reason="Missing workflow proposal")

        wf = process_operations(proposal.operations)

        for node in self.nodes:
            if node not in wf.nodes:
                return EvaluationReason(value=False, reason=f"Node {node} not found in workflow")

        for edge in self.edges:
            if edge not in wf.edges:
                return EvaluationReason(value=False, reason=f"Edge {edge} not found in workflow")

        if len(wf.nodes) != len(self.nodes):
            return EvaluationReason(
                value=False,
                reason=f"Workflow has {len(wf.nodes)} nodes, expected {len(self.nodes)}",
            )

        if len(wf.edges) != len(self.edges):
            return EvaluationReason(
                value=False,
                reason=f"Workflow has {len(wf.edges)} edges, expected {len(self.edges)}",
            )

        return EvaluationReason(value=True, reason="Workflow shape matches")


@dataclass
class EphemeralMachineWorkflow(Evaluator):
    """Checks for an ephemeral preview environment orchestration pattern."""

    def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> EvaluationReason:
        proposal = ctx.output.proposal
        if proposal is None:
            return EvaluationReason(value=False, reason="Missing workflow proposal")

        workflow = process_operations(proposal.operations)
        normalized_nodes = [name.lower() for name in workflow.nodes]
        answer_text = f"{ctx.output.answer} {proposal.summary}".lower()

        has_github_pr_trigger = any(
            ("github" in name) and ("pr" in name or "pull" in name)
            for name in normalized_nodes
        )
        if not has_github_pr_trigger:
            return EvaluationReason(
                value=False,
                reason="Expected a GitHub PR trigger node for ephemeral preview orchestration",
            )

        http_like_count = sum(
            1
            for name in normalized_nodes
            if "http" in name or "request" in name or "api" in name
        )
        if http_like_count < 2:
            return EvaluationReason(
                value=False,
                reason="Expected create and teardown HTTP/API actions",
            )

        has_schedule = any(
            keyword in name
            for name in normalized_nodes
            for keyword in ("schedule", "cron", "delay", "timer", "wait")
        )
        if not has_schedule:
            return EvaluationReason(
                value=False,
                reason="Expected a scheduled timeout/cleanup step",
            )

        has_memory = any(
            keyword in name
            for name in normalized_nodes
            for keyword in ("memory", "cache", "state", "kv")
        )
        if not has_memory:
            return EvaluationReason(
                value=False,
                reason="Expected workflow state persistence for app-id/PR mapping",
            )

        has_cleanup_language = ("teardown" in answer_text or "delete" in answer_text) and (
            "close" in answer_text or "timeout" in answer_text
        )
        if not has_cleanup_language:
            return EvaluationReason(
                value=False,
                reason="Expected explanation of cleanup on PR close or timeout",
            )

        return EvaluationReason(value=True, reason="Ephemeral machine workflow pattern detected")


class CanvasShape:
    nodes: list[str]
    edges: list[tuple[str, str]]

    def __init__(self, nodes: list[str], edges: list[tuple[str, str]]):
        self.nodes = nodes
        self.edges = edges

    def __eq__(self, other: "CanvasShape") -> bool:
        return self.nodes == other.nodes and self.edges == other.edges

    def __str__(self) -> str:
        return f"CanvasShape(nodes={self.nodes}, edges={self.edges})"


def process_operations(operations: list[CanvasOperation]) -> CanvasShape:
    added_nodes: dict[str, CanvasOperation] = {}
    nodes: list[str] = []
    edges: list[tuple[str, str]] = []

    for operation in operations:
        if operation.type != "add_node":
            continue
        if operation.node_key is None:
            continue
        added_nodes[operation.node_key] = operation
        nodes.append(operation.block_name)

    for operation in operations:
        if operation.type != "connect_nodes":
            continue
        source_key = operation.source.node_key
        target_key = operation.target.node_key
        if source_key is None or target_key is None:
            continue
        source = added_nodes.get(source_key)
        target = added_nodes.get(target_key)
        if source is None or target is None:
            continue
        edges.append((source.block_name, target.block_name))

    return CanvasShape(nodes, edges)