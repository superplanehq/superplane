from ai.agent import build_agent, build_prompt
from ai.models import CanvasQuestionRequest


def test_build_prompt_contains_question() -> None:
    prompt = build_prompt(CanvasQuestionRequest(question="What triggers this flow?"))
    assert "triggers" in prompt


def test_build_agent_returns_agent_instance() -> None:
    agent = build_agent()
    assert agent is not None
