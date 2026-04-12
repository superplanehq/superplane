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
from ai.tools.load_agent_skill import LoadAgentSkill
from ai.tools.search_decision_patterns import SearchDecisionPatterns

CANVAS_TOOL_CLASSES: tuple[type[Any], ...] = (
    GetCanvas,
    GetCanvasMemory,
    ListDecisionPatterns,
    SearchDecisionPatterns,
    GetDecisionPattern,
    LoadAgentSkill,
    ListComponents,
    DescribeComponent,
    ListTriggers,
    DescribeTrigger,
    ListOrgIntegrations,
    ListAvailableIntegrations,
    ListIntegrationResources,
    GetCanvasShape,
    GetNodeDetails,
    ListNodeEvents,
    ListNodeExecutions,
)

TOOLS_BY_NAME: dict[str, type[Any]] = {cls.name: cls for cls in CANVAS_TOOL_CLASSES}


class _CanvasTool(Protocol):
    name: str
    description: str

    @staticmethod
    def run(*args: Any, **kwargs: Any) -> Any: ...

    @staticmethod
    def label(*args: Any, **kwargs: Any) -> str: ...


def _as_tool(cls: type[_CanvasTool]) -> Tool[Any]:
    return Tool(cls.run, name=cls.name, description=cls.description)


default_tools: list[Tool[Any]] = [_as_tool(cls) for cls in CANVAS_TOOL_CLASSES]
