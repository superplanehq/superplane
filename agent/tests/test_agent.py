from ai.agent import build_agent, build_prompt
from ai.jsonutil import to_jsonable
from ai.models import CanvasAnswer, CanvasProposal, CanvasQuestionRequest


def test_build_prompt_contains_question() -> None:
    prompt = build_prompt(CanvasQuestionRequest(question="What triggers this flow?"))
    assert "triggers" in prompt


def test_build_agent_returns_agent_instance() -> None:
    agent = build_agent()
    assert agent is not None


def test_canvas_answer_serializes_proposal_with_aliases() -> None:
    answer = CanvasAnswer(
        answer="Plan ready.",
        confidence=0.8,
        proposal=CanvasProposal(
            summary="Add a webhook trigger and connect to Slack.",
            operations=[
                {
                    "type": "add_node",
                    "blockName": "webhook.inbound",
                    "nodeKey": "trigger_1",
                    "nodeName": "Inbound Webhook",
                },
                {
                    "type": "add_node",
                    "blockName": "slack.send_message",
                    "nodeKey": "slack_1",
                    "nodeName": "Send Slack Message",
                    "source": {"nodeKey": "trigger_1"},
                },
            ],
        ),
    )

    payload = to_jsonable(answer)
    assert isinstance(payload, dict)
    proposal = payload.get("proposal")
    assert isinstance(proposal, dict)
    operations = proposal.get("operations")
    assert isinstance(operations, list)
    assert operations[0]["blockName"] == "webhook.inbound"
    assert operations[0]["nodeKey"] == "trigger_1"
