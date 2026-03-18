import pytest
from pydantic import ValidationError

from ai.models import CanvasQuestionRequest


def test_canvas_question_request_accepts_valid_question() -> None:
    payload = CanvasQuestionRequest(question="What triggers this flow?")
    assert payload.question == "What triggers this flow?"


def test_canvas_question_request_rejects_empty_question() -> None:
    with pytest.raises(ValidationError):
        CanvasQuestionRequest(question="")
