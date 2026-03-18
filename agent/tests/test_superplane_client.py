from typing import Any

from ai.superplane_client import SuperplaneClient, SuperplaneClientConfig


class FakeSuperplaneClient(SuperplaneClient):
    def __init__(self, payloads: dict[str, dict[str, Any]]) -> None:
        super().__init__(
            SuperplaneClientConfig(
                base_url="https://example.test",
                api_token="token",
                organization_id="org-id",
            )
        )
        self._payloads = payloads

    def _request_json(self, path: str, query: dict[str, str | int] | None = None) -> dict[str, Any]:
        _ = query
        payload = self._payloads.get(path)
        if payload is None:
            raise ValueError(f"Missing payload for path: {path}")
        return payload


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
