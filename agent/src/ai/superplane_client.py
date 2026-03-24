import os
import warnings
from dataclasses import dataclass
from typing import Any
from urllib.parse import urlencode

# Suppress a known pydantic warning emitted by generated OpenAPI models.
# Keep this narrow to avoid hiding unrelated warnings.
warnings.filterwarnings(
    "ignore",
    message=(
        r'Field name "validate" in "OrganizationsSetAgentOpenAIKeyBody" '
        r'shadows an attribute in parent "BaseModel"'
    ),
    category=UserWarning,
)

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
from superplaneapi.api.component_api import ComponentApi
from superplaneapi.api.integration_api import IntegrationApi
from superplaneapi.api.organization_api import OrganizationApi
from superplaneapi.api.trigger_api import TriggerApi
from superplaneapi.api_client import ApiClient
from superplaneapi.configuration import Configuration
from superplaneapi.exceptions import ApiException
from superplaneapi.models.components_component import ComponentsComponent
from superplaneapi.models.components_describe_component_response import (
    ComponentsDescribeComponentResponse,
)
from superplaneapi.models.canvases_canvas_memory import CanvasesCanvasMemory
from superplaneapi.models.components_list_components_response import ComponentsListComponentsResponse
from superplaneapi.models.canvases_describe_canvas_response import CanvasesDescribeCanvasResponse
from superplaneapi.models.canvases_list_canvas_memories_response import (
    CanvasesListCanvasMemoriesResponse,
)
from superplaneapi.models.canvases_list_node_events_response import CanvasesListNodeEventsResponse
from superplaneapi.models.components_node_type import ComponentsNodeType
from superplaneapi.models.configuration_field import ConfigurationField
from superplaneapi.models.organizations_integration import OrganizationsIntegration
from superplaneapi.models.organizations_list_integration_resources_response import (
    OrganizationsListIntegrationResourcesResponse,
)
from superplaneapi.models.superplane_integrations_list_integrations_response import (
    SuperplaneIntegrationsListIntegrationsResponse,
)
from superplaneapi.models.superplane_organizations_list_integrations_response import (
    SuperplaneOrganizationsListIntegrationsResponse,
)
from superplaneapi.models.triggers_describe_trigger_response import TriggersDescribeTriggerResponse
from superplaneapi.models.triggers_list_triggers_response import TriggersListTriggersResponse
from superplaneapi.models.triggers_trigger import TriggersTrigger


def _debug_enabled() -> bool:
    return os.getenv("REPL_WEB_DEBUG", "").strip().lower() in {"1", "true", "yes", "on"}


