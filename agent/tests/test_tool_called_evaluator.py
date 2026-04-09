from pydantic_evals.evaluators import EvaluatorContext
from pydantic_evals.otel._errors import SpanTreeRecordingError

from ai.models import CanvasAnswer, CanvasProposal
from evals.evaluators.tool_called import ToolCalled
from evals.run_tool_registry import clear_tool_call_registry, record_tool_call


def _ctx(question: str, answer: CanvasAnswer) -> EvaluatorContext[str, CanvasAnswer, None]:
    return EvaluatorContext(
        name=None,
        inputs=question,
        metadata=None,
        expected_output=None,
        output=answer,
        duration=0.0,
        _span_tree=SpanTreeRecordingError("test"),
        attributes={},
        metrics={},
    )


def test_tool_called_passes_when_tool_invoked() -> None:
    clear_tool_call_registry()
    try:
        q = "Build a workflow"
        record_tool_call(q, "get_canvas")
        record_tool_call(q, "validate_proposal")
        answer = CanvasAnswer(
            answer="ok",
            confidence=0.5,
            proposal=CanvasProposal(summary="s", operations=[]),
        )
        result = ToolCalled("validate_proposal").evaluate(_ctx(q, answer))
        assert result.value is True
    finally:
        clear_tool_call_registry()


def test_tool_called_fails_when_tool_missing() -> None:
    clear_tool_call_registry()
    try:
        q = "Build a workflow"
        record_tool_call(q, "get_canvas")
        answer = CanvasAnswer(answer="ok", confidence=0.5, proposal=None)
        result = ToolCalled("validate_proposal").evaluate(_ctx(q, answer))
        assert result.value is False
    finally:
        clear_tool_call_registry()


def test_tool_called_min_calls_two() -> None:
    clear_tool_call_registry()
    try:
        q = "Build a workflow"
        record_tool_call(q, "validate_proposal")
        answer = CanvasAnswer(answer="ok", confidence=0.5, proposal=None)
        result = ToolCalled("validate_proposal", min_calls=2).evaluate(_ctx(q, answer))
        assert result.value is False
        record_tool_call(q, "validate_proposal")
        result = ToolCalled("validate_proposal", min_calls=2).evaluate(_ctx(q, answer))
        assert result.value is True
    finally:
        clear_tool_call_registry()
