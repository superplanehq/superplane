import type {
  SuperplaneComponentsEdge as ComponentsEdge,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";

export type CanvasChangesetChangeType = "ADD_NODE" | "UPDATE_NODE" | "DELETE_NODE" | "ADD_EDGE" | "DELETE_EDGE";

export type CanvasChangesetChange = {
  type?: CanvasChangesetChangeType;
  node?: ComponentsNode;
  nodeId?: string;
  edge?: ComponentsEdge;
  edgeId?: string;
};
