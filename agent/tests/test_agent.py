from collections.abc import Callable
from types import SimpleNamespace
from typing import cast

import pytest
from pydantic_ai import ModelRetry

import ai.agent as agent_module
from ai.agent import (
    AgentContextState,
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


def _build_proposal_validator(
    monkeypatch: pytest.MonkeyPatch,
) -> Callable[[SimpleNamespace, CanvasAnswer], CanvasAnswer]:
    captured: dict[str, object] = {}

    def _capture_output_validator(self: object, func: object) -> object:
        captured["validator"] = func
        return func

    monkeypatch.setattr(agent_module.Agent, "output_validator", _capture_output_validator)
    build_agent()
    validator = captured.get("validator")
    assert callable(validator)
    return cast(Callable[[SimpleNamespace, CanvasAnswer], CanvasAnswer], validator)


def test_validate_answer_proposal_calls_server_validation(monkeypatch: pytest.MonkeyPatch) -> None:
    validator = _build_proposal_validator(monkeypatch)

    class StubClient:
        def __init__(self) -> None:
            self.calls: list[dict[str, object]] = []

        def validate_canvas_version_changeset(
            self, canvas_id: str, canvas_version_id: str, changeset: object
        ) -> None:
            self.calls.append(
                {
                    "canvas_id": canvas_id,
                    "canvas_version_id": canvas_version_id,
                    "changeset": changeset,
                }
            )

    client = StubClient()
    deps = AgentDeps(
        client=client,  # type: ignore[arg-type]
        canvas_id="canvas-1",
        agent_context=AgentContextState(enabled=True, mode="build", canvas_version="draft-ver"),
    )
    answer = CanvasAnswer(
        answer="Plan ready",
        confidence=0.9,
        proposal=CanvasProposal(summary="s", changeset={"changes": []}),
    )

    result = validator(SimpleNamespace(deps=deps), answer)

    assert result is answer
    assert len(client.calls) == 1
    assert client.calls[0]["canvas_id"] == "canvas-1"
    assert client.calls[0]["canvas_version_id"] == "draft-ver"


def test_validate_answer_proposal_raises_retry_on_validation_failure(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    validator = _build_proposal_validator(monkeypatch)

    class StubClient:
        def validate_canvas_version_changeset(
            self, canvas_id: str, canvas_version_id: str, changeset: object
        ) -> None:
            _ = (canvas_id, canvas_version_id, changeset)
            raise RuntimeError("validation failed")

    deps = AgentDeps(
        client=StubClient(),  # type: ignore[arg-type]
        canvas_id="canvas-1",
        agent_context=AgentContextState(enabled=True, mode="build", canvas_version="draft-ver"),
    )
    answer = CanvasAnswer(
        answer="Plan ready",
        confidence=0.9,
        proposal=CanvasProposal(summary="s", changeset={"changes": []}),
    )

    with pytest.raises(ModelRetry):
        validator(SimpleNamespace(deps=deps), answer)


def test_validate_answer_proposal_raises_retry_without_canvas_version(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    validator = _build_proposal_validator(monkeypatch)

    class StubClient:
        def validate_canvas_version_changeset(
            self, canvas_id: str, canvas_version_id: str, changeset: object
        ) -> None:
            _ = (canvas_id, canvas_version_id, changeset)
            raise AssertionError("should not be called")

    deps = AgentDeps(client=StubClient(), canvas_id="canvas-1")  # type: ignore[arg-type]
    answer = CanvasAnswer(
        answer="Plan ready",
        confidence=0.9,
        proposal=CanvasProposal(summary="s", changeset={"changes": []}),
    )

    with pytest.raises(ModelRetry):
        validator(SimpleNamespace(deps=deps), answer)
