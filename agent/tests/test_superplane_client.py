from typing import Any

from ai.superplane_client import SuperplaneClient, SuperplaneClientConfig
from superplaneapi.models.actions_list_actions_response import (
    ActionsListActionsResponse,
)
from superplaneapi.models.canvases_canvas_changeset import CanvasesCanvasChangeset
from superplaneapi.models.canvases_create_canvas_version_response import (
    CanvasesCreateCanvasVersionResponse,
)
from superplaneapi.models.canvases_describe_canvas_response import CanvasesDescribeCanvasResponse
from superplaneapi.models.canvases_describe_canvas_version_response import (
    CanvasesDescribeCanvasVersionResponse,
)
from superplaneapi.models.canvases_list_node_events_response import CanvasesListNodeEventsResponse
from superplaneapi.models.canvases_list_node_executions_response import (
    CanvasesListNodeExecutionsResponse,
)
from superplaneapi.models.canvases_validate_canvas_version_changeset_body import (
    CanvasesValidateCanvasVersionChangesetBody,
)
from superplaneapi.models.canvases_validate_canvas_version_changeset_response import (
    CanvasesValidateCanvasVersionChangesetResponse,
)
from superplaneapi.models.organizations_list_integration_resources_response import (
    OrganizationsListIntegrationResourcesResponse,
)
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
        result = CanvasesDescribeCanvasResponse.from_dict(payload)
        assert result is not None
        return result


class FakeCanvasVersionApi:
    def __init__(self, payloads: dict[str, dict[str, Any]]) -> None:
        self._payloads = payloads
        self.create_calls: list[dict[str, Any]] = []
        self.validate_calls: list[dict[str, Any]] = []

    def canvases_describe_canvas_version(
        self,
        canvas_id: str,
        version_id: str,
        _request_timeout: int | tuple[int, int] | None = None,
    ) -> CanvasesDescribeCanvasVersionResponse:
        _ = _request_timeout
        key = f"/api/v1/canvases/{canvas_id}/versions/{version_id}"
        payload = self._payloads.get(key)
        if payload is None:
            raise ValueError(f"Missing payload for canvas version: {canvas_id}/{version_id}")
        result = CanvasesDescribeCanvasVersionResponse.from_dict(payload)
        assert result is not None
        return result

    def canvases_create_canvas_version(
        self,
        canvas_id: str,
        body: dict[str, Any],
        _request_timeout: int | tuple[int, int] | None = None,
    ) -> CanvasesCreateCanvasVersionResponse:
        self.create_calls.append(
            {
                "canvas_id": canvas_id,
                "body": body,
                "request_timeout": _request_timeout,
            }
        )
        key = f"/api/v1/canvases/{canvas_id}/versions"
        payload = self._payloads.get(key)
        if payload is None:
            raise ValueError(f"Missing payload for canvas version create: {canvas_id}")
        result = CanvasesCreateCanvasVersionResponse.from_dict(payload)
        assert result is not None
        return result

    def canvases_validate_canvas_version_changeset(
        self,
        canvas_id: str,
        version_id: str,
        body: CanvasesValidateCanvasVersionChangesetBody,
        _request_timeout: int | tuple[int, int] | None = None,
    ) -> CanvasesValidateCanvasVersionChangesetResponse:
        self.validate_calls.append(
            {
                "canvas_id": canvas_id,
                "version_id": version_id,
                "body": body,
                "request_timeout": _request_timeout,
            }
        )
        key = f"/api/v1/canvases/{canvas_id}/versions/{version_id}/validate"
        payload = self._payloads.get(key)
        if payload is None:
            raise ValueError(
                f"Missing payload for canvas version changeset validation: {canvas_id}/{version_id}"
            )
        result = CanvasesValidateCanvasVersionChangesetResponse.from_dict(payload)
        assert result is not None
        return result


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
        result = CanvasesListNodeEventsResponse.from_dict(payload)
        assert result is not None
        return result

    def canvases_list_node_executions(
        self,
        canvas_id: str,
        node_id: str,
        states: list[str] | None = None,
        results: list[str] | None = None,
        limit: int | None = None,
        before: object | None = None,
        _request_timeout: int | tuple[int, int] | None = None,
    ) -> CanvasesListNodeExecutionsResponse:
        _ = (states, results, before, _request_timeout)
        payload = self._payloads.get(f"/api/v1/canvases/{canvas_id}/nodes/{node_id}/executions")
        if payload is None:
            raise ValueError(f"Missing payload for node executions: {canvas_id}/{node_id}")
        result = CanvasesListNodeExecutionsResponse.from_dict(payload)
        assert result is not None
        return result


