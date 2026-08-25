import type { FactoriesWorkOrderResult, FactoriesWorkOrderState } from "@/api-client";

import type { SplitRunFooterKind, SplitRunStopChoice } from "./splitRunFooter";

export type SplitRunStopRun = { appId: string; runId: string };

/**
 * Stop as Canceled is the former Reject close. Completed and Draft keep
 * the same close and status endpoints.
 */
export async function applySplitRunStopChoice(
  choice: SplitRunStopChoice,
  handlers: {
    onClose: (result: FactoriesWorkOrderResult) => void | Promise<void>;
    onStatusChange: (state: FactoriesWorkOrderState) => void | Promise<void>;
  },
): Promise<void> {
  if (choice === "canceled") {
    await handlers.onClose("RESULT_REJECTED");
    return;
  }
  if (choice === "completed") {
    await handlers.onClose("RESULT_COMPLETED");
    return;
  }
  await handlers.onStatusChange("STATE_DRAFT");
}

/**
 * When a line run is in flight, cancel that canvas run first, then apply
 * the work-order outcome. Waiting and failed orders skip cancel.
 */
export async function applySplitRunStop(
  choice: SplitRunStopChoice,
  input: {
    kind: SplitRunFooterKind;
    run?: SplitRunStopRun;
    cancelRun?: (run: SplitRunStopRun) => Promise<void>;
    onClose: (result: FactoriesWorkOrderResult) => void | Promise<void>;
    onStatusChange: (state: FactoriesWorkOrderState) => void | Promise<void>;
  },
): Promise<void> {
  if (input.kind === "running" && input.run && input.cancelRun) {
    await input.cancelRun(input.run);
  }
  await applySplitRunStopChoice(choice, input);
}
