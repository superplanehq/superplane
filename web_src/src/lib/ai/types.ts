export interface NodeRef {
  nodeKey?: string;
  nodeId?: string;
  nodeName?: string;
}

export interface SourceNodeRef extends NodeRef {
  handleId?: string | null;
}

export interface AddNodeOperation {
  type: "add_node";
  nodeKey?: string;
  blockName: string;
  nodeName?: string;
  configuration?: Record<string, unknown>;
  position?: { x: number; y: number };
  source?: SourceNodeRef;
}

export interface ConnectNodesOperation {
  type: "connect_nodes";
  source: SourceNodeRef;
  target: NodeRef;
}

export interface DisconnectNodesOperation {
  type: "disconnect_nodes";
  source: SourceNodeRef;
  target: NodeRef;
}

export type ConnectionNodesOperation = ConnectNodesOperation | DisconnectNodesOperation;

export interface UpdateNodeConfigOperation {
  type: "update_node_config";
  target: NodeRef;
  configuration: Record<string, unknown>;
  nodeName?: string;
}

export interface DeleteNodeOperation {
  type: "delete_node";
  target: NodeRef;
}

export type CanvasOperation =
  | AddNodeOperation
  | ConnectionNodesOperation
  | UpdateNodeConfigOperation
  | DeleteNodeOperation;
