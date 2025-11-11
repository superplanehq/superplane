import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { WorkflowsWorkflow, WorkflowsWorkflowNodeExecution } from "@/api-client";
import { workflowsInvokeNodeExecutionAction } from "@/api-client";
import { workflowKeys } from "@/hooks/useWorkflowData";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { showErrorToast, showSuccessToast } from "@/utils/toast";

type Params = {
  workflowId: string;
  organizationId?: string;
  workflow?: WorkflowsWorkflow | null;
};

export function usePassThroughHandler({ workflowId, organizationId, workflow }: Params) {
  const queryClient = useQueryClient();
  const refetchNodeData = useNodeExecutionStore((state) => state.refetchNodeData);

  const onPassThrough = useCallback(
    async (nodeId: string, executionId?: string) => {
      // Pull latest executions lazily from the store to avoid hook ordering deps
      let executions: WorkflowsWorkflowNodeExecution[] = [];
      try {
        const storeData: any = useNodeExecutionStore.getState().data;
        const nodeData = typeof storeData?.get === "function" ? storeData.get(nodeId) : storeData?.[nodeId];
        executions = nodeData?.executions || [];
      } catch {
        executions = [];
      }
      const execution = executionId
        ? executions.find((e) => e.id === executionId)
        : executions.find((e) => e.state === "STATE_STARTED") || executions[0];
      if (!execution?.id) return;
      try {
        await workflowsInvokeNodeExecutionAction(
          withOrganizationHeader({
            path: {
              workflowId,
              executionId: execution.id,
              actionName: "passThrough",
            },
            body: { parameters: {} },
          }),
        );
        // Invalidate and force-refresh node data so sidebar updates immediately
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
