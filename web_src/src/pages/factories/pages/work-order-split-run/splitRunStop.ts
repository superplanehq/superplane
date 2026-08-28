import type { FactoriesWorkOrderResult, FactoriesWorkOrderState } from "@/api-client";

import type { WorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import {
  isClosedWorkOrderDisplayStatus,
  isSplitRunRerunChoice,
  type SplitRunFooterKind,
  type SplitRunStopChoice,
} from "./splitRunFooter";

export type SplitRunStopRun = { appId: string; runId: string };

type SplitRunStopHandlers = {
  onClose: (result: FactoriesWorkOrderResult) => void | Promise<void>;
  onStatusChange: (state: FactoriesWorkOrderState) => void | Promise<void>;
  onRerun?: (choice: "rerun-step" | "rerun-start") => void | Promise<void>;
  status?: WorkOrderDisplayStatus;
};

/**
 * Stop and Close is the former Reject close. Completed closes. Rerun
 * starts the current step or the first step again.
 */
export async function applySplitRunStopChoice(
  choice: SplitRunStopChoice,
  handlers: SplitRunStopHandlers,
): Promise<void> {
  if (choice === "canceled") {
    await handlers.onClose("RESULT_REJECTED");
    return;
  }
  if (choice === "completed") {
    await handlers.onClose("RESULT_COMPLETED");
    return;
  }
  if (isSplitRunRerunChoice(choice)) {
    await handlers.onRerun?.(choice);
    return;
  }
  if (choice === "reopen" || isClosedWorkOrderDisplayStatus(handlers.status)) {
    await handlers.onStatusChange("STATE_OPEN");
  }
}

/**
 * When a line run is in flight, cancel that canvas run first, then apply
 * the work-order outcome. Waiting and failed orders skip cancel.
 */
export async function stopSplitRunAutomation(
  run: SplitRunStopRun,
  cancelRun: (run: SplitRunStopRun) => Promise<void>,
): Promise<void> {
  await cancelRun(run);
}

export async function applySplitRunStop(
  choice: SplitRunStopChoice,
  input: {
    kind: SplitRunFooterKind;
    run?: SplitRunStopRun;
    cancelRun?: (run: SplitRunStopRun) => Promise<void>;
    onClose: (result: FactoriesWorkOrderResult) => void | Promise<void>;
    onStatusChange: (state: FactoriesWorkOrderState) => void | Promise<void>;
    onRerun?: (choice: "rerun-step" | "rerun-start") => void | Promise<void>;
    status?: WorkOrderDisplayStatus;
  },
): Promise<void> {
  if (input.kind === "running" && input.run && input.cancelRun) {
    await input.cancelRun(input.run);
  }
  await applySplitRunStopChoice(choice, input);
}
