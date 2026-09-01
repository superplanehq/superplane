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
    return "Task closed as completed.";
  }
  if (choice === "canceled") {
    return "Task closed as rejected.";
  }
  if (choice === "reopen") {
    return "Task reopened.";
  }
  if (choice === "rerun-start") {
    return "Task started from the first step.";
  }
  if (choice === "rerun-step") {
    return "Task step started again.";
  }
  return "Task closed as failed.";
}

function rejectToast(): string {
  return closeToast("canceled");
}

function stopErrorFallback(choice: SplitRunStopChoice, footer: StopFooter): string {
  if (choice === "reopen") {
    return "Failed to reopen task";
  }
  if (isSplitRunRerunChoice(choice)) {
    return "Failed to start the task";
  }
  if (footer.kind === "running" && footer.run) {
    return "Failed to stop the run";
  }
  return "Failed to close task";
}

function useSplitRunCancelRun(organizationId?: string, factoryId?: string, orderId?: string) {
  const queryClient = useQueryClient();
  return useMutation({
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
}

export function useSplitRunFooterActions(organizationId?: string, factoryId?: string, orderId?: string) {
  const closeWorkOrder = useCloseWorkOrder(organizationId ?? "", factoryId ?? "");
  const updateStatus = useUpdateWorkOrderStatus(organizationId ?? "", factoryId ?? "");
  const dispatchWorkOrder = useDispatchWorkOrder(organizationId ?? "", factoryId ?? "");
  const live = Boolean(organizationId && factoryId && orderId);
  const cancelRun = useSplitRunCancelRun(organizationId, factoryId, orderId);

  const busy =
    cancelRun.isPending ||
    closeWorkOrder.isPending ||
    updateStatus.isPending ||
    dispatchWorkOrder.isPending;

  const handleBackToDraft = useCallback(async () => {
    if (!live || !orderId || busy) {
      return false;
    }
    try {
      await updateStatus.mutateAsync({ orderId, state: "STATE_DRAFT" });
      showSuccessToast("Task returned to the Backlog.");
      return true;
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to return the task to the Backlog"));
      return false;
    }
  }, [busy, live, orderId, updateStatus]);

  const handleReject = useCallback(async () => {
    if (!live || !orderId || busy) {
      return false;
    }
    try {
      await closeWorkOrder.mutateAsync({ orderId, result: "RESULT_REJECTED" });
      showSuccessToast(rejectToast());
      return true;
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to close task"));
      return false;
    }
  }, [busy, closeWorkOrder, live, orderId]);

  const handleStop = useCallback(
    async (choice: SplitRunStopChoice, footer: StopFooter) => {
      if (!live || !orderId || busy) {
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
              throw new Error("A factory line is required to rerun this task");
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
    [busy, cancelRun, closeWorkOrder, dispatchWorkOrder, live, orderId, updateStatus],
  );

  const handleStopAutomation = useCallback(
    async (run: SplitRunStopRun) => {
      if (!live || busy) {
        return;
      }
      try {
        await stopSplitRunAutomation(run, (next) => cancelRun.mutateAsync(next));
        showSuccessToast("Automation stopped.");
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, "Failed to stop the run"));
      }
    },
    [busy, cancelRun, live],
  );

  return {
    handleStop,
    handleStopAutomation,
    handleReject,
    handleBackToDraft,
    busy,
  };
}

export type SplitRunFooterActions = ReturnType<typeof useSplitRunFooterActions>;
