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
  const getNodeData = useNodeExecutionStore((state) => state.getNodeData);

  const isUuid = (value?: string) =>
    typeof value === "string" &&
    /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$/.test(value);

  const onPassThrough = useCallback(
    async (nodeId: string, incomingExecutionId: string) => {
      try {
        // Resolve a valid executionId (UUID) robustly
        let executionId = incomingExecutionId;
        if (!isUuid(executionId)) {
          // Try to find a running execution for this node from the store
          const nodeData = useNodeExecutionStore.getState().getNodeData(nodeId);
          const running = (nodeData.executions || []).find((e) => e.state === "STATE_STARTED");
          if (isUuid(running?.id)) {
            executionId = running!.id!;
          }
        }

        if (!isUuid(executionId)) {
          console.error("onPassThrough: invalid executionId", { nodeId, incomingExecutionId, resolved: executionId });
          showErrorToast("Failed to push through: missing execution ID");
          return;
        }

        await workflowsInvokeNodeExecutionAction(
          withOrganizationHeader({
            path: {
              workflowId,
              executionId,
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
    [workflowId, organizationId, queryClient, workflow?.spec?.nodes, refetchNodeData, getNodeData],
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
