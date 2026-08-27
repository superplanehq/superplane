import { canvasesCancelRun } from "@/api-client";
import {
  factoryQueryKeys,
  useCloseWorkOrder,
  useDispatchWorkOrder,
  useUpdateWorkOrderStatus,
} from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import type { SplitRunFooter, SplitRunStopChoice } from "./splitRunFooter";
import { isSplitRunRerunChoice, rerunStartStepIndex } from "./splitRunFooter";
import { applySplitRunStop, stopSplitRunAutomation, type SplitRunStopRun } from "./splitRunStop";

type StopFooter = Pick<SplitRunFooter, "kind" | "run" | "status"> & {
  lineName?: string;
  stepIndex?: number;
};

function closeToast(choice: SplitRunStopChoice): string {
  if (choice === "completed") {
    return "Work order closed as completed.";
  }
  if (choice === "canceled") {
    return "Work order closed as rejected.";
  }
  if (choice === "reopen") {
    return "Work order reopened.";
  }
  if (choice === "rerun-start") {
    return "Work order started from the first step.";
  }
  if (choice === "rerun-step") {
    return "Work order step started again.";
  }
  return "Work order closed as failed.";
}

function rejectToast(): string {
  return closeToast("canceled");
}

function stopErrorFallback(choice: SplitRunStopChoice, footer: StopFooter): string {
  if (choice === "reopen") {
    return "Failed to reopen work order";
  }
  if (isSplitRunRerunChoice(choice)) {
    return "Failed to start the work order";
  }
  if (footer.kind === "running" && footer.run) {
    return "Failed to stop the run";
  }
  return "Failed to close work order";
}

export function useSplitRunFooterActions(organizationId?: string, factoryId?: string, orderId?: string) {
  const queryClient = useQueryClient();
  const closeWorkOrder = useCloseWorkOrder(organizationId ?? "", factoryId ?? "");
  const updateStatus = useUpdateWorkOrderStatus(organizationId ?? "", factoryId ?? "");
  const dispatchWorkOrder = useDispatchWorkOrder(organizationId ?? "", factoryId ?? "");
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
      if (!organizationId || !factoryId) {
        return;
      }
      await queryClient.invalidateQueries({ queryKey: factoryQueryKeys.workOrders(organizationId, factoryId) });
      if (orderId) {
        await queryClient.invalidateQueries({
          queryKey: factoryQueryKeys.workOrderDetail(organizationId, factoryId, orderId),
        });
      }
    },
  });

  const handleReject = useCallback(async () => {
    if (!live || !orderId) {
      return false;
    }
    try {
      await closeWorkOrder.mutateAsync({ orderId, result: "RESULT_REJECTED" });
      showSuccessToast(rejectToast());
      return true;
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to close work order"));
      return false;
    }
  }, [closeWorkOrder, live, orderId]);

  const handleStop = useCallback(
    async (choice: SplitRunStopChoice, footer: StopFooter) => {
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
            showSuccessToast(closeToast(choice));
          },
          onStatusChange: async (state) => {
            await updateStatus.mutateAsync({ orderId, state });
            showSuccessToast(closeToast(choice));
          },
          onRerun: async (rerunChoice) => {
            const lineName = footer.lineName?.trim();
            if (!lineName) {
              throw new Error("A factory line is required to rerun this work order");
            }
            await dispatchWorkOrder.mutateAsync({
              orderId,
              lineName,
              startStepIndex: rerunStartStepIndex(rerunChoice, footer.stepIndex),
              replaceActive: true,
            });
            showSuccessToast(closeToast(rerunChoice));
          },
        });
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, stopErrorFallback(choice, footer)));
      }
    },
    [cancelRun, closeWorkOrder, dispatchWorkOrder, live, orderId, updateStatus],
  );

  const handleStopAutomation = useCallback(
    async (run: SplitRunStopRun) => {
      if (!live) {
        return;
      }
      try {
        await stopSplitRunAutomation(run, (next) => cancelRun.mutateAsync(next));
        showSuccessToast("Automation stopped.");
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, "Failed to stop the run"));
      }
    },
    [cancelRun, live],
  );

  return {
    handleStop,
    handleStopAutomation,
    handleReject,
    busy: cancelRun.isPending || closeWorkOrder.isPending || updateStatus.isPending || dispatchWorkOrder.isPending,
  };
}
