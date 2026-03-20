from typing import Any

from ai.superplane_client import SuperplaneClient, SuperplaneClientConfig
from superplaneapi.models.canvases_describe_canvas_response import CanvasesDescribeCanvasResponse
from superplaneapi.models.canvases_list_node_events_response import CanvasesListNodeEventsResponse


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
