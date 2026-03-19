from pydantic_ai import Agent

from ai.deps import AgentDeps
from ai.models import CanvasAnswer
from ai.tools.get_canvas import register as register_get_canvas
from ai.tools.get_canvas_shape import register as register_get_canvas_shape
from ai.tools.get_node_details import register as register_get_node_details
from ai.tools.request_canvas_details import register as register_request_canvas_details


def register_tools(agent: Agent[AgentDeps, CanvasAnswer]) -> None:
    register_get_canvas_shape(agent)
    register_request_canvas_details(agent)
    register_get_canvas(agent)
    register_get_node_details(agent)
