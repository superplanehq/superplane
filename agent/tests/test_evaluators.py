from typing import Any, cast

from ai.models import CanvasAnswer, CanvasProposal
from evals.evaluators import EphemeralMachineWorkflow


class _Context:
    def __init__(self, output: CanvasAnswer) -> None:
        self.output = output


def test_ephemeral_machine_workflow_passes_when_pattern_is_present() -> None:
    output = CanvasAnswer(
        answer="Create preview on PR open, then delete the app on PR close or timeout.",
        proposal=CanvasProposal(
            summary="PR preview environment with cleanup and state tracking.",
            operations=[
                {"type": "add_node", "blockName": "github.onPullRequest", "nodeKey": "pr"},
                {"type": "add_node", "blockName": "http.request", "nodeKey": "create"},
                {"type": "add_node", "blockName": "memory.upsert", "nodeKey": "memory_write"},
                {"type": "add_node", "blockName": "schedule.delay", "nodeKey": "timer"},
                {"type": "add_node", "blockName": "memory.read", "nodeKey": "memory_read"},
                {"type": "add_node", "blockName": "http.request", "nodeKey": "teardown"},
            ],
        ),
    )

    result = EphemeralMachineWorkflow().evaluate(cast(Any, _Context(output)))
    assert result.value is True, result.reason


def test_ephemeral_machine_workflow_fails_without_cleanup_semantics() -> None:
    output = CanvasAnswer(
        answer="Creates preview infrastructure from PR events.",
        proposal=CanvasProposal(
            summary="No teardown mentioned.",
            operations=[
                {"type": "add_node", "blockName": "github.onPullRequest", "nodeKey": "pr"},
                {"type": "add_node", "blockName": "http.request", "nodeKey": "create"},
                {"type": "add_node", "blockName": "schedule.delay", "nodeKey": "timer"},
                {"type": "add_node", "blockName": "memory.upsert", "nodeKey": "memory_write"},
                {"type": "add_node", "blockName": "http.request", "nodeKey": "notify"},
            ],
        ),
    )

    result = EphemeralMachineWorkflow().evaluate(cast(Any, _Context(output)))
    assert result.value is False
    assert result.reason == "Expected explanation of cleanup on PR close or timeout"
