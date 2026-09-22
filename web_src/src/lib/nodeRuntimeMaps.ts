import type { CanvasesCanvasEvent, CanvasesCanvasNodeExecution, CanvasesCanvasNodeQueueItem } from "@/api-client";

export type NodeRuntimeMaps = {
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>;
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>;
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>;
};

export const EMPTY_NODE_RUNTIME_MAPS: NodeRuntimeMaps = {
  nodeExecutionsMap: {},
  nodeQueueItemsMap: {},
  nodeEventsMap: {},
};

export type NodeRuntimeStoreEntry = {
  executions: CanvasesCanvasNodeExecution[];
  queueItems: CanvasesCanvasNodeQueueItem[];
  events: CanvasesCanvasEvent[];
};

export function buildNodeRuntimeMaps(data: Iterable<[string, NodeRuntimeStoreEntry]>): NodeRuntimeMaps {
  const nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]> = {};
  const nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]> = {};
  const nodeEventsMap: Record<string, CanvasesCanvasEvent[]> = {};

  for (const [nodeId, entry] of data) {
    if (entry.executions.length > 0) {
      nodeExecutionsMap[nodeId] = entry.executions;
    }
    if (entry.queueItems.length > 0) {
      nodeQueueItemsMap[nodeId] = entry.queueItems;
    }
    if (entry.events.length > 0) {
      nodeEventsMap[nodeId] = entry.events;
    }
  }

  return { nodeExecutionsMap, nodeQueueItemsMap, nodeEventsMap };
}

export function nodeRuntimeMapsEqual(left: NodeRuntimeMaps, right: NodeRuntimeMaps): boolean {
  return (
    recordRefsEqual(left.nodeExecutionsMap, right.nodeExecutionsMap) &&
    recordRefsEqual(left.nodeQueueItemsMap, right.nodeQueueItemsMap) &&
    recordRefsEqual(left.nodeEventsMap, right.nodeEventsMap)
  );
}

function recordRefsEqual<T>(left: Record<string, T>, right: Record<string, T>): boolean {
  const leftKeys = Object.keys(left);
  if (leftKeys.length !== Object.keys(right).length) {
    return false;
  }
  return leftKeys.every((key) => left[key] === right[key]);
}
