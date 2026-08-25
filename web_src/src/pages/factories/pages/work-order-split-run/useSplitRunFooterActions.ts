import { canvasesCancelRun } from "@/api-client";
import { useCloseWorkOrder, useUpdateWorkOrderStatus } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import type { SplitRunFooter, SplitRunStopChoice } from "./splitRunFooter";
import { isClosedWorkOrderDisplayStatus } from "./splitRunFooter";
import { applySplitRunStop, type SplitRunStopRun } from "./splitRunStop";

function closeToast(choice: SplitRunStopChoice, status?: SplitRunFooter["status"]): string {
  if (choice === "completed") {
    return "Work order closed as completed.";
  }
  if (choice === "reopen" || (choice === "draft" && isClosedWorkOrderDisplayStatus(status))) {
    return "Work order reopened.";
  }
  if (choice === "draft") {
    return "Work order moved to draft.";
  }
  return "Work order closed as canceled.";
}

export function useSplitRunFooterActions(organizationId?: string, factoryId?: string, orderId?: string) {
  const queryClient = useQueryClient();
  const closeWorkOrder = useCloseWorkOrder(organizationId ?? "", factoryId ?? "");
  const updateStatus = useUpdateWorkOrderStatus(organizationId ?? "", factoryId ?? "");
  const live = Boolean(organizationId && factoryId && orderId);
  const cancelRun = useMutation({
    mutationFn: async (run: SplitRunStopRun) => {
      await canvasesCancelRun(
        withOrganizationHeader({
          organizationId,
          path: { canvasId: run.appId, runId: run.runId },
        }),
      );
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["canvases"] });
    },
  });

  const handleReject = useCallback(async () => {
    if (!live || !orderId) {
      return;
    }
    try {
      await closeWorkOrder.mutateAsync({ orderId, result: "RESULT_REJECTED" });
      showSuccessToast("Work order closed as canceled.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to close work order"));
    }
  }, [closeWorkOrder, live, orderId]);

  const handleStop = useCallback(
    async (choice: SplitRunStopChoice, footer: Pick<SplitRunFooter, "kind" | "run" | "status">) => {
      if (!live || !orderId) {
        return;
      }
      try {
        await applySplitRunStop(choice, {
          kind: footer.kind,
          run: footer.run,
          status: footer.status,
          cancelRun: (run) => cancelRun.mutateAsync(run),
          onClose: async (result) => {
            await closeWorkOrder.mutateAsync({ orderId, result });
            showSuccessToast(closeToast(choice, footer.status));
          },
          onStatusChange: async (state) => {
            await updateStatus.mutateAsync({ orderId, state });
            showSuccessToast(closeToast(choice, footer.status));
          },
        });
      } catch (error) {
        const fallback =
          choice === "reopen" || (choice === "draft" && isClosedWorkOrderDisplayStatus(footer.status))
            ? "Failed to reopen work order"
            : footer.kind === "running" && footer.run
              ? "Failed to stop the run"
              : "Failed to close work order";
        showErrorToast(getApiErrorMessage(error, fallback));
      }
    },
    [cancelRun, closeWorkOrder, live, orderId, updateStatus],
  );

  return {
    handleStop,
    handleReject,
    busy: cancelRun.isPending || closeWorkOrder.isPending || updateStatus.isPending,
  };
}
