from ai.evals.basic_workflow import score_manual_run_two_noop_proposal
from ai.models import (
    AddNodeOperation,
    CanvasOperationNodeRef,
    CanvasProposal,
    ConnectNodesOperation,
)


def _good_proposal() -> CanvasProposal:
    return CanvasProposal(
        summary="Manual run, then two no-ops in series.",
        operations=[
            AddNodeOperation(type="add_node", block_name="start", node_key="t"),
            AddNodeOperation(type="add_node", block_name="noop", node_key="n1"),
            AddNodeOperation(type="add_node", block_name="noop", node_key="n2"),
            ConnectNodesOperation(
                type="connect_nodes",
                source=CanvasOperationNodeRef(node_key="t"),
                target=CanvasOperationNodeRef(node_key="n1"),
            ),
            ConnectNodesOperation(
                type="connect_nodes",
                source=CanvasOperationNodeRef(node_key="n1"),
                target=CanvasOperationNodeRef(node_key="n2"),
            ),
        ],
    )


def test_score_perfect_manual_run_two_noop() -> None:
    scores = score_manual_run_two_noop_proposal(_good_proposal())
    assert scores["nodes"] == 1.0
    assert scores["connections"] == 1.0
    assert scores["combined"] == 1.0


def test_score_reversed_chain_fails_connections() -> None:
    bad = CanvasProposal(
        summary="Wrong edge direction / topology.",
        operations=[
            AddNodeOperation(type="add_node", block_name="start", node_key="t"),
            AddNodeOperation(type="add_node", block_name="noop", node_key="n1"),
            AddNodeOperation(type="add_node", block_name="noop", node_key="n2"),
            ConnectNodesOperation(
                type="connect_nodes",
                source=CanvasOperationNodeRef(node_key="n1"),
                target=CanvasOperationNodeRef(node_key="t"),
            ),
            ConnectNodesOperation(
                type="connect_nodes",
                source=CanvasOperationNodeRef(node_key="n2"),
                target=CanvasOperationNodeRef(node_key="n1"),
            ),
        ],
    )
    scores = score_manual_run_two_noop_proposal(bad)
    assert scores["nodes"] == 1.0
    assert scores["connections"] == 0.0


def test_score_wrong_block_counts() -> None:
    bad = CanvasProposal(
        summary="Only one noop.",
        operations=[
            AddNodeOperation(type="add_node", block_name="start", node_key="t"),
            AddNodeOperation(type="add_node", block_name="noop", node_key="n1"),
            ConnectNodesOperation(
                type="connect_nodes",
                source=CanvasOperationNodeRef(node_key="t"),
                target=CanvasOperationNodeRef(node_key="n1"),
            ),
        ],
    )
    scores = score_manual_run_two_noop_proposal(bad)
    assert scores["nodes"] == 0.0
