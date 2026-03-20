"""In-memory Superplane API stub for evals and local runs (no HTTP)."""

from __future__ import annotations

from copy import deepcopy
from typing import Any

from ai.models import CanvasSummary
from ai.superplane_client import SuperplaneClient, SuperplaneClientConfig

_CATALOG_TRIGGER_START: dict[str, Any] = {
    "name": "start",
    "provider": None,
    "label": "Manual Run",
    "description": "Start a new execution chain manually",
    "icon": "play",
    "color": "purple",
    "configuration_fields": [],
    "required_fields": [],
    "example_data": None,
}

_CATALOG_COMPONENT_NOOP: dict[str, Any] = {
    "name": "noop",
    "provider": None,
    "label": "No Operation",
    "description": "Just pass events through without any additional processing",
    "icon": "circle-off",
    "color": "blue",
    "configuration_fields": [],
    "required_fields": [],
    "output_channels": [
        {"name": "default", "label": "Default", "description": ""},
    ],
    "example_output": None,
}


class StubSuperplaneClient(SuperplaneClient):
    """``SuperplaneClient`` shape for agent tools; catalog is only ``start`` and ``noop``."""

    def __init__(self) -> None:
        super().__init__(
            SuperplaneClientConfig(
                base_url="https://stub.superplane.eval",
                api_token="stub",
                organization_id="stub-org",
            )
        )

    def describe_canvas(self, canvas_id: str) -> CanvasSummary:
        return CanvasSummary(canvas_id=canvas_id, name="Stub canvas", nodes=[], edges=[])

    def list_components(
        self,
        provider: str | None = None,
        query: str | None = None,
    ) -> list[dict[str, Any]]:
        if SuperplaneClient._matches_filters(
            name=_CATALOG_COMPONENT_NOOP["name"],
            label=_CATALOG_COMPONENT_NOOP["label"],
            description=_CATALOG_COMPONENT_NOOP["description"],
            provider=provider,
            query=query,
        ):
            return [deepcopy(_CATALOG_COMPONENT_NOOP)]
        return []

    def describe_component(self, name: str) -> dict[str, Any]:
        if name != _CATALOG_COMPONENT_NOOP["name"]:
            msg = f"Component '{name}' was not found or response shape was invalid."
            raise ValueError(msg)
        return deepcopy(_CATALOG_COMPONENT_NOOP)

    def list_triggers(
        self,
        provider: str | None = None,
        query: str | None = None,
    ) -> list[dict[str, Any]]:
        if SuperplaneClient._matches_filters(
            name=_CATALOG_TRIGGER_START["name"],
            label=_CATALOG_TRIGGER_START["label"],
            description=_CATALOG_TRIGGER_START["description"],
            provider=provider,
            query=query,
        ):
            return [deepcopy(_CATALOG_TRIGGER_START)]
        return []

    def describe_trigger(self, name: str) -> dict[str, Any]:
        if name != _CATALOG_TRIGGER_START["name"]:
            msg = f"Trigger '{name}' was not found or response shape was invalid."
            raise ValueError(msg)
        return deepcopy(_CATALOG_TRIGGER_START)

    def list_org_integrations(self) -> list[dict[str, Any]]:
        return []

    def list_integration_resources(
        self,
        integration_id: str,
        type: str,
        parameters: dict[str, str] | None = None,
    ) -> list[dict[str, Any]]:
        _ = (integration_id, type, parameters)
        return []
