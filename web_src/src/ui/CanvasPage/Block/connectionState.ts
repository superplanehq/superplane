import type { BlockConnectionState, BlockEdgeState } from "./types";

export function isAlreadyConnectedToNode(
  edges: BlockEdgeState[],
  connection: BlockConnectionState | undefined,
  sourceNodeId: string | undefined,
  targetNodeId: string | undefined,
  sourceHandle?: string | null,
) {
  if (!connection || !sourceNodeId || !targetNodeId) {
    return false;
  }

  return edges.some(
    (edge) =>
      edge.source === sourceNodeId &&
      edge.sourceHandle === (sourceHandle ?? connection.handleId) &&
      edge.target === targetNodeId,
  );
}
