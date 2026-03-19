import os
from dataclasses import dataclass
from typing import Any

from ai.models import (
    CanvasEdge,
    CanvasNode,
    CanvasShape,
    CanvasShapeEdge,
    CanvasShapeNode,
    CanvasSummary,
    NodeDetails,
    NodeEvent,
)
from superplaneapi.api.canvas_api import CanvasApi
from superplaneapi.api.canvas_node_api import CanvasNodeApi
from superplaneapi.api_client import ApiClient
from superplaneapi.configuration import Configuration
from superplaneapi.exceptions import ApiException
from superplaneapi.models.canvases_describe_canvas_response import CanvasesDescribeCanvasResponse
from superplaneapi.models.canvases_list_node_events_response import CanvasesListNodeEventsResponse
from superplaneapi.models.components_node_type import ComponentsNodeType


def _debug_enabled() -> bool:
    return os.getenv("REPL_WEB_DEBUG", "").strip().lower() in {"1", "true", "yes", "on"}


def _debug_log(message: str) -> None:
    if _debug_enabled():
        print(f"[repl_web][superplane_client] {message}", flush=True)


@dataclass(frozen=True)
class SuperplaneClientConfig:
    base_url: str
    api_token: str
    organization_id: str
    timeout_seconds: int = 20


class SuperplaneClient:
    def __init__(self, config: SuperplaneClientConfig) -> None:
        self._config = config
        configuration = Configuration(
            host=config.base_url.rstrip("/"),
            ignore_operation_servers=True,
        )
        self._api_client = ApiClient(configuration=configuration)
        self._api_client.set_default_header("Authorization", f"Bearer {self._config.api_token}")
        self._api_client.set_default_header("x-organization-id", self._config.organization_id)
        self._api_client.set_default_header("Accept", "application/json")
        self._api_client.set_default_header(
            "User-Agent",
            os.getenv("SUPERPLANE_USER_AGENT", "curl/8.7.1"),
        )
        self._canvas_api = CanvasApi(self._api_client)
        self._canvas_node_api = CanvasNodeApi(self._api_client)

    def _with_error_guidance(self, callback: Any, operation: str) -> Any:
        _debug_log(
            f"request operation={operation} base_url={self._config.base_url.rstrip('/')} "
            f"org_id={self._config.organization_id} timeout={self._config.timeout_seconds}s"
        )
        try:
            response = callback()
            _debug_log(f"response operation={operation} status=ok")
            return response
        except ApiException as error:
            status = error.status if isinstance(error.status, int) else None
            response_text = error.body if isinstance(error.body, str) else ""
            _debug_log(
                f"http_error operation={operation} status={status} body={response_text[:400]}"
            )

            guidance = (
                "Check SUPERPLANE_API_TOKEN, SUPERPLANE_ORG_ID, and canvas access permissions."
                if status in {401, 403}
                else "Check SUPERPLANE_BASE_URL and request parameters."
            )
            if response_text:
                details = f" HTTP {status}: {response_text}"
            else:
                details = f" HTTP {status or 'unknown'}."
            raise RuntimeError(f"Superplane API request failed.{details} {guidance}") from error
        except Exception as error:
            _debug_log(f"url_error operation={operation} reason={error}")
            raise RuntimeError(
                "Failed to reach Superplane API. "
                "Check SUPERPLANE_BASE_URL and network connectivity."
            ) from error

    def describe_canvas(self, canvas_id: str) -> CanvasSummary:
        response = self._with_error_guidance(
            lambda: self._canvas_api.canvases_describe_canvas(
                canvas_id,
                _request_timeout=self._config.timeout_seconds,
            ),
            operation="canvases_describe_canvas",
        )
        if not isinstance(response, CanvasesDescribeCanvasResponse):
            raise ValueError("Expected typed response from Superplane API.")

        raw_canvas = response.canvas
        if raw_canvas is None:
            raise ValueError("Canvas response is missing 'canvas'.")
        metadata = raw_canvas.metadata
        spec = raw_canvas.spec
        raw_nodes = spec.nodes if spec is not None and spec.nodes is not None else []
        nodes: list[CanvasNode] = []
        for item in raw_nodes:
            node_id = item.id
            if not isinstance(node_id, str) or not node_id:
                continue

            block_name: str | None = None
            if item.trigger is not None and isinstance(item.trigger.name, str):
                block_name = item.trigger.name
            elif item.component is not None and isinstance(item.component.name, str):
                block_name = item.component.name

            nodes.append(
                CanvasNode(
                    id=node_id,
                    name=item.name if isinstance(item.name, str) else None,
                    type=item.type.value if isinstance(item.type, ComponentsNodeType) else None,
                    block_name=block_name,
                )
            )

        raw_edges = spec.edges if spec is not None and spec.edges is not None else []
        edges: list[CanvasEdge] = []
        for item in raw_edges:
            source_id = item.source_id
            target_id = item.target_id
            if not isinstance(source_id, str) or not isinstance(target_id, str):
                continue
            channel = item.channel
            edges.append(
                CanvasEdge(
                    source_id=source_id,
                    target_id=target_id,
                    channel=channel if isinstance(channel, str) and channel else "default",
                )
            )

        return CanvasSummary(
            canvas_id=(
                metadata.id
                if metadata is not None and isinstance(metadata.id, str)
                else canvas_id
            ),
            name=metadata.name if metadata is not None and isinstance(metadata.name, str) else None,
            description=metadata.description
            if metadata is not None and isinstance(metadata.description, str)
            else None,
            nodes=nodes,
            edges=edges,
        )

    def get_canvas_shape(self, canvas_id: str) -> CanvasShape:
        summary = self.describe_canvas(canvas_id)
        node_kind_by_type = {
            "TYPE_TRIGGER": "trigger",
            "TYPE_COMPONENT": "component",
            "TYPE_BLUEPRINT": "blueprint",
            "TYPE_WIDGET": "widget",
        }
        node_label_by_id: dict[str, str] = {}
        shape_nodes = [
            CanvasShapeNode(
                n=(node.name or node.id),
                k=node_kind_by_type.get(node.type or ""),
                b=node.block_name,
            )
            for node in summary.nodes
        ]
        for node in summary.nodes:
            node_label_by_id[node.id] = node.name or node.id

        edge_pairs = {
            (
                node_label_by_id.get(edge.source_id, edge.source_id),
                node_label_by_id.get(edge.target_id, edge.target_id),
            )
            for edge in summary.edges
        }
        shape_edges = [
            CanvasShapeEdge(s=source, t=target) for (source, target) in sorted(edge_pairs)
        ]
        return CanvasShape(
            canvas_id=summary.canvas_id,
            name=summary.name,
            node_count=len(shape_nodes),
            edge_count=len(shape_edges),
            nodes=shape_nodes,
            edges=shape_edges,
        )

    def list_node_events(self, canvas_id: str, node_id: str, limit: int = 5) -> list[NodeEvent]:
        response = self._with_error_guidance(
            lambda: self._canvas_node_api.canvases_list_node_events(
                canvas_id,
                node_id,
                limit=limit,
                _request_timeout=self._config.timeout_seconds,
            ),
            operation="canvases_list_node_events",
        )
        if not isinstance(response, CanvasesListNodeEventsResponse) or not isinstance(
            response.events, list
        ):
            return []

        events: list[NodeEvent] = []
        for item in response.events:
            data = item.data
            events.append(
                NodeEvent(
                    id=item.id if isinstance(item.id, str) else None,
                    node_id=item.node_id if isinstance(item.node_id, str) else None,
                    channel=item.channel if isinstance(item.channel, str) else None,
                    created_at=item.created_at.isoformat()
                    if item.created_at is not None
                    else None,
                    data=data if isinstance(data, dict) else {},
                )
            )
        return events

    def get_node_details(
        self, canvas_id: str, node_id: str, include_recent_events: bool = True
    ) -> NodeDetails:
        canvas = self.describe_canvas(canvas_id)
        node = next((current for current in canvas.nodes if current.id == node_id), None)
        if node is None:
            raise ValueError(f"Node '{node_id}' not found in canvas '{canvas_id}'.")

        recent_events = self.list_node_events(canvas_id, node_id) if include_recent_events else []
        return NodeDetails(
            canvas_id=canvas.canvas_id,
            node=node,
            configuration={},
            recent_events=recent_events,
        )
