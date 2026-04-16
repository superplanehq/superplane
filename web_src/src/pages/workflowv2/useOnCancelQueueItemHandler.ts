import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import type { CanvasesCanvas } from "@/api-client";
import { canvasesDeleteNodeQueueItem } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { showErrorToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";

type Params = {
  organizationId: string;
  canvasId: string;
  canvas?: CanvasesCanvas | null;
  loadSidebarData: (nodeId: string) => void;
};

/**
 * Returns a handler that cancels a queued item for a node and refreshes related data.
 */
export function useOnCancelQueueItemHandler({ organizationId, canvasId, canvas, loadSidebarData }: Params) {
  const queryClient = useQueryClient();
  const refetchNodeDataMethod = useNodeExecutionStore((state) => state.refetchNodeData);

  return useCallback(
    async (nodeId: string, queueItemId: string) => {
      if (!window.confirm("Cancel this queued event?")) return;

      try {
        await canvasesDeleteNodeQueueItem(
          withOrganizationHeader({
            organizationId,
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
    [organizationId, canvasId, queryClient, canvas, refetchNodeDataMethod, loadSidebarData],
  );
}
