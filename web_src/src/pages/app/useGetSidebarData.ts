import { useCallback } from "react";
import type {
  ActionsAction,
  CanvasesCanvasEvent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  SuperplaneComponentsNode as ComponentsNode,
  TriggersTrigger,
} from "@/api-client";
import type { SidebarData } from "@/ui/CanvasPage";
import { buildSidebarRuntimeMaps } from "./buildSidebarRuntimeMaps";
import { prepareSidebarData } from "./workflowPageHelpers";

type NodeStoreData = {
  executions: CanvasesCanvasNodeExecution[];
  queueItems: CanvasesCanvasNodeQueueItem[];
  events: CanvasesCanvasEvent[];
  totalInHistoryCount: number;
  totalInQueueCount: number;
  isLoading: boolean;
};

export function useGetSidebarData({
  canvasNodes,
  canvasNodesById,
  allComponents,
  allTriggers,
  visibleNodeEventsMap,
  showNodeRuntimeActivity,
  getNodeData,
}: {
  canvasNodes: ComponentsNode[];
  canvasNodesById: Map<string, ComponentsNode>;
  allComponents: ActionsAction[];
  allTriggers: TriggersTrigger[];
  visibleNodeEventsMap: Record<string, CanvasesCanvasEvent[]>;
  showNodeRuntimeActivity: boolean;
  getNodeData: (nodeId: string) => NodeStoreData;
}) {
  return useCallback(
    (nodeId: string): SidebarData | null => {
      const node = canvasNodesById.get(nodeId);
      if (!node) return null;

      const nodeData = getNodeData(nodeId);
      const { executionsMap, queueItemsMap, eventsMap, totalHistoryCount, totalQueueCount } = buildSidebarRuntimeMaps({
        nodeId,
        showNodeRuntimeActivity,
        nodeData,
        fallbackEvents: visibleNodeEventsMap[nodeId],
      });

      return {
        ...prepareSidebarData(
          node,
          canvasNodes,
          allComponents,
          allTriggers,
          executionsMap,
          queueItemsMap,
          eventsMap,
          totalHistoryCount,
          totalQueueCount,
        ),
        isLoading: nodeData.isLoading,
      };
    },
    [
      canvasNodes,
      canvasNodesById,
      allComponents,
      allTriggers,
      visibleNodeEventsMap,
      showNodeRuntimeActivity,
      getNodeData,
    ],
  );
}
