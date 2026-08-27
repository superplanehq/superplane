import type { FactoriesWorkOrder, FactoriesWorkOrderPrFeedbackRun } from "@/api-client";
import { useMemo } from "react";

import { activePRFeedbackWorkOrderIds, type PRFeedbackLogRun } from "./prFeedbackSettingsModel";

export function useActivePRFeedbackWorkOrderIds(workOrders: FactoriesWorkOrder[]): ReadonlySet<string> {
  return useMemo(() => activePRFeedbackWorkOrderIds(workOrders), [workOrders]);
}

export function useWorkOrderPRFeedbackLog(order: FactoriesWorkOrder | undefined): PRFeedbackLogRun[] {
  return useMemo(() => prFeedbackLogRunsFromItems(order?.prFeedbackRuns ?? []), [order?.prFeedbackRuns]);
}

export function prFeedbackLogRunsFromItems(items: FactoriesWorkOrderPrFeedbackRun[]): PRFeedbackLogRun[] {
  return [...items]
    .sort((left, right) => Date.parse(left.run?.createdAt ?? "") - Date.parse(right.run?.createdAt ?? ""))
    .flatMap(prFeedbackLogRunFromItem);
}

function prFeedbackLogRunFromItem(item: FactoriesWorkOrderPrFeedbackRun): PRFeedbackLogRun[] {
  if (!item.canvasId || !item.run?.id) {
    return [];
  }
  return [{ canvasId: item.canvasId, handlerName: item.handlerName, run: item.run }];
}
