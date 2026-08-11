import type { Connection, Edge, Node } from "@xyflow/react";

/**
 * React Flow completes connections by handle proximity, so a drag can be
 * dropped onto a handle of a node that must not accept connections — e.g. a
 * "removed" ghost node rendered by the draft visual diff for a node deleted
 * from the draft. Only allow connections whose endpoints are both connectable
 * nodes currently on the canvas.
 */
export function isValidCanvasConnection(nodes: Node[], connection: Connection | Edge): boolean {
  return isConnectableCanvasNode(nodes, connection.source) && isConnectableCanvasNode(nodes, connection.target);
}

export function isConnectableCanvasNode(nodes: Node[], nodeId: string): boolean {
  const node = nodes.find((candidate) => candidate.id === nodeId);
  if (!node || node.connectable === false) {
    return false;
  }

  return node.data._draftDiffStatus !== "removed";
}
