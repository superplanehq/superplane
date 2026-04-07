import re
from dataclasses import dataclass
from typing import Any

from pydantic_evals.evaluators import EvaluationReason, Evaluator, EvaluatorContext

from ai.models import CanvasAnswer
from evals.evaluators.workflow_utils import iter_config_strings_from_operations

_DATE_CALL_RE = re.compile(r"\bdate\s*\(")
_NOW_CALL_RE = re.compile(r"\bnow\s*\(")
_THREE_ARG_HINT = "date(str, format"

_DATE_DATE_INFIX_OPS = ("<=", ">=", "==", "!=", "<", ">", "-")


def _date_call_positions(s: str) -> list[int]:
    return [m.start() for m in _DATE_CALL_RE.finditer(s)]


def _date_open_paren_index(s: str, date_keyword_start: int) -> int | None:
    """Index of '(' starting the date(...) call; date_keyword_start is the 'd' in `date`."""
    i = date_keyword_start + 4
    n = len(s)
    while i < n and s[i].isspace():
        i += 1
    if i < n and s[i] == "(":
        return i
    return None


def _matching_close_paren(s: str, open_paren: int) -> int | None:
    """Index of ')' that closes the '(' at open_paren.

    Uses paren depth (strings not special-cased).
    """
    depth = 0
    i = open_paren
    n = len(s)
    while i < n:
        c = s[i]
        if c == "(":
            depth += 1
        elif c == ")":
            depth -= 1
            if depth == 0:
                return i
        i += 1
    return None


def _has_infix_between_first_two_date_calls(s: str) -> bool:
    """True if an infix op appears between the first two date() calls.

    Checks for `-`, `<`, `>`, `==`, etc. after the first date(...) closes
    and before the second date(. Ignores content inside the first call
    (e.g. hyphens in ISO strings inside the first literal).
    """
    positions = _date_call_positions(s)
    if len(positions) < 2:
        return False
    first_kw, second_kw = positions[0], positions[1]
    open1 = _date_open_paren_index(s, first_kw)
    if open1 is None:
        return False
    close1 = _matching_close_paren(s, open1)
    if close1 is None:
        return False
    between = s[close1 + 1 : second_kw]
    return any(op in between for op in _DATE_DATE_INFIX_OPS)


def _has_duration_method(s: str) -> bool:
    return ".Hours()" in s or ".Minutes()" in s or ".Seconds()" in s


@dataclass
class ContainsDatetimeExpression(Evaluator):
    """Proposal config should include expr-lang datetime usage.

    Checks for date, duration, now, or duration methods.
    """

    def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> EvaluationReason:
        if ctx.output.proposal is None:
            return EvaluationReason(value=False, reason="No proposal to check")

        texts = list(iter_config_strings_from_operations(ctx.output.proposal.operations))
        if not texts:
            return EvaluationReason(
                value=False, reason="No configuration strings in proposal operations"
            )

        combined = "\n".join(texts)

        if _THREE_ARG_HINT in combined:
            return EvaluationReason(
                value=False,
                reason=(
                    "Three-argument date(str, format, ...) style "
                    "is not supported in SuperPlane expressions"
                ),
            )

        date_count = len(_DATE_CALL_RE.findall(combined))
        has_date_and_duration = (
            "duration(" in combined and _DATE_CALL_RE.search(combined) is not None
        )
        has_now_and_duration = _NOW_CALL_RE.search(combined) is not None and "duration(" in combined
        has_two_date_infix = date_count >= 2 and any(
            _has_infix_between_first_two_date_calls(t) for t in texts
        )
        has_date_and_duration_method = _DATE_CALL_RE.search(combined) is not None and any(
            _has_duration_method(t) for t in texts
        )

        if (
            has_two_date_infix
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
                "Expected datetime expression patterns: two date() with "
                "- or comparison between them, or date()+duration(), "
                "or now()+duration(), or date() with .Hours/.Minutes/.Seconds"
            ),
        )
