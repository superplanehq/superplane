from pydantic_ai import Agent

from ai.deps import AgentDeps
from ai.models import CanvasAnswer
from ai.tools.get_canvas import get_canvas
from ai.tools.get_canvas_shape import get_canvas_shape
from ai.tools.get_node_details import get_node_details
from ai.tools.request_canvas_details import request_canvas_details


def register_tools(agent: Agent[AgentDeps, CanvasAnswer]) -> None:
    agent.tool(get_canvas_shape)
    agent.tool(request_canvas_details)
    agent.tool(get_canvas)
    agent.tool(get_node_details)
