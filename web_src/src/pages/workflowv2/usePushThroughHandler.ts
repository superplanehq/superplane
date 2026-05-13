import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { canvasesInvokeNodeExecutionAction } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

/**
 * Returns a handler that invokes the "pushThrough" action on a given
 * execution. Invalidates the per-node execution query so the canvas/run
 * view reflects the new state. Used by the Run View to surface inline
 * "Push Through" buttons for waiting components (time gate, wait, etc).
 */
export function usePushThroughHandler(canvasId: string | undefined) {
  const queryClient = useQueryClient();

  return useCallback(
    async (nodeId: string, executionId: string) => {
      if (!canvasId) return;
      try {
        await canvasesInvokeNodeExecutionAction(
          withOrganizationHeader({
            path: { canvasId, executionId, actionName: "pushThrough" },
            body: { parameters: null },
          }),
        );
        queryClient.invalidateQueries({ queryKey: canvasKeys.nodeExecution(canvasId, nodeId) });
        queryClient.invalidateQueries({ queryKey: canvasKeys.runs() });
        showSuccessToast("Execution pushed through");
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, "failed to push through execution"));
      }
    },
    [canvasId, queryClient],
  );
}
