import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import { canvasesInvokeNodeExecutionAction, CanvasesCanvas } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";

type Params = {
  canvasId: string;
  organizationId?: string;
  workflow?: CanvasesCanvas | null;
};

export function usePushThroughHandler({ canvasId, organizationId, workflow }: Params) {
  const queryClient = useQueryClient();
  const refetchNodeData = useNodeExecutionStore((state) => state.refetchNodeData);
  const getNodeData = useNodeExecutionStore((state) => state.getNodeData);

  const onPushThrough = useCallback(
    async (nodeId: string, executionId: string) => {
      try {
        await canvasesInvokeNodeExecutionAction(
          withOrganizationHeader({
            path: {
              canvasId,
              executionId,
              actionName: "pushThrough",
            },
            body: { parameters: {} },
          }),
        );

        await queryClient.invalidateQueries({ queryKey: canvasKeys.nodeExecution(canvasId, nodeId) });
        const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);

        if (node) {
          await refetchNodeData(canvasId, nodeId, node.type!, queryClient);
        }

        showSuccessToast("Pushed through");
      } catch (error) {
        console.error("Failed to push through", error);
        showErrorToast("Failed to push through");
      }
    },
    [canvasId, organizationId, queryClient, workflow?.spec?.nodes, refetchNodeData, getNodeData],
  );

  const supportsPushThrough = useCallback(
    (nodeId: string) => {
      const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);
      const name = node?.component?.name;
      return name === "wait" || name === "timeGate";
    },
    [workflow],
  );

  return { onPushThrough, supportsPushThrough } as const;
}