class FakeActionApi:
    def __init__(self, payloads: dict[str, dict[str, Any]]) -> None:
        self._payloads = payloads

    def actions_list_actions(
        self, _request_timeout: int | tuple[int, int] | None = None
    ) -> ActionsListActionsResponse:
        _ = _request_timeout
        payload = self._payloads.get("/api/v1/actions")
        if payload is None:
            raise ValueError("Missing payload for actions list.")
        result = ActionsListActionsResponse.from_dict(payload)
        assert result is not None
        return result


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
        result = TriggersListTriggersResponse.from_dict(payload)
        assert result is not None
        return result


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
        result = SuperplaneIntegrationsListIntegrationsResponse.from_dict(payload)
        assert result is not None
        return result


class FakeListResourcesResponseData:
    def __init__(self) -> None:
        self.status = 200
        self.data: bytes | None = None

    def read(self) -> bytes:
        self.data = b'{"resources":[{"id":"proj-1","name":"Project 1","type":"project"}]}'
        return self.data

    def getheader(self, name: str) -> str | None:
        if name.lower() == "content-type":
            return "application/json"
        return None

    def getheaders(self) -> list[tuple[str, str]]:
        return [("content-type", "application/json")]


class FakeListResourcesApiClient:
    def __init__(self) -> None:
        self.query_params: list[tuple[str, str]] = []
        self.request_timeout: int | tuple[int, int] | None = None

    def param_serialize(
        self,
        method: str,
        resource_path: str,
        path_params: dict[str, str] | None = None,
        query_params: list[tuple[str, str]] | None = None,
        header_params: dict[str, str] | None = None,
        body: Any = None,
        post_params: list[Any] | None = None,
        files: dict[str, Any] | None = None,
        auth_settings: list[str] | None = None,
        collection_formats: dict[str, str] | None = None,
        _host: str | None = None,
        _request_auth: Any = None,
    ) -> tuple[str, str, dict[str, str], Any, list[Any]]:
        _ = (
            path_params,
            header_params,
            body,
            post_params,
            files,
            auth_settings,
            collection_formats,
            _host,
            _request_auth,
        )
        self.query_params = query_params or []
        return method, f"https://example.test{resource_path}", {}, None, []

    def call_api(
        self,
        method: str,
        url: str,
        header_params: dict[str, str] | None = None,
        body: Any = None,
        post_params: list[Any] | None = None,
        _request_timeout: int | tuple[int, int] | None = None,
    ) -> FakeListResourcesResponseData:
        _ = (method, url, header_params, body, post_params)
        self.request_timeout = _request_timeout
        return FakeListResourcesResponseData()

    def response_deserialize(
        self,
        response_data: FakeListResourcesResponseData,
        response_types_map: dict[str, str] | None = None,
    ) -> Any:
        _ = (response_data, response_types_map)
        payload = OrganizationsListIntegrationResourcesResponse.from_dict(
            {"resources": [{"id": "proj-1", "name": "Project 1", "type": "project"}]}
        )
        assert payload is not None
        return type("ResponseWrapper", (), {"data": payload})()


