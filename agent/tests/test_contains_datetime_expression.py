from pydantic_evals.evaluators import EvaluatorContext
from pydantic_evals.otel._errors import SpanTreeRecordingError

from ai.models import CanvasAnswer, CanvasProposal
from evals.evaluators.contains_datetime_expression import ContainsDatetimeExpression


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


def _answer_with_expression(expression: str) -> CanvasAnswer:
    return CanvasAnswer(
        answer="Proposal ready.",
        confidence=0.9,
        proposal=CanvasProposal(
            summary="Add filter with datetime expression.",
            operations=[
                {  # type: ignore[list-item]
                    "type": "add_node",
                    "blockName": "filter",
                    "configuration": {"expression": expression},
                },
            ],
        ),
    )


def test_contains_datetime_expression_passes_date_subtraction() -> None:
    ev = ContainsDatetimeExpression()
    expr = 'date(a) - date(b) > duration("1h")'
    assert ev.evaluate(_ctx(_answer_with_expression(expr))).value is True


def test_contains_datetime_expression_passes_now_and_duration() -> None:
    ev = ContainsDatetimeExpression()
    expr = 'now() - date(x) < duration("1d")'
    assert ev.evaluate(_ctx(_answer_with_expression(expr))).value is True


def test_contains_datetime_expression_passes_date_and_hours() -> None:
    ev = ContainsDatetimeExpression()
    expr = "date(x).Hours() > 1"
    assert ev.evaluate(_ctx(_answer_with_expression(expr))).value is True


def test_contains_datetime_expression_passes_date_and_duration_without_subtraction() -> None:
    ev = ContainsDatetimeExpression()
    expr = 'date(x) > now() - duration("1h")'
    assert ev.evaluate(_ctx(_answer_with_expression(expr))).value is True


def test_contains_datetime_expression_passes_two_iso_dates_compared() -> None:
    ev = ContainsDatetimeExpression()
    expr = 'date("2024-01-01") > date("2024-02-01")'
    assert ev.evaluate(_ctx(_answer_with_expression(expr))).value is True


def test_contains_datetime_expression_passes_two_dates_equality() -> None:
    ev = ContainsDatetimeExpression()
    expr = "date(a) == date(b)"
    assert ev.evaluate(_ctx(_answer_with_expression(expr))).value is True


def test_contains_datetime_expression_fails_two_dates_without_infix_between() -> None:
    ev = ContainsDatetimeExpression()
    expr = "date(a) && date(b)"
    assert ev.evaluate(_ctx(_answer_with_expression(expr))).value is False


def test_contains_datetime_expression_passes_iso_strings_with_actual_subtraction() -> None:
    ev = ContainsDatetimeExpression()
    expr = 'date("2024-02-01") - date("2024-01-01") > duration("1h")'
    assert ev.evaluate(_ctx(_answer_with_expression(expr))).value is True


def test_contains_datetime_expression_fails_plain_expression() -> None:
    ev = ContainsDatetimeExpression()
    expr = 'payload.action == "closed"'
    assert ev.evaluate(_ctx(_answer_with_expression(expr))).value is False


def test_contains_datetime_expression_fails_single_date_only() -> None:
    ev = ContainsDatetimeExpression()
    expr = "date(x) != nil"
    assert ev.evaluate(_ctx(_answer_with_expression(expr))).value is False


def test_contains_datetime_expression_fails_three_arg_hint() -> None:
    ev = ContainsDatetimeExpression()
    expr = 'date(str, format, tz) > duration("1h")'
    assert ev.evaluate(_ctx(_answer_with_expression(expr))).value is False


def test_contains_datetime_expression_fails_no_proposal() -> None:
    ev = ContainsDatetimeExpression()
    answer = CanvasAnswer(answer="No proposal.", confidence=0.5, proposal=None)
    assert ev.evaluate(_ctx(answer)).value is False
