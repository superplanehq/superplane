from typing import Any, Protocol

from pydantic_ai import Tool

from ai.tools.describe_component import DescribeComponent
from ai.tools.describe_trigger import DescribeTrigger
from ai.tools.get_canvas import GetCanvas
from ai.tools.get_canvas_memory import GetCanvasMemory
from ai.tools.get_canvas_shape import GetCanvasShape
from ai.tools.get_decision_pattern import GetDecisionPattern
from ai.tools.get_node_details import GetNodeDetails
from ai.tools.list_available_integrations import ListAvailableIntegrations
from ai.tools.list_components import ListComponents
from ai.tools.list_decision_patterns import ListDecisionPatterns
from ai.tools.list_integration_resources import ListIntegrationResources
from ai.tools.list_node_events import ListNodeEvents
from ai.tools.list_node_executions import ListNodeExecutions
from ai.tools.list_org_integrations import ListOrgIntegrations
from ai.tools.list_triggers import ListTriggers
from ai.tools.search_decision_patterns import SearchDecisionPatterns


class _CanvasTool(Protocol):
    name: str
    description: str

    @staticmethod
    def run(*args: Any, **kwargs: Any) -> Any: ...


def _as_tool(cls: type[_CanvasTool]) -> Tool[Any]:
    return Tool(cls.run, name=cls.name, description=cls.description)


default_tools: list[Tool[Any]] = [
    _as_tool(GetCanvas),
    _as_tool(GetCanvasMemory),
    _as_tool(ListDecisionPatterns),
    _as_tool(SearchDecisionPatterns),
    _as_tool(GetDecisionPattern),
    _as_tool(ListComponents),
    _as_tool(DescribeComponent),
    _as_tool(ListTriggers),
    _as_tool(DescribeTrigger),
    _as_tool(ListOrgIntegrations),
    _as_tool(ListAvailableIntegrations),
    _as_tool(ListIntegrationResources),
    _as_tool(GetCanvasShape),
    _as_tool(GetNodeDetails),
    _as_tool(ListNodeEvents),
    _as_tool(ListNodeExecutions),
]

__all__ = [
    "DescribeComponent",
    "DescribeTrigger",
    "GetCanvas",
    "GetCanvasMemory",
    "GetCanvasShape",
    "GetDecisionPattern",
    "GetNodeDetails",
    "ListAvailableIntegrations",
    "ListComponents",
    "ListDecisionPatterns",
    "ListIntegrationResources",
    "ListNodeEvents",
    "ListNodeExecutions",
    "ListOrgIntegrations",
    "ListTriggers",
    "SearchDecisionPatterns",
    "default_tools",
]
