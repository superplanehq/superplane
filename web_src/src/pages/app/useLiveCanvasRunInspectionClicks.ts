import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import type { QueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef } from "react";
import type { SidebarEvent } from "@/ui/componentSidebar/types";

import { resolveCachedNodeRunId, resolveRunLookupEventForNodeActivity } from "./runInspectionLiveNodeLookup";

type UseLiveCanvasRunInspectionClicksOptions = {
  isRunInspectionMode: boolean;
  isEditing: boolean;
  liveSidebarRunLookupEnabled: boolean;
  canvasId?: string;
  canvasNodesById: Map<string, ComponentsNode>;
  queryClient: QueryClient;
  refetchNodeDataMethod: (
    workflowId: string,
    nodeId: string,
    nodeType: string,
    queryClient: QueryClient,
  ) => Promise<void>;
  resolveRunIdForSidebarEvent: (event: SidebarEvent) => string | null;
  fetchRunIdForSidebarEvent: (event: SidebarEvent, options?: { maxPages?: number }) => Promise<string | null>;
  handleSelectRunFromSidebarEvent: (runId: string, options?: { nodeId?: string }) => void;
};

export function useLiveCanvasRunInspectionClicks(options: UseLiveCanvasRunInspectionClicksOptions) {
  const {
    isRunInspectionMode,
    isEditing,
    liveSidebarRunLookupEnabled,
    canvasId,
    canvasNodesById,
    queryClient,
    refetchNodeDataMethod,
    resolveRunIdForSidebarEvent,
    fetchRunIdForSidebarEvent,
    handleSelectRunFromSidebarEvent,
  } = options;

  const liveCanvasNodeClickLookupRef = useRef(0);

  const resolveLatestNodeRunLookupEvent = useCallback(
    async (nodeId: string) => {
      const workflowNode = canvasNodesById.get(nodeId);
      if (!workflowNode || !canvasId) {
        return null;
      }

      const nodeType = workflowNode.type || "TYPE_ACTION";
      await refetchNodeDataMethod(canvasId, nodeId, nodeType, queryClient);

      const nodeData = useNodeExecutionStore.getState().getNodeData(nodeId);
      return resolveRunLookupEventForNodeActivity(nodeId, nodeType, nodeData);
    },
    [canvasId, canvasNodesById, refetchNodeDataMethod, queryClient],
  );

  const handleLiveCanvasNodeClick = useCallback(
    (nodeId: string) => {
      if (isRunInspectionMode || isEditing || !liveSidebarRunLookupEnabled) return;

      const lookupId = liveCanvasNodeClickLookupRef.current + 1;
      liveCanvasNodeClickLookupRef.current = lookupId;

      const cachedRunId = resolveCachedNodeRunId(nodeId, canvasNodesById.get(nodeId), resolveRunIdForSidebarEvent);
      if (cachedRunId) return handleSelectRunFromSidebarEvent(cachedRunId, { nodeId });

      void (async () => {
        try {
          const lookupEvent = await resolveLatestNodeRunLookupEvent(nodeId);
          if (!lookupEvent || liveCanvasNodeClickLookupRef.current !== lookupId) return;

          const runId = await fetchRunIdForSidebarEvent(lookupEvent, { maxPages: 1 });
          if (!runId || liveCanvasNodeClickLookupRef.current !== lookupId) return;

          handleSelectRunFromSidebarEvent(runId, { nodeId });
        } catch (error) {
          console.error("Failed to inspect latest node run", error);
        }
      })();
    },
    [
      canvasNodesById,
      fetchRunIdForSidebarEvent,
      handleSelectRunFromSidebarEvent,
      isEditing,
      isRunInspectionMode,
      liveSidebarRunLookupEnabled,
      resolveLatestNodeRunLookupEvent,
      resolveRunIdForSidebarEvent,
    ],
  );

  useEffect(() => {
    liveCanvasNodeClickLookupRef.current += 1;
  }, [isEditing, isRunInspectionMode, liveSidebarRunLookupEnabled]);

  return { handleLiveCanvasNodeClick };
}
