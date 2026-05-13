import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { canvasesInvokeNodeExecutionAction } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export type ApprovalActionName = "approve" | "reject";

export interface ApprovalActionParams {
  index: number;
  comment?: string;
  reason?: string;
}

/**
 * Returns a handler that invokes approve/reject on an approval execution,
 * then invalidates the per-node execution + runs queries so the Run View
 * and Runs list immediately pick up the new state. Used by the Activity
 * section to offer inline Approve/Reject for any approval waiting on the
 * current user.
 */
export function useApprovalActionHandler(canvasId: string | undefined) {
  const queryClient = useQueryClient();

  return useCallback(
    async (nodeId: string, executionId: string, actionName: ApprovalActionName, parameters: ApprovalActionParams) => {
      if (!canvasId) return;
      try {
        await canvasesInvokeNodeExecutionAction(
          withOrganizationHeader({
            path: { canvasId, executionId, actionName },
            body: { parameters: parameters as unknown as Record<string, unknown> },
          }),
        );
        queryClient.invalidateQueries({ queryKey: canvasKeys.nodeExecution(canvasId, nodeId) });
        queryClient.invalidateQueries({ queryKey: canvasKeys.runs() });
        showSuccessToast(actionName === "approve" ? "Approval recorded" : "Rejection recorded");
      } catch (error) {
        const fallback = actionName === "approve" ? "failed to approve" : "failed to reject";
        showErrorToast(getApiErrorMessage(error, fallback));
      }
    },
    [canvasId, queryClient],
  );
}
