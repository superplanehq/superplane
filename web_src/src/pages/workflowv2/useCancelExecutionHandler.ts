import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import { canvasesCancelExecution, CanvasesCanvas } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";

type Params = {
  canvasId: string;
  workflow?: CanvasesCanvas | null;
};

export function useCancelExecutionHandler({ canvasId, workflow }: Params) {
  const queryClient = useQueryClient();
  const refetchNodeData = useNodeExecutionStore((state) => state.refetchNodeData);

  const onCancelExecution = useCallback(
    async (nodeId: string, executionId: string) => {
      try {
        await canvasesCancelExecution(
          withOrganizationHeader({
            path: {
              canvasId,
              executionId,
            },
          }),
        );

        await queryClient.invalidateQueries({ queryKey: canvasKeys.nodeExecution(canvasId, nodeId) });
        const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);

        if (node) {
          await refetchNodeData(canvasId, nodeId, node.type!, queryClient);
        }

        showSuccessToast("Execution cancelled");
      } catch (error) {
        console.error("Failed to cancel execution", error);
        showErrorToast("Failed to cancel execution");
      }
    },
    [canvasId, queryClient, workflow?.spec?.nodes, refetchNodeData],
  );

  return { onCancelExecution } as const;
}
