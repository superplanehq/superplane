import json
import os
from dataclasses import dataclass
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import Request, urlopen

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

    def _build_url(self, path: str, query: dict[str, str | int] | None = None) -> str:
        base = self._config.base_url.rstrip("/")
        url = f"{base}{path}"
        if query:
            url = f"{url}?{urlencode(query)}"
        return url

    def _headers(self) -> dict[str, str]:
        user_agent = os.getenv("SUPERPLANE_USER_AGENT", "curl/8.7.1")
        return {
            "Authorization": f"Bearer {self._config.api_token}",
            "x-organization-id": self._config.organization_id,
            "Accept": "application/json",
            "User-Agent": user_agent,
        }

    def _request_json(self, path: str, query: dict[str, str | int] | None = None) -> dict[str, Any]:
        url = self._build_url(path, query)
        _debug_log(
            f"request method=GET url={url} org_id={self._config.organization_id} timeout={self._config.timeout_seconds}s"
        )
        request = Request(
            url=url,
            method="GET",
            headers=self._headers(),
        )
        try:
            with urlopen(request, timeout=self._config.timeout_seconds) as response:
                raw = response.read().decode("utf-8")
                _debug_log(f"response status={response.status} bytes={len(raw.encode('utf-8'))}")
        except HTTPError as error:
            response_text = ""
            try:
                response_text = error.read().decode("utf-8")
            except Exception:
                response_text = ""
            _debug_log(f"http_error status={error.code} body={response_text[:400]}")

            guidance = (
                "Check SUPERPLANE_API_TOKEN, SUPERPLANE_ORG_ID, and canvas access permissions."
                if error.code in {401, 403}
                else "Check SUPERPLANE_BASE_URL and request parameters."
            )
            details = (
                f" HTTP {error.code}: {response_text}" if response_text else f" HTTP {error.code}."
            )
            raise RuntimeError(f"Superplane API request failed.{details} {guidance}") from error
        except URLError as error:
            _debug_log(f"url_error reason={error}")
            raise RuntimeError(
                "Failed to reach Superplane API. "
                "Check SUPERPLANE_BASE_URL and network connectivity."
            ) from error

        payload = json.loads(raw)
        if not isinstance(payload, dict):
            raise ValueError("Expected JSON object response from Superplane API.")
        return payload

    def describe_canvas(self, canvas_id: str) -> CanvasSummary:
        payload = self._request_json(f"/api/v1/canvases/{canvas_id}")
        raw_canvas = payload.get("canvas")
        if not isinstance(raw_canvas, dict):
            raise ValueError("Canvas response is missing 'canvas'.")

        metadata = (
            raw_canvas.get("metadata") if isinstance(raw_canvas.get("metadata"), dict) else {}
        )
        spec = raw_canvas.get("spec") if isinstance(raw_canvas.get("spec"), dict) else {}

        raw_nodes = spec.get("nodes") if isinstance(spec.get("nodes"), list) else []
        nodes: list[CanvasNode] = []
        for item in raw_nodes:
            if not isinstance(item, dict):
                continue
            node_id = item.get("id")
            if not isinstance(node_id, str) or not node_id.strip():
                continue

            block_name: str | None = None
            trigger = item.get("trigger")
            component = item.get("component")
            if isinstance(trigger, dict) and isinstance(trigger.get("name"), str):
                block_name = trigger["name"]
            elif isinstance(component, dict) and isinstance(component.get("name"), str):
                block_name = component["name"]

            nodes.append(
                CanvasNode(
                    id=node_id,
                    name=item.get("name") if isinstance(item.get("name"), str) else None,
                    type=item.get("type") if isinstance(item.get("type"), str) else None,
                    block_name=block_name,
                )
            )

        raw_edges = spec.get("edges") if isinstance(spec.get("edges"), list) else []
        edges: list[CanvasEdge] = []
        for item in raw_edges:
            if not isinstance(item, dict):
                continue
            source_id = item.get("sourceId")
            target_id = item.get("targetId")
            if not isinstance(source_id, str) or not isinstance(target_id, str):
                continue
            channel = item.get("channel")
            edges.append(
                CanvasEdge(
                    source_id=source_id,
                    target_id=target_id,
                    channel=channel if isinstance(channel, str) and channel.strip() else "default",
                )
            )

        return CanvasSummary(
            canvas_id=metadata.get("id") if isinstance(metadata.get("id"), str) else canvas_id,
            name=metadata.get("name") if isinstance(metadata.get("name"), str) else None,
            description=(
                metadata.get("description")
                if isinstance(metadata.get("description"), str)
                else None
            ),
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
        payload = self._request_json(
            f"/api/v1/canvases/{canvas_id}/nodes/{node_id}/events",
            query={"limit": limit},
        )
        raw_events = payload.get("events")
        if not isinstance(raw_events, list):
            return []

        events: list[NodeEvent] = []
        for item in raw_events:
            if not isinstance(item, dict):
                continue
            data = item.get("data")
            events.append(
                NodeEvent(
                    id=item.get("id") if isinstance(item.get("id"), str) else None,
                    node_id=item.get("nodeId") if isinstance(item.get("nodeId"), str) else None,
                    channel=item.get("channel") if isinstance(item.get("channel"), str) else None,
                    created_at=(
                        item.get("createdAt") if isinstance(item.get("createdAt"), str) else None
                    ),
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
