import { useCallback } from "react";
import type { QueryClient } from "@tanstack/react-query";

import { canvasesInvokeNodeTriggerHook, type CanvasesCanvas } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import type { DashboardTriggerOptions } from "./DashboardContext";
import { buildDashboardTriggerParameters } from "./dashboardTriggerParameters";

export function useDashboardTriggerNode({
  canvasId,
  canvas,
  queryClient,
}: {
  canvasId: string | undefined;
  canvas: CanvasesCanvas | undefined;
  queryClient: QueryClient;
}) {
  return useCallback(
    async (nodeId: string, options?: DashboardTriggerOptions) => {
      if (!canvasId) return;

      const hookName = options?.hookName ?? "run";
      const templateName = options?.templateName ?? options?.triggerName;
      const node = canvas?.spec?.nodes?.find((item) => item.id === nodeId);
      const parameters = options?.parameters ?? buildDashboardTriggerParameters(node, hookName, templateName);

      try {
        await canvasesInvokeNodeTriggerHook(
          withOrganizationHeader({
            path: { canvasId, nodeId, hookName },
            body: { parameters },
          }),
        );
        showSuccessToast(options?.successLabel ? `Triggered: ${options.successLabel}` : "Triggered node");
        invalidateDashboardTriggerQueries(queryClient, canvasId, nodeId);
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, "Failed to trigger node"));
        throw error;
      }
    },
    [canvasId, canvas, queryClient],
  );
}

function invalidateDashboardTriggerQueries(queryClient: QueryClient, canvasId: string, nodeId: string) {
  queryClient.invalidateQueries({ queryKey: canvasKeys.nodeExecution(canvasId, nodeId) });
  queryClient.invalidateQueries({ queryKey: canvasKeys.infiniteEvents(canvasId) });
  queryClient.invalidateQueries({ queryKey: canvasKeys.infiniteRuns(canvasId) });
  queryClient.invalidateQueries({ queryKey: canvasKeys.canvasMemoryEntries(canvasId) });
}
