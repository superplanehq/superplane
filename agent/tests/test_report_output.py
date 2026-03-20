import asyncio
from io import StringIO

from ai.evals.basic_workflow import build_manual_run_two_noop_dataset
from ai.evals.report_output import print_eval_report_plain
from ai.models import (
    AddNodeOperation,
    CanvasAnswer,
    CanvasOperationNodeRef,
    CanvasProposal,
    ConnectNodesOperation,
)


def test_print_eval_report_plain_includes_scores() -> None:
    proposal = CanvasProposal(
        summary="ok",
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

    async def task(prompt: str) -> CanvasAnswer:
        return CanvasAnswer(answer="y", proposal=proposal)

    report = asyncio.run(
        build_manual_run_two_noop_dataset().evaluate(task, progress=False),
    )
    buf = StringIO()
    print_eval_report_plain(report, file=buf, include_input=False, include_durations=False)
    text = buf.getvalue()
    assert "combined" in text
    assert "1" in text
    assert "manual_run_then_two_noops" in text
