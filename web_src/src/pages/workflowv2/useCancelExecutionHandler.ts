import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import { workflowsCancelExecution, WorkflowsWorkflow } from "@/api-client";
import { workflowKeys } from "@/hooks/useWorkflowData";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";

type Params = {
  workflowId: string;
  workflow?: WorkflowsWorkflow | null;
};

export function useCancelExecutionHandler({ workflowId, workflow }: Params) {
  const queryClient = useQueryClient();
  const refetchNodeData = useNodeExecutionStore((state) => state.refetchNodeData);

  const onCancelExecution = useCallback(
    async (nodeId: string, executionId: string) => {
      try {
        await workflowsCancelExecution(
          withOrganizationHeader({
            path: {
              workflowId,
              executionId,
            },
          }),
        );

        await queryClient.invalidateQueries({ queryKey: workflowKeys.nodeExecution(workflowId, nodeId) });
        const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);

        if (node) {
          await refetchNodeData(workflowId, nodeId, node.type!, queryClient);
        }

        showSuccessToast("Execution cancelled");
      } catch (error) {
        console.error("Failed to cancel execution:", error);
        showErrorToast("Failed to cancel execution");
      }
    },
    [workflowId, queryClient, workflow?.spec?.nodes, refetchNodeData],
  );

  return { onCancelExecution } as const;
}
