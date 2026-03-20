from typing import Any

from ai.superplane_client import SuperplaneClient, SuperplaneClientConfig
from superplaneapi.models.canvases_describe_canvas_response import CanvasesDescribeCanvasResponse
from superplaneapi.models.canvases_list_node_events_response import CanvasesListNodeEventsResponse
from superplaneapi.models.components_list_components_response import ComponentsListComponentsResponse
from superplaneapi.models.superplane_integrations_list_integrations_response import (
    SuperplaneIntegrationsListIntegrationsResponse,
)
from superplaneapi.models.triggers_list_triggers_response import TriggersListTriggersResponse


class FakeCanvasApi:
    def __init__(self, payloads: dict[str, dict[str, Any]]) -> None:
        self._payloads = payloads

    def canvases_describe_canvas(
        self, canvas_id: str, _request_timeout: int | tuple[int, int] | None = None
    ) -> CanvasesDescribeCanvasResponse:
        _ = _request_timeout
        payload = self._payloads.get(f"/api/v1/canvases/{canvas_id}")
        if payload is None:
            raise ValueError(f"Missing payload for canvas: {canvas_id}")
        return CanvasesDescribeCanvasResponse.from_dict(payload)


class FakeCanvasNodeApi:
    def __init__(self, payloads: dict[str, dict[str, Any]]) -> None:
        self._payloads = payloads

    def canvases_list_node_events(
        self,
        canvas_id: str,
        node_id: str,
        limit: int | None = None,
        before: object | None = None,
        _request_timeout: int | tuple[int, int] | None = None,
    ) -> CanvasesListNodeEventsResponse:
        _ = (limit, before, _request_timeout)
        payload = self._payloads.get(f"/api/v1/canvases/{canvas_id}/nodes/{node_id}/events")
        if payload is None:
            raise ValueError(f"Missing payload for node events: {canvas_id}/{node_id}")
        return CanvasesListNodeEventsResponse.from_dict(payload)


class FakeComponentApi:
    def __init__(self, payloads: dict[str, dict[str, Any]]) -> None:
        self._payloads = payloads

    def components_list_components(
        self, _request_timeout: int | tuple[int, int] | None = None
    ) -> ComponentsListComponentsResponse:
        _ = _request_timeout
        payload = self._payloads.get("/api/v1/components")
        if payload is None:
            raise ValueError("Missing payload for components list.")
        return ComponentsListComponentsResponse.from_dict(payload)


class FakeTriggerApi:
    def __init__(self, payloads: dict[str, dict[str, Any]]) -> None:
        self._payloads = payloads

    def triggers_list_triggers(
        self, _request_timeout: int | tuple[int, int] | None = None
    ) -> TriggersListTriggersResponse:
        _ = _request_timeout
        payload = self._payloads.get("/api/v1/triggers")
        if payload is None:
            raise ValueError("Missing payload for triggers list.")
        return TriggersListTriggersResponse.from_dict(payload)


class FakeIntegrationApi:
    def __init__(self, payloads: dict[str, dict[str, Any]]) -> None:
        self._payloads = payloads

    def integrations_list_integrations(
        self, _request_timeout: int | tuple[int, int] | None = None
    ) -> SuperplaneIntegrationsListIntegrationsResponse:
        _ = _request_timeout
        payload = self._payloads.get("/api/v1/integrations")
        if payload is None:
            raise ValueError("Missing payload for integration catalog list.")
        return SuperplaneIntegrationsListIntegrationsResponse.from_dict(payload)


class FakeSuperplaneClient(SuperplaneClient):
    def __init__(self, payloads: dict[str, dict[str, Any]]) -> None:
        super().__init__(
            SuperplaneClientConfig(
                base_url="https://example.test",
                api_token="token",
                organization_id="org-id",
            )
        )
        self._canvas_api = FakeCanvasApi(payloads)
        self._canvas_node_api = FakeCanvasNodeApi(payloads)
        self._component_api = FakeComponentApi(payloads)
        self._trigger_api = FakeTriggerApi(payloads)
        self._integration_api = FakeIntegrationApi(payloads)


