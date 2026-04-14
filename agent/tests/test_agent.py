from ai.agent import (
    AgentDeps,
    _catalog_list_cache_key,
    _get_cached_catalog_list,
    _put_cached_catalog_list,
    build_agent,
)
from ai.jsonutil import to_jsonable
from ai.models import CanvasAnswer, CanvasProposal


def test_build_agent_returns_agent_instance() -> None:
    agent = build_agent()
    assert agent is not None


def test_canvas_answer_serializes_proposal_with_aliases() -> None:
    answer = CanvasAnswer(
        answer="Plan ready.",
        confidence=0.8,
        proposal=CanvasProposal(
            summary="Add a webhook trigger and connect to Slack.",
            changeset={
                "changes": [
                    {
                        "type": "ADD_NODE",
                        "node": {
                            "id": "trigger_1",
                            "name": "Inbound Webhook",
                            "block": "webhook.inbound",
                        },
                    },
                    {
                        "type": "ADD_NODE",
                        "node": {
                            "id": "slack_1",
                            "name": "Send Slack Message",
                            "block": "slack.send_message",
                        },
                    },
                    {
                        "type": "ADD_EDGE",
                        "edge": {
                            "sourceId": "trigger_1",
                            "targetId": "slack_1",
                            "channel": "default",
                        },
                    },
                ]
            },
        ),
    )

    payload = to_jsonable(answer)
    assert isinstance(payload, dict)
    proposal = payload.get("proposal")
    assert isinstance(proposal, dict)
    changeset = proposal.get("changeset")
    assert isinstance(changeset, dict)
    changes = changeset.get("changes")
    assert isinstance(changes, list)
    assert changes[0]["type"] == "ADD_NODE"
    assert changes[0]["node"]["block"] == "webhook.inbound"
    assert changes[0]["node"]["id"] == "trigger_1"


def test_catalog_list_cache_key_normalizes_provider_and_query() -> None:
    assert _catalog_list_cache_key("components", "Slack", "  Text ") == (
        "components",
        "slack",
        "text",
    )
    assert _catalog_list_cache_key("triggers", None, None) == ("triggers", "", "")


def test_catalog_list_cache_returns_detached_rows() -> None:
    deps = AgentDeps(client=None, canvas_id="c")  # type: ignore[arg-type]
    _put_cached_catalog_list(
        deps,
        "components",
        "slack",
        None,
        [{"name": "slack.sendTextMessage", "output_channel_names": ["default"]}],
    )
    first = _get_cached_catalog_list(deps, "components", "slack", None)
    assert first is not None
    first[0]["name"] = "mutated"
    second = _get_cached_catalog_list(deps, "components", "slack", None)
    assert second is not None
    assert second[0]["name"] == "slack.sendTextMessage"
    assert second[0]["output_channel_names"] == ["default"]


def test_catalog_list_cache_same_key_after_case_normalization() -> None:
    deps = AgentDeps(client=None, canvas_id="c")  # type: ignore[arg-type]
    _put_cached_catalog_list(deps, "triggers", "GitHub", "PR", [{"name": "github.onPR"}])
    hit = _get_cached_catalog_list(deps, "triggers", "github", "pr")
    assert hit is not None
    assert hit[0]["name"] == "github.onPR"
