from typing import Annotated, Any, Literal

from pydantic import BaseModel, Field


class CanvasQuestionRequest(BaseModel):
    question: str = Field(min_length=1, max_length=2000)
    canvas_id: str | None = Field(default=None, min_length=1, max_length=200)


class CanvasNode(BaseModel):
    id: str
    name: str | None = None
    type: str | None = None
    block_name: str | None = None


class CanvasEdge(BaseModel):
    source_id: str
    target_id: str
    channel: str = "default"


class CanvasSummary(BaseModel):
    canvas_id: str
    name: str | None = None
    description: str | None = None
    nodes: list[CanvasNode] = Field(default_factory=list)
    edges: list[CanvasEdge] = Field(default_factory=list)


class CanvasShapeNode(BaseModel):
    n: str
    k: str | None = None
    b: str | None = None


class CanvasShapeEdge(BaseModel):
    s: str
    t: str


class CanvasShape(BaseModel):
    canvas_id: str
    name: str | None = None
    node_count: int
    edge_count: int
    nodes: list[CanvasShapeNode] = Field(default_factory=list)
    edges: list[CanvasShapeEdge] = Field(default_factory=list)


class NodeEvent(BaseModel):
    id: str | None = None
    node_id: str | None = None
    channel: str | None = None
    created_at: str | None = None
    data: dict[str, object] = Field(default_factory=dict)


class NodeDetails(BaseModel):
    canvas_id: str
    node: CanvasNode
    configuration: dict[str, object] = Field(default_factory=dict)
    recent_events: list[NodeEvent] = Field(default_factory=list)


class AnswerCitation(BaseModel):
    kind: str = Field(description="canvas|node|edge|event")
    id: str | None = None
    note: str | None = None


class CanvasOperationNodeRef(BaseModel):
    node_key: str | None = Field(default=None, alias="nodeKey")
    node_id: str | None = Field(default=None, alias="nodeId")
    node_name: str | None = Field(default=None, alias="nodeName")
    handle_id: str | None = Field(default=None, alias="handleId")


class CanvasOperationPosition(BaseModel):
    x: float
    y: float


class AddNodeOperation(BaseModel):
    type: Literal["add_node"]
    block_name: str = Field(alias="blockName")
    node_key: str | None = Field(default=None, alias="nodeKey")
    node_name: str | None = Field(default=None, alias="nodeName")
    configuration: dict[str, Any] = Field(default_factory=dict)
    position: CanvasOperationPosition | None = None
    source: CanvasOperationNodeRef | None = None


class ConnectNodesOperation(BaseModel):
    type: Literal["connect_nodes"]
    source: CanvasOperationNodeRef
    target: CanvasOperationNodeRef


class DisconnectNodesOperation(BaseModel):
    type: Literal["disconnect_nodes"]
    source: CanvasOperationNodeRef
    target: CanvasOperationNodeRef


class UpdateNodeConfigOperation(BaseModel):
    type: Literal["update_node_config"]
    target: CanvasOperationNodeRef
    configuration: dict[str, Any] = Field(default_factory=dict)
    node_name: str | None = Field(default=None, alias="nodeName")


class DeleteNodeOperation(BaseModel):
    type: Literal["delete_node"]
    target: CanvasOperationNodeRef


CanvasOperation = Annotated[
    AddNodeOperation
    | ConnectNodesOperation
    | DisconnectNodesOperation
    | UpdateNodeConfigOperation
    | DeleteNodeOperation,
    Field(discriminator="type"),
]


class CanvasProposal(BaseModel):
    summary: str = Field(min_length=1, max_length=500)
    operations: list[CanvasOperation] = Field(default_factory=list)


class CanvasAnswer(BaseModel):
    answer: str = Field(min_length=1, max_length=4000)
    confidence: float = Field(ge=0.0, le=1.0, default=0.5)
    citations: list[AnswerCitation] = Field(default_factory=list)
    follow_up_questions: list[str] = Field(default_factory=list)
    proposal: CanvasProposal | None = None