def test_describe_canvas_maps_nodes_and_edges() -> None:
    client = FakeSuperplaneClient(
        payloads={
            "/api/v1/canvases/canvas-1": {
                "canvas": {
                    "metadata": {"id": "canvas-1", "name": "Demo"},
                    "spec": {
                        "nodes": [
                            {
                                "id": "node-trigger",
                                "name": "On Push",
                                "type": "TYPE_TRIGGER",
                                "trigger": {"name": "github.onPush"},
                            },
                            {
                                "id": "node-action",
                                "name": "Notify Slack",
                                "type": "TYPE_COMPONENT",
                                "component": {"name": "slack.sendTextMessage"},
                            },
                        ],
                        "edges": [
                            {
                                "sourceId": "node-trigger",
                                "targetId": "node-action",
                                "channel": "default",
                            },
                        ],
                    },
                }
            }
        }
    )

    summary = client.describe_canvas("canvas-1")

    assert summary.canvas_id == "canvas-1"
    assert summary.name == "Demo"
    assert len(summary.nodes) == 2
    assert summary.nodes[0].block_name == "github.onPush"
    assert len(summary.edges) == 1


def test_get_node_details_includes_recent_events() -> None:
    client = FakeSuperplaneClient(
        payloads={
            "/api/v1/canvases/canvas-1": {
                "canvas": {
                    "metadata": {"id": "canvas-1"},
                    "spec": {
                        "nodes": [
                            {
                                "id": "node-action",
                                "name": "Notify Slack",
                                "type": "TYPE_COMPONENT",
                                "component": {"name": "slack.sendTextMessage"},
                            }
                        ],
                        "edges": [],
                    },
                }
            },
            "/api/v1/canvases/canvas-1/nodes/node-action/events": {
                "events": [
                    {
                        "id": "evt-1",
                        "nodeId": "node-action",
                        "channel": "default",
                        "createdAt": "2026-01-01T00:00:00Z",
                        "data": {"ok": True},
                    }
                ]
            },
        }
    )

    details = client.get_node_details(canvas_id="canvas-1", node_id="node-action")

    assert details.node.id == "node-action"
    assert len(details.recent_events) == 1
    assert details.recent_events[0].id == "evt-1"


def test_get_canvas_shape_returns_nodes_and_connections_without_channel_details() -> None:
    client = FakeSuperplaneClient(
        payloads={
            "/api/v1/canvases/canvas-1": {
                "canvas": {
                    "metadata": {"id": "canvas-1", "name": "Demo"},
                    "spec": {
                        "nodes": [
                            {
                                "id": "node-trigger",
                                "name": "On Push",
                                "type": "TYPE_TRIGGER",
                                "trigger": {"name": "github.onPush"},
                            },
                            {
                                "id": "node-action",
                                "name": "Notify Slack",
                                "type": "TYPE_COMPONENT",
                                "component": {"name": "slack.sendTextMessage"},
                            },
                        ],
                        "edges": [
                            {
                                "sourceId": "node-trigger",
                                "targetId": "node-action",
                                "channel": "default",
                            },
                        ],
                    },
                }
            }
        }
    )

    shape = client.get_canvas_shape("canvas-1")

    assert shape.canvas_id == "canvas-1"
    assert shape.node_count == 2
    assert shape.edge_count == 1
    assert shape.nodes[0].b == "github.onPush"
    assert shape.nodes[0].n == "On Push"
    assert shape.nodes[0].k == "trigger"
    assert shape.edges[0].s == "On Push"
    assert shape.edges[0].t == "Notify Slack"
    assert not hasattr(shape.edges[0], "channel")


def test_list_components_includes_integration_scoped_components() -> None:
    client = FakeSuperplaneClient(
        payloads={
            "/api/v1/components": {"components": [{"name": "noop", "label": "Noop"}]},
            "/api/v1/integrations": {
                "integrations": [
                    {
                        "name": "slack",
                        "components": [
                            {
                                "name": "slack.sendTextMessage",
                                "label": "Send Text Message",
                            }
                        ],
                    }
                ]
            },
        }
    )

    components = client.list_components(provider="slack")

    assert len(components) == 1
    assert components[0]["name"] == "slack.sendTextMessage"
    assert components[0]["provider"] == "slack"


def test_list_triggers_includes_integration_scoped_triggers() -> None:
    client = FakeSuperplaneClient(
        payloads={
            "/api/v1/triggers": {"triggers": [{"name": "start", "label": "Manual Run"}]},
            "/api/v1/integrations": {
                "integrations": [
                    {
                        "name": "github",
                        "triggers": [
                            {
                                "name": "github.onPullRequestReviewComment",
                                "label": "On Pull Request Review Comment",
                            }
                        ],
                    }
                ]
            },
        }
    )

    triggers = client.list_triggers(provider="github")

    assert len(triggers) == 1
    assert triggers[0]["name"] == "github.onPullRequestReviewComment"
    assert triggers[0]["provider"] == "github"
