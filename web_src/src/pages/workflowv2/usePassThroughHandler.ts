import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import { workflowsInvokeNodeExecutionAction, WorkflowsWorkflow } from "@/api-client";
import { workflowKeys } from "@/hooks/useWorkflowData";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";

type Params = {
  workflowId: string;
  organizationId?: string;
  workflow?: WorkflowsWorkflow | null;
};

export function usePassThroughHandler({ workflowId, organizationId, workflow }: Params) {
  const queryClient = useQueryClient();
  const refetchNodeData = useNodeExecutionStore((state) => state.refetchNodeData);

  const onPassThrough = useCallback(
    async (nodeId: string, executionId: string) => {
      try {
        await workflowsInvokeNodeExecutionAction(
          withOrganizationHeader({
            path: {
              workflowId,
              executionId: executionId,
              actionName: "passThrough",
            },
            body: { parameters: {} },
          }),
        );

        await queryClient.invalidateQueries({ queryKey: workflowKeys.nodeExecution(workflowId, nodeId) });
        const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);

        if (node) {
          await refetchNodeData(workflowId, nodeId, node.type!, queryClient);
        }

        showSuccessToast("Pushed through");
      } catch (error) {
        console.error("Failed to push through:", error);
        showErrorToast("Failed to push through");
      }
    },
    [workflowId, organizationId, queryClient, workflow?.spec?.nodes, refetchNodeData],
  );

  const supportsPassThrough = useCallback(
    (nodeId: string) => {
      const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);
      const name = node?.component?.name;
      return name === "wait" || name === "time_gate";
    },
    [workflow],
  );

  return { onPassThrough, supportsPassThrough } as const;
}
