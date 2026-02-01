import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import { canvasesDeleteNodeQueueItem, CanvasesCanvas } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { showErrorToast } from "@/utils/toast";

type Params = {
  canvasId: string;
  organizationId?: string;
  canvas?: CanvasesCanvas | null;
  loadSidebarData: (nodeId: string) => void;
};

/**
 * Returns a handler that cancels a queued item for a node and refreshes related data.
 */
export function useOnCancelQueueItemHandler({ canvasId, organizationId, canvas, loadSidebarData }: Params) {
  const queryClient = useQueryClient();
  const refetchNodeDataMethod = useNodeExecutionStore((state) => state.refetchNodeData);

  return useCallback(
    async (nodeId: string, queueItemId: string) => {
      if (!window.confirm("Cancel this queued event?")) return;

      try {
        await canvasesDeleteNodeQueueItem(
          withOrganizationHeader({
            path: {
              canvasId,
              nodeId,
              itemId: queueItemId,
            },
          }),
        );

        // Refresh queue items and sidebar data for this node
        await queryClient.invalidateQueries({ queryKey: canvasKeys.nodeQueueItem(canvasId, nodeId) });

        const node = canvas?.spec?.nodes?.find((n) => n.id === nodeId);
        if (node) {
          await refetchNodeDataMethod(canvasId, nodeId, node.type!, queryClient);
        } else {
          loadSidebarData(nodeId);
        }
      } catch (err) {
        console.error("Failed to cancel queue item", err);
        showErrorToast("Failed to cancel queue item");
      }
    },
    [canvasId, organizationId, queryClient, canvas, refetchNodeDataMethod, loadSidebarData],
  );
}