class FakeSuperplaneClient(SuperplaneClient):
    def __init__(self, payloads: dict[str, dict[str, Any]]) -> None:
        super().__init__(
            SuperplaneClientConfig(
                base_url="https://example.test",
                api_token="token",
                organization_id="org-id",
            )
        )
        self._canvas_api = FakeCanvasApi(payloads)  # type: ignore[assignment]
        self._canvas_version_api = FakeCanvasVersionApi(payloads)  # type: ignore[assignment]
        self._canvas_node_api = FakeCanvasNodeApi(payloads)  # type: ignore[assignment]
        self._action_api = FakeActionApi(payloads)  # type: ignore[assignment]
        self._trigger_api = FakeTriggerApi(payloads)  # type: ignore[assignment]
        self._integration_api = FakeIntegrationApi(payloads)  # type: ignore[assignment]


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
                                "component": "github.onPush",
                            },
                            {
                                "id": "node-action",
                                "name": "Notify Slack",
                                "type": "TYPE_ACTION",
                                "component": "slack.sendTextMessage",
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


def test_describe_editing_canvas_prefers_version_nodes_and_live_metadata_name() -> None:
    client = FakeSuperplaneClient(
        payloads={
            "/api/v1/canvases/canvas-1": {
                "canvas": {
                    "metadata": {"id": "canvas-1", "name": "Production name"},
                    "spec": {
                        "nodes": [
                            {
                                "id": "live-only",
                                "name": "Live node",
                                "type": "TYPE_ACTION",
                                "component": "noop",
                            }
                        ],
                        "edges": [],
                    },
                }
            },
            "/api/v1/canvases/canvas-1/versions/draft-ver": {
                "version": {
                    "metadata": {"id": "draft-ver", "canvasId": "canvas-1"},
                    "spec": {
                        "nodes": [
                            {
                                "id": "draft-only",
                                "name": "Draft node",
                                "type": "TYPE_ACTION",
                                "component": "slack.sendTextMessage",
                            }
                        ],
                        "edges": [],
                    },
                }
            },
        }
    )

    summary = client.describe_editing_canvas("canvas-1", "draft-ver")

    assert summary.name == "Production name"
    assert len(summary.nodes) == 1
    assert summary.nodes[0].id == "draft-only"
    assert summary.nodes[0].name == "Draft node"


def test_validate_canvas_version_changeset_sends_expected_payload() -> None:
    client = FakeSuperplaneClient(
        payloads={
            "/api/v1/canvases/canvas-1/versions/draft-ver/validate": {
                "version": {
                    "metadata": {"id": "draft-ver", "canvasId": "canvas-1"},
                    "spec": {"nodes": [], "edges": []},
                }
            }
        }
    )
    changeset = CanvasesCanvasChangeset.from_dict(
        {
            "changes": [
                {
                    "type": "ADD_NODE",
                    "node": {
                        "id": "node-1",
                        "name": "Slack",
                        "block": "slack.sendTextMessage",
                    },
                }
            ]
        }
    )
    assert changeset is not None

    response = client.validate_canvas_version_changeset(
        canvas_id="canvas-1",
        canvas_version_id="draft-ver",
        changeset=changeset,
    )

    assert response.version is not None
    assert response.version.metadata is not None
    assert response.version.metadata.id == "draft-ver"

    canvas_version_api = client._canvas_version_api
    assert isinstance(canvas_version_api, FakeCanvasVersionApi)
    assert len(canvas_version_api.validate_calls) == 1
    call = canvas_version_api.validate_calls[0]
    assert call["canvas_id"] == "canvas-1"
    assert call["version_id"] == "draft-ver"
    body = call["body"]
    assert isinstance(body, CanvasesValidateCanvasVersionChangesetBody)
    assert body.changeset is not None
    assert body.changeset.to_dict().get("changes", [])[0]["node"]["id"] == "node-1"


def test_create_canvas_version_returns_metadata_id_and_sends_empty_body() -> None:
    client = FakeSuperplaneClient(
        payloads={
            "/api/v1/canvases/canvas-1/versions": {
                "version": {
                    "metadata": {"id": "draft-ver", "canvasId": "canvas-1"},
                    "spec": {"nodes": [], "edges": []},
                }
            }
        }
    )

    version_id = client.create_canvas_version("canvas-1")
    assert version_id == "draft-ver"

    canvas_version_api = client._canvas_version_api
    assert isinstance(canvas_version_api, FakeCanvasVersionApi)
    assert len(canvas_version_api.create_calls) == 1
    call = canvas_version_api.create_calls[0]
    assert call["canvas_id"] == "canvas-1"
    assert call["body"] == {}


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
                                "type": "TYPE_ACTION",
                                "component": "slack.sendTextMessage",
                                "configuration": {"channel": "#alerts", "text": "hello"},
                                "errorMessage": "missing scope",
                                "warningMessage": "deprecated field",
                                "paused": True,
                                "integration": {"id": "int-1", "name": "Slack workspace"},
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
    assert details.configuration == {"channel": "#alerts", "text": "hello"}
    assert details.error_message == "missing scope"
    assert details.warning_message == "deprecated field"
    assert details.paused is True
    assert details.integration == {"id": "int-1", "name": "Slack workspace"}
    assert len(details.recent_events) == 1
    assert details.recent_events[0].id == "evt-1"


def test_list_node_executions_maps_rows() -> None:
    client = FakeSuperplaneClient(
        payloads={
            "/api/v1/canvases/canvas-1": {
                "canvas": {
                    "metadata": {"id": "canvas-1"},
                    "spec": {"nodes": [{"id": "n1", "type": "TYPE_ACTION"}], "edges": []},
                }
            },
            "/api/v1/canvases/canvas-1/nodes/n1/executions": {
                "executions": [
                    {
                        "id": "ex-1",
                        "state": "STATE_FINISHED",
                        "result": "RESULT_FAILED",
                        "resultReason": "RESULT_REASON_ERROR",
                        "resultMessage": "timeout",
                        "createdAt": "2026-01-02T00:00:00Z",
                        "updatedAt": "2026-01-02T00:01:00Z",
                    }
                ]
            },
        }
    )

    rows = client.list_node_executions("canvas-1", "n1", limit=5)
    assert len(rows) == 1
    assert rows[0].id == "ex-1"
    assert rows[0].state == "STATE_FINISHED"
    assert rows[0].result == "RESULT_FAILED"
    assert rows[0].result_reason == "RESULT_REASON_ERROR"
    assert rows[0].result_message == "timeout"
    assert rows[0].created_at is not None
    assert rows[0].updated_at is not None


def test_list_integration_resources_sends_top_level_query_params() -> None:
    client = FakeSuperplaneClient(payloads={})
    fake_api_client = FakeListResourcesApiClient()
    client._api_client = fake_api_client  # type: ignore[assignment]

    resources = client.list_integration_resources(
        integration_id="int-1",
        type="project",
        parameters={"region": "us-east-1", "ignored-empty": "", "": "ignored-key"},
    )

    assert resources == [{"id": "proj-1", "name": "Project 1", "type": "project"}]
    query = dict(fake_api_client.query_params)
    assert query["type"] == "project"
    assert query["region"] == "us-east-1"
    assert "parameters" not in query
    assert "ignored-empty" not in query
    assert fake_api_client.request_timeout == client._config.timeout_seconds


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
                                "component": "github.onPush",
                            },
                            {
                                "id": "node-action",
                                "name": "Notify Slack",
                                "type": "TYPE_ACTION",
                                "component": "slack.sendTextMessage",
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
    assert shape.nodes[1].b == "slack.sendTextMessage"
    assert shape.nodes[1].n == "Notify Slack"
    assert shape.nodes[1].k == "action"
    assert shape.edges[0].s == "On Push"
    assert shape.edges[0].t == "Notify Slack"
    assert not hasattr(shape.edges[0], "channel")


def test_list_components_includes_integration_scoped_components() -> None:
    client = FakeSuperplaneClient(
        payloads={
            "/api/v1/actions": {"actions": [{"name": "noop", "label": "Noop"}]},
            "/api/v1/integrations": {
                "integrations": [
                    {
                        "name": "slack",
                        "actions": [
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
    assert "configuration_fields" not in components[0]
    assert components[0].get("output_channel_names") == []


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
                                "name": "github.onPRReviewComment",
                                "label": "On PR Review Comment",
                            }
                        ],
                    }
                ]
            },
        }
    )

    triggers = client.list_triggers(provider="github")

    assert len(triggers) == 1
    assert triggers[0]["name"] == "github.onPRReviewComment"
    assert triggers[0]["provider"] == "github"
    assert "configuration_fields" not in triggers[0]


def test_matches_filters_natural_language_query_matches_block_name() -> None:
    assert SuperplaneClient._matches_filters(
        name="slack.sendTextMessage",
        label="Send Text Message",
        description="",
        provider=None,
        query="slack send text message",
    )


def test_matches_filters_partial_query_tokens() -> None:
    assert SuperplaneClient._matches_filters(
        name="slack.sendTextMessage",
        label="Send Text Message",
        description="",
        provider=None,
        query="slack text",
    )


def test_matches_filters_contiguous_phrase_still_matches() -> None:
    assert SuperplaneClient._matches_filters(
        name="filter",
        label="Filter",
        description="Filter events based on content",
        provider=None,
        query="filter events",
    )


def test_matches_filters_excludes_unrelated_block() -> None:
    assert not SuperplaneClient._matches_filters(
        name="noop",
        label="Noop",
        description="",
        provider=None,
        query="slack",
    )
