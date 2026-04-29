from pydantic_evals.evaluators import EvaluatorContext
from pydantic_evals.otel._errors import SpanTreeRecordingError

from ai.models import CanvasAnswer, CanvasChangeset, CanvasProposal
from evals.evaluators.canvas_edge_uses_channel import CanvasEdgeUsesChannel


def _ctx(answer: CanvasAnswer) -> EvaluatorContext[str, CanvasAnswer, None]:
    return EvaluatorContext(
        name=None,
        inputs="",
        metadata=None,
        expected_output=None,
        output=answer,
        duration=0.0,
        _span_tree=SpanTreeRecordingError("test"),
        attributes={},
        metrics={},
    )


def _answer_with_edges(*channels: str | None) -> CanvasAnswer:
    changes: list[dict] = [
        {"type": "ADD_NODE", "node": {"id": "trigger_1", "name": "Trigger", "block": "start"}},
        {"type": "ADD_NODE", "node": {"id": "action_1", "name": "Action", "block": "noop"}},
    ]
    for channel in channels:
        edge: dict = {"type": "ADD_EDGE", "edge": {"sourceId": "trigger_1", "targetId": "action_1"}}
        if channel is not None:
            edge["edge"]["channel"] = channel
        changes.append(edge)

    return CanvasAnswer(
        answer="Proposal ready.",
        confidence=0.9,
        proposal=CanvasProposal(
            summary="Workflow with branching edges.",
            changeset=CanvasChangeset.model_validate({"changes": changes}),
        ),
    )


def test_passes_when_channel_present() -> None:
    ev = CanvasEdgeUsesChannel("true")
    assert ev.evaluate(_ctx(_answer_with_edges("true", "false"))).value is True


def test_fails_when_channel_absent() -> None:
    ev = CanvasEdgeUsesChannel("approved")
    assert ev.evaluate(_ctx(_answer_with_edges("true", "false"))).value is False


def test_fails_when_no_edges() -> None:
    ev = CanvasEdgeUsesChannel("true")
    answer = CanvasAnswer(
        answer="ok",
        confidence=0.5,
        proposal=CanvasProposal(
            summary="no edges",
            changeset=CanvasChangeset.model_validate({"changes": []}),
        ),
    )
    assert ev.evaluate(_ctx(answer)).value is False


def test_fails_when_no_proposal() -> None:
    ev = CanvasEdgeUsesChannel("true")
    answer = CanvasAnswer(answer="No proposal.", confidence=0.5, proposal=None)
    assert ev.evaluate(_ctx(answer)).value is False


def test_passes_min_count_exact() -> None:
    ev = CanvasEdgeUsesChannel("false", count=2)
    assert ev.evaluate(_ctx(_answer_with_edges("false", "false"))).value is True


def test_fails_min_count_not_met() -> None:
    ev = CanvasEdgeUsesChannel("false", count=2)
    assert ev.evaluate(_ctx(_answer_with_edges("false"))).value is False


def test_fails_invalid_count() -> None:
    ev = CanvasEdgeUsesChannel("true", count=0)
    assert ev.evaluate(_ctx(_answer_with_edges("true"))).value is False


def test_fails_empty_channel_name() -> None:
    ev = CanvasEdgeUsesChannel("   ")
    assert ev.evaluate(_ctx(_answer_with_edges("true"))).value is False


def test_edge_with_null_channel_does_not_match_default() -> None:
    ev = CanvasEdgeUsesChannel("default")
    # channel=None (field absent in JSON) is not equal to the string "default"
    assert ev.evaluate(_ctx(_answer_with_edges(None))).value is False
