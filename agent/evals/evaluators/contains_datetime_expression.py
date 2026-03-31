import re
from dataclasses import dataclass
from typing import Any

from pydantic_evals.evaluators import EvaluationReason, Evaluator, EvaluatorContext

from ai.models import CanvasAnswer
from evals.evaluators.workflow_utils import iter_config_strings_from_operations

_DATE_CALL_RE = re.compile(r"\bdate\s*\(")
_NOW_CALL_RE = re.compile(r"\bnow\s*\(")
_THREE_ARG_HINT = "date(str, format"


def _date_call_positions(s: str) -> list[int]:
    return [m.start() for m in _DATE_CALL_RE.finditer(s)]


def _has_hyphen_between_first_two_date_calls(s: str) -> bool:
    positions = _date_call_positions(s)
    if len(positions) < 2:
        return False
    first, second = positions[0], positions[1]
    hyphen = s.find("-", first)
    return first < hyphen < second


def _has_duration_method(s: str) -> bool:
    return ".Hours()" in s or ".Minutes()" in s or ".Seconds()" in s


@dataclass
class ContainsDatetimeExpression(Evaluator):
    """Proposal configuration should include expr-lang datetime usage (date, duration, now, or duration methods)."""

    def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> EvaluationReason:
        if ctx.output.proposal is None:
            return EvaluationReason(value=False, reason="No proposal to check")

        texts = list(iter_config_strings_from_operations(ctx.output.proposal.operations))
        if not texts:
            return EvaluationReason(value=False, reason="No configuration strings in proposal operations")

        combined = "\n".join(texts)

        if _THREE_ARG_HINT in combined:
            return EvaluationReason(
                value=False,
                reason="Three-argument date(str, format, ...) style is not supported in SuperPlane expressions",
            )

        date_count = len(_DATE_CALL_RE.findall(combined))
        has_date_and_duration = "duration(" in combined and _DATE_CALL_RE.search(combined) is not None
        has_now_and_duration = (
            _NOW_CALL_RE.search(combined) is not None and "duration(" in combined
        )
        has_date_subtraction = date_count >= 2 and any(
            _has_hyphen_between_first_two_date_calls(t) for t in texts
        )
        has_date_and_duration_method = _DATE_CALL_RE.search(combined) is not None and any(
            _has_duration_method(t) for t in texts
        )

        if (
            has_date_subtraction
            or has_date_and_duration
            or has_now_and_duration
            or has_date_and_duration_method
        ):
            return EvaluationReason(
                value=True,
                reason="Found expr-lang datetime patterns (date/duration/now or duration methods)",
            )

        return EvaluationReason(
            value=False,
            reason=(
                "Expected datetime expression patterns: two date() with subtraction, or date()+duration(), "
                "or now()+duration(), or date() with .Hours/.Minutes/.Seconds"
            ),
        )
