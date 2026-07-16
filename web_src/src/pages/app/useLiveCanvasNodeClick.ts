import { useCallback, useEffect, useRef } from "react";
import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import { runLiveCanvasNodeClickLookup } from "./liveCanvasNodeClickLookup";
import {
  resolveLiveCanvasNodeClickSyncAction,
  shouldDeferRunInspectionForLiveNodeClick,
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

  const cancelLiveNodeClickLookup = useCallback(() => {
    liveCanvasNodeClickLookupRef.current += 1;
  }, []);

  const handleLiveCanvasNodeClick = useCallback(
    (
      nodeId: string,
      actions: {
        openConfigurationSidebar: (options?: { preferSettingsTab?: boolean }) => void;
      },
    ) => {
      if (isRunInspectionMode || isEditing || !liveSidebarRunLookupEnabled) return;

      const lookupId = liveCanvasNodeClickLookupRef.current + 1;
      liveCanvasNodeClickLookupRef.current = lookupId;

      const workflowNode = canvasNodesById.get(nodeId);
      const nodeActivity = useNodeExecutionStore.getState().getNodeData(nodeId);
      if (shouldDeferRunInspectionForLiveNodeClick(workflowNode, nodeActivity)) {
        actions.openConfigurationSidebar({ preferSettingsTab: true });
        return;
      }

      const syncAction = resolveLiveCanvasNodeClickSyncAction(nodeId, workflowNode, resolveRunIdForSidebarEvent);

      if (syncAction.kind === "inspectRun") {
        handleSelectRunFromSidebarEvent(syncAction.runId, { nodeId });
        return;
      }

      void runLiveCanvasNodeClickLookup({
        nodeId,
        workflowNode,
        isLookupStale: () => liveCanvasNodeClickLookupRef.current !== lookupId,
        resolveLatestNodeRunLookupEvent,
        openConfigurationSidebar: actions.openConfigurationSidebar,
        fetchRunIdForSidebarEvent,
        handleSelectRunFromSidebarEvent,
      });
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