def _debug_log(message: str) -> None:
    if _debug_enabled():
        print(message, flush=True)


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
        self._component_api = ComponentApi(self._api_client)
        self._trigger_api = TriggerApi(self._api_client)
        self._integration_api = IntegrationApi(self._api_client)
        self._organization_api = OrganizationApi(self._api_client)

    def _api_request(self, callback: Any, operation: str, fields: dict[str, Any] | None = None) -> Any:
        org_id = self._config.organization_id
        fields_str = " ".join([f"{key}={value}" for key, value in fields.items()]) if fields else ""

        try:
            response = callback()
            _debug_log(f"[api] org={org_id} operation={operation} {fields_str} status=200 OK")
            return response
        except ApiException as error:
            status = error.status if isinstance(error.status, int) else "unknown"
            _debug_log(f"[api] org={org_id} operation={operation} {fields_str} status={status} reason={error}")

            raise RuntimeError("Superplane API request failed.") from error
        except Exception as error:
            _debug_log(f"[api] org={org_id} operation={operation} {fields_str} status=unknown_error reason={error}")

            raise RuntimeError("Failed to reach Superplane API.") from error

    def describe_canvas(self, canvas_id: str) -> CanvasSummary:
        response = self._api_request(
            lambda: self._canvas_api.canvases_describe_canvas(
                canvas_id,
                _request_timeout=self._config.timeout_seconds,
            ),
            operation="canvases_describe_canvas",
            fields={"canvas_id": canvas_id},
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

    @staticmethod
    def _provider_from_name(name: str | None) -> str | None:
        if not isinstance(name, str) or "." not in name:
            return None
        provider = name.split(".", 1)[0].strip()
        return provider or None

    @staticmethod
    def _matches_filters(
        *,
        name: str | None,
        label: str | None,
        description: str | None,
        provider: str | None,
        query: str | None,
    ) -> bool:
        resolved_provider = provider.strip().lower() if isinstance(provider, str) and provider.strip() else None
        resolved_query = query.strip().lower() if isinstance(query, str) and query.strip() else None
        provider_from_name = SuperplaneClient._provider_from_name(name)
        if resolved_provider and provider_from_name != resolved_provider:
            return False
        if not resolved_query:
            return True

        haystack_parts = [name, label, description]
        haystack = " ".join(part for part in haystack_parts if isinstance(part, str)).lower()
        return resolved_query in haystack

    @staticmethod
    def _serialize_configuration_fields(fields: list[ConfigurationField] | None) -> list[dict[str, Any]]:
        if not isinstance(fields, list):
            return []
        serialized: list[dict[str, Any]] = []
        for field in fields:
            if not isinstance(field, ConfigurationField):
                continue
            serialized.append(
                {
                    "name": field.name,
                    "label": field.label,
                    "type": field.type,
                    "description": field.description,
                    "required": bool(field.required),
                    "default_value": field.default_value,
                    "placeholder": field.placeholder,
                    "sensitive": bool(field.sensitive),
                    "togglable": bool(field.togglable),
                    "required_conditions": [
                        condition.model_dump(mode="json", by_alias=True)
                        for condition in (field.required_conditions or [])
                        if condition is not None
                    ],
                    "visibility_conditions": [
                        condition.model_dump(mode="json", by_alias=True)
                        for condition in (field.visibility_conditions or [])
                        if condition is not None
                    ],
                    "type_options": field.type_options.model_dump(mode="json", by_alias=True)
                    if field.type_options is not None
                    else None,
                }
            )
        return serialized

    @staticmethod
    def _serialize_component(component: ComponentsComponent) -> dict[str, Any]:
        output_channels = component.output_channels or []
        configuration_fields = SuperplaneClient._serialize_configuration_fields(component.configuration)
        required_fields = [field["name"] for field in configuration_fields if field["required"] and field["name"]]
        return {
            "name": component.name,
            "provider": SuperplaneClient._provider_from_name(component.name),
            "label": component.label,
            "description": component.description,
            "icon": component.icon,
            "color": component.color,
            "configuration_fields": configuration_fields,
            "required_fields": required_fields,
            "output_channels": [
                {
                    "name": channel.name,
                    "label": channel.label,
                    "description": channel.description,
                }
                for channel in output_channels
                if channel is not None
            ],
            "example_output": component.example_output,
        }

    @staticmethod
    def _serialize_trigger(trigger: TriggersTrigger) -> dict[str, Any]:
        configuration_fields = SuperplaneClient._serialize_configuration_fields(trigger.configuration)
        required_fields = [field["name"] for field in configuration_fields if field["required"] and field["name"]]
        return {
            "name": trigger.name,
            "provider": SuperplaneClient._provider_from_name(trigger.name),
            "label": trigger.label,
            "description": trigger.description,
            "icon": trigger.icon,
            "color": trigger.color,
            "configuration_fields": configuration_fields,
            "required_fields": required_fields,
            "example_data": trigger.example_data,
        }

    @staticmethod
    def _serialize_org_integration(integration: OrganizationsIntegration) -> dict[str, Any]:
        metadata = integration.metadata
        spec = integration.spec
        status = integration.status
        return {
            "id": metadata.id if metadata is not None else None,
            "name": metadata.name if metadata is not None else None,
            "integration_name": spec.integration_name if spec is not None else None,
            "provider": spec.integration_name if spec is not None else None,
            "state": status.state if status is not None else None,
            "state_description": status.state_description if status is not None else None,
        }

    @staticmethod
    def _serialize_available_integration(integration: Any) -> dict[str, Any]:
        components = integration.components if isinstance(integration.components, list) else []
        triggers = integration.triggers if isinstance(integration.triggers, list) else []
        return {
            "name": integration.name,
            "provider": integration.name,
            "label": integration.label,
            "description": integration.description,
            "icon": integration.icon,
            "component_count": len(components),
            "trigger_count": len(triggers),
        }

    def _list_available_integrations_raw(self) -> list[Any]:
        response = self._api_request(
            lambda: self._integration_api.integrations_list_integrations(
                _request_timeout=self._config.timeout_seconds,
            ),
            operation="integrations_list_integrations",
        )
        if not isinstance(response, SuperplaneIntegrationsListIntegrationsResponse):
            return []
        return response.integrations if isinstance(response.integrations, list) else []

    def list_components(
        self,
        provider: str | None = None,
        query: str | None = None,
    ) -> list[dict[str, Any]]:
        response = self._api_request(
            lambda: self._component_api.components_list_components(
                _request_timeout=self._config.timeout_seconds,
            ),
            operation="components_list_components",
        )
        if not isinstance(response, ComponentsListComponentsResponse):
            return []
        root_components = response.components if isinstance(response.components, list) else []
        components_by_name: dict[str, ComponentsComponent] = {}
        for component in root_components:
            if not isinstance(component, ComponentsComponent):
                continue
            if isinstance(component.name, str) and component.name:
                components_by_name[component.name] = component

        # Integration-scoped components are exposed under /api/v1/integrations.
        try:
            integration_definitions = self._list_available_integrations_raw()
        except Exception as error:
            _debug_log(f"integration_components_unavailable reason={error}")
            integration_definitions = []

        for integration in integration_definitions:
            scoped_components = integration.components if isinstance(integration.components, list) else []
            for component in scoped_components:
                if not isinstance(component, ComponentsComponent):
                    continue
                if isinstance(component.name, str) and component.name:
                    components_by_name[component.name] = component

        matches = [
            component
            for component in components_by_name.values()
            if self._matches_filters(
                name=component.name,
                label=component.label,
                description=component.description,
                provider=provider,
                query=query,
            )
        ]
        return [self._serialize_component(component) for component in sorted(matches, key=lambda item: item.name or "")]

    def describe_component(self, name: str) -> dict[str, Any]:
        response = self._api_request(
            lambda: self._component_api.components_describe_component(
                name,
                _request_timeout=self._config.timeout_seconds,
            ),
            operation="components_describe_component",
        )
        if not isinstance(response, ComponentsDescribeComponentResponse) or not isinstance(
            response.component, ComponentsComponent
        ):
            raise ValueError(f"Component '{name}' was not found or response shape was invalid.")
        return self._serialize_component(response.component)

    def list_triggers(
        self,
        provider: str | None = None,
        query: str | None = None,
    ) -> list[dict[str, Any]]:
        response = self._api_request(
            lambda: self._trigger_api.triggers_list_triggers(
                _request_timeout=self._config.timeout_seconds,
            ),
            operation="triggers_list_triggers",
        )
        if not isinstance(response, TriggersListTriggersResponse):
            return []
        root_triggers = response.triggers if isinstance(response.triggers, list) else []
        triggers_by_name: dict[str, TriggersTrigger] = {}
        for trigger in root_triggers:
            if not isinstance(trigger, TriggersTrigger):
                continue
            if isinstance(trigger.name, str) and trigger.name:
                triggers_by_name[trigger.name] = trigger

        try:
            integration_definitions = self._list_available_integrations_raw()
        except Exception as error:
            _debug_log(f"integration_triggers_unavailable reason={error}")
            integration_definitions = []

        for integration in integration_definitions:
            scoped_triggers = integration.triggers if isinstance(integration.triggers, list) else []
            for trigger in scoped_triggers:
                if not isinstance(trigger, TriggersTrigger):
                    continue
                if isinstance(trigger.name, str) and trigger.name:
                    triggers_by_name[trigger.name] = trigger

        matches = [
            trigger
            for trigger in triggers_by_name.values()
            if self._matches_filters(
                name=trigger.name,
                label=trigger.label,
                description=trigger.description,
                provider=provider,
                query=query,
            )
        ]
        return [self._serialize_trigger(trigger) for trigger in sorted(matches, key=lambda item: item.name or "")]

    def list_available_integrations(self) -> list[dict[str, Any]]:
        integrations = self._list_available_integrations_raw()
        return [
            self._serialize_available_integration(integration)
            for integration in integrations
            if integration is not None
        ]

    def describe_trigger(self, name: str) -> dict[str, Any]:
        response = self._api_request(
            lambda: self._trigger_api.triggers_describe_trigger(
                name,
                _request_timeout=self._config.timeout_seconds,
            ),
            operation="triggers_describe_trigger",
        )
        if not isinstance(response, TriggersDescribeTriggerResponse) or not isinstance(
            response.trigger, TriggersTrigger
        ):
            raise ValueError(f"Trigger '{name}' was not found or response shape was invalid.")
        return self._serialize_trigger(response.trigger)

    def list_org_integrations(self) -> list[dict[str, Any]]:
        response = self._api_request(
            lambda: self._organization_api.organizations_list_integrations(
                self._config.organization_id,
                _request_timeout=self._config.timeout_seconds,
            ),
            operation="organizations_list_integrations",
        )
        if not isinstance(response, SuperplaneOrganizationsListIntegrationsResponse):
            return []
        integrations = response.integrations if isinstance(response.integrations, list) else []
        return [
            self._serialize_org_integration(integration)
            for integration in integrations
            if isinstance(integration, OrganizationsIntegration)
        ]

    def list_integration_resources(
        self,
        integration_id: str,
        type: str,
        parameters: dict[str, str] | None = None,
    ) -> list[dict[str, Any]]:
        query_params = {"type": type}
        if isinstance(parameters, dict):
            query_params.update(
                {str(key): str(value) for key, value in parameters.items() if key and value}
            )
        encoded_parameters = urlencode(query_params)
        response = self._api_request(
            lambda: self._organization_api.organizations_list_integration_resources(
                self._config.organization_id,
                integration_id,
                parameters=encoded_parameters,
                _request_timeout=self._config.timeout_seconds,
            ),
            operation="organizations_list_integration_resources",
        )
        if not isinstance(response, OrganizationsListIntegrationResourcesResponse):
            return []
        resources = response.resources if isinstance(response.resources, list) else []
        return [
            {
                "id": resource.id,
                "name": resource.name,
                "type": resource.type,
            }
            for resource in resources
            if resource is not None
        ]

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
        response = self._api_request(
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
