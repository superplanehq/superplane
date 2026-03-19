from dataclasses import dataclass, field

from ai.models import CanvasShape, CanvasSummary
from ai.superplane_client import SuperplaneClient


@dataclass
class AgentDeps:
    client: SuperplaneClient
    default_canvas_id: str | None = None
    canvas_cache: dict[str, CanvasSummary] = field(default_factory=dict)
    canvas_shape_cache: dict[str, CanvasShape] = field(default_factory=dict)
    allow_canvas_details: bool = False
