import type { CanvasChangesetChange } from "@/api-client";
import { useCallback } from "react";

export function useFormatOperation(): (change: CanvasChangesetChange) => string {
  return useCallback((change: CanvasChangesetChange) => {
    const getNodeId = (nodeId?: string) => nodeId || "node";

    switch (change.type) {
      case "ADD_NODE":
        return `Add node ${getNodeId(change.node?.id)} (${change.node?.block || "unknown"})`;
      case "UPDATE_NODE":
        return `Update node ${getNodeId(change.node?.id)}`;
      case "DELETE_NODE":
        return `Delete node ${getNodeId(change.node?.id)}`;
      case "ADD_EDGE":
        return `Connect ${getNodeId(change.edge?.sourceId)} -> ${getNodeId(change.edge?.targetId)}`;
      case "DELETE_EDGE":
        return `Disconnect ${getNodeId(change.edge?.sourceId)} -> ${getNodeId(change.edge?.targetId)}`;
      default:
        return "Update canvas";
    }
  }, []);
}
