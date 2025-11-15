import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import { workflowsDeleteNodeQueueItem, WorkflowsWorkflow } from "@/api-client";
import { workflowKeys } from "@/hooks/useWorkflowData";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";

type Params = {
  workflowId: string;
  organizationId?: string;
  workflow?: WorkflowsWorkflow | null;
  loadSidebarData: (nodeId: string) => void;
};

/**
 * Returns a handler that cancels a queued item for a node and refreshes related data.
 */
export function useOnCancelQueueItemHandler({ workflowId, organizationId, workflow, loadSidebarData }: Params) {
  const queryClient = useQueryClient();
  const refetchNodeDataMethod = useNodeExecutionStore((state) => state.refetchNodeData);

  return useCallback(
    async (nodeId: string, queueItemId: string) => {
      if (!window.confirm("Cancel this queued event?")) return;

      try {
        await workflowsDeleteNodeQueueItem(
          withOrganizationHeader({
            path: {
              workflowId,
              nodeId,
              itemId: queueItemId,
            },
          }),
        );

        // Refresh queue items and sidebar data for this node
        await queryClient.invalidateQueries({ queryKey: workflowKeys.nodeQueueItem(workflowId, nodeId) });

        const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);
        if (node) {
          await refetchNodeDataMethod(workflowId, nodeId, node.type!, queryClient);
        } else {
          loadSidebarData(nodeId);
        }
      } catch (err) {
        console.error("Failed to cancel queue item", err);
      }
    },
    [workflowId, organizationId, queryClient, workflow, refetchNodeDataMethod, loadSidebarData],
  );
}
