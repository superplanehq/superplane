import { useCallback, useEffect, useRef } from "react";
import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import { runLiveCanvasNodeClickLookup } from "./liveCanvasNodeClickLookup";
import {
  resolveLiveCanvasNodeClickSyncAction,
  shouldOpenConfigurationSidebarForLiveNodeClick,
} from "./runInspectionLiveNodeLookup";

type UseLiveCanvasNodeClickOptions = {
  canvasNodesById: Map<string, ComponentsNode>;
  fetchRunIdForSidebarEvent: (event: SidebarEvent, options?: { maxPages?: number }) => Promise<string | null>;
  handleSelectRunFromSidebarEvent: (runId: string, options?: { nodeId?: string }) => void;
  isEditing: boolean;
  isRunInspectionMode: boolean;
  liveSidebarRunLookupEnabled: boolean;
  resolveLatestNodeRunLookupEvent: (nodeId: string) => Promise<SidebarEvent | null>;
  resolveRunIdForSidebarEvent: (event: SidebarEvent) => string | null;
};

export function useLiveCanvasNodeClick({
  canvasNodesById,
  fetchRunIdForSidebarEvent,
  handleSelectRunFromSidebarEvent,
  isEditing,
  isRunInspectionMode,
  liveSidebarRunLookupEnabled,
  resolveLatestNodeRunLookupEvent,
  resolveRunIdForSidebarEvent,
}: UseLiveCanvasNodeClickOptions) {
  const liveCanvasNodeClickLookupRef = useRef(0);
  const liveCanvasNodeClickLookupNodeRef = useRef<string | null>(null);

  const cancelLiveNodeClickLookup = useCallback((closingNodeId?: string) => {
    const lookupNodeId = liveCanvasNodeClickLookupNodeRef.current;
    if (closingNodeId && lookupNodeId && closingNodeId !== lookupNodeId) {
      return;
    }

    liveCanvasNodeClickLookupRef.current += 1;
    liveCanvasNodeClickLookupNodeRef.current = null;
  }, []);

  const handleLiveCanvasNodeClick = useCallback(
    (
      nodeId: string,
      actions: {
        openConfigurationSidebar: (options?: { preferSettingsTab?: boolean }) => void;
      },
    ) => {
      if (isRunInspectionMode || isEditing) return;

      if (!liveSidebarRunLookupEnabled) {
        actions.openConfigurationSidebar();
        return;
      }

      const lookupId = liveCanvasNodeClickLookupRef.current + 1;
      liveCanvasNodeClickLookupRef.current = lookupId;

      const workflowNode = canvasNodesById.get(nodeId);
      const nodeActivity = useNodeExecutionStore.getState().getNodeData(nodeId);
      if (shouldOpenConfigurationSidebarForLiveNodeClick(workflowNode, nodeActivity)) {
        actions.openConfigurationSidebar({ preferSettingsTab: true });
        return;
      }

      const syncAction = resolveLiveCanvasNodeClickSyncAction(nodeId, workflowNode, resolveRunIdForSidebarEvent);

      if (syncAction.kind === "inspectRun") {
        handleSelectRunFromSidebarEvent(syncAction.runId, { nodeId });
        return;
      }

      void (async () => {
        liveCanvasNodeClickLookupNodeRef.current = nodeId;
        try {
          await runLiveCanvasNodeClickLookup({
            nodeId,
            workflowNode,
            isLookupStale: () => liveCanvasNodeClickLookupRef.current !== lookupId,
            resolveLatestNodeRunLookupEvent,
            openConfigurationSidebar: actions.openConfigurationSidebar,
            fetchRunIdForSidebarEvent,
            handleSelectRunFromSidebarEvent,
          });
        } finally {
          if (liveCanvasNodeClickLookupRef.current === lookupId) {
            liveCanvasNodeClickLookupNodeRef.current = null;
          }
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

  return {
    cancelLiveNodeClickLookup,
    handleLiveCanvasNodeClick,
  };
}
