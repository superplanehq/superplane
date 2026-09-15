import type {
  FactoriesWorkOrder,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderExecutionResult,
  FactoriesWorkOrderExecutionState,
  FactoriesWorkOrderLineDispatch,
  FactoriesWorkOrderQueueItem,
} from "@/api-client";
import { factoryAppRunPath } from "./factoryPagePaths";

/**
 * One row in a task's step activity: either a real execution or a
 * queue item projected into the same shape. `queuePosition` is set only
 * for queued rows (1 is next to be admitted).
 */
export type WorkOrderStepRow = FactoriesWorkOrderExecution & {
  queuePosition?: number;
};

export function isQueuedStepRow(row: WorkOrderStepRow): boolean {
  return row.queuePosition !== undefined;
}

/** 1 is next to be admitted. Position 0 (or missing) is queued without a place. */
export function queuePositionLabel(position: number | undefined): string {
  return (position ?? 0) > 0 ? `Queued #${position}` : "Queued";
}

/**
 * True when this dispatch waits to enter the first line step and has not
 * started a run yet. Those tasks stay in Backlog until a slot is free.
 */
export function isFirstStepQueueOnly(dispatch: FactoriesWorkOrderLineDispatch): boolean {
  if (!dispatch.queueItem) {
    return false;
  }
  if ((dispatch.queueItem.stepIndex ?? 0) !== 0) {
    return false;
  }
  return (dispatch.stepExecutions ?? []).length === 0;
}

/** Queue item for a task that waits to enter the first line step. */
export function firstStepQueueItem(order: FactoriesWorkOrder): FactoriesWorkOrderQueueItem | undefined {
  const queued = (order.lineDispatches ?? []).filter(isFirstStepQueueOnly);
  if (queued.length === 0) {
    return undefined;
  }
  return queued.reduce((latest, candidate) => {
    const latestAt = Date.parse(latest.createdAt ?? "") || 0;
    const candidateAt = Date.parse(candidate.createdAt ?? "") || 0;
    return candidateAt >= latestAt ? candidate : latest;
  }).queueItem;
}

export function firstStepQueueLabel(order: FactoriesWorkOrder): string | null {
  const item = firstStepQueueItem(order);
  if (!item) {
    return null;
  }
  return queuePositionLabel(item.position);
}

/** Latest dispatch on this line waits to enter step 0 and has not started. */
export function isQueuedAtFirstLineStep(order: FactoriesWorkOrder, lineId: string): boolean {
  const dispatches = (order.lineDispatches ?? []).filter((dispatch) => dispatch.line?.id === lineId);
  if (dispatches.length === 0) {
    return false;
  }
  const latest = dispatches.reduce((best, candidate) => {
    const bestAt = Date.parse(best.createdAt ?? "") || 0;
    const candidateAt = Date.parse(candidate.createdAt ?? "") || 0;
    return candidateAt >= bestAt ? candidate : best;
  });
  return isFirstStepQueueOnly(latest);
}

/** Drafts, plus open tasks that wait to enter the first step of `lineId`. */
export function isLineBoardBacklogOrder(order: FactoriesWorkOrder, lineId?: string): boolean {
  if (!order.id) {
    return false;
  }
  if (order.state === "STATE_DRAFT") {
    return true;
  }
  if (order.state !== "STATE_OPEN") {
    return false;
  }
  if (lineId) {
    return isQueuedAtFirstLineStep(order, lineId);
  }
  return (order.lineDispatches ?? []).some(isFirstStepQueueOnly);
}

export function queueItemToStepRow(item: FactoriesWorkOrderQueueItem): WorkOrderStepRow {
  return {
    id: item.id,
    step: item.stepName,
    stepIndex: item.stepIndex,
    createdAt: item.createdAt,
    // No run exists for a queued step yet; project the step's app into the
    // run shape so board-column matching and card links treat queued rows
    // like execution rows.
    run: item.appId ? { appId: item.appId } : undefined,
    queuePosition: item.position ?? 0,
  };
}

/**
 * A dispatch's step activity as one chronological list: its step
 * executions, plus a projected row for the step it is queued at, if any.
 */
export function dispatchStepRows(dispatch: FactoriesWorkOrderLineDispatch): WorkOrderStepRow[] {
  const rows: WorkOrderStepRow[] = [...(dispatch.stepExecutions ?? [])];
  if (dispatch.queueItem) {
    rows.push(queueItemToStepRow(dispatch.queueItem));
  }
  return rows;
}

export interface WorkOrderExecutionDisplayMeta {
  label: string;
  className: string;
  isActive: boolean;
}

const EXECUTION_STATE_META: Record<
  FactoriesWorkOrderExecutionState,
  Omit<WorkOrderExecutionDisplayMeta, "isActive">
> = {
  STATE_UNKNOWN: {
    label: "Unknown",
    className: "border-gray-200 bg-gray-50 text-gray-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300",
  },
  STATE_PENDING: {
    label: "Pending",
    className:
      "border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200",
  },
  STATE_STARTED: {
    label: "Running",
    className:
      "border-violet-200 bg-violet-50 text-violet-800 dark:border-violet-500/30 dark:bg-violet-500/10 dark:text-violet-200",
  },
  STATE_CANCELLING: {
    label: "Cancelling",
    className:
      "border-orange-200 bg-orange-50 text-orange-800 dark:border-orange-500/30 dark:bg-orange-500/10 dark:text-orange-200",
  },
  STATE_FINISHED: {
    label: "Finished",
    className:
      "border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200",
  },
};

const EXECUTION_RESULT_META: Record<
  FactoriesWorkOrderExecutionResult,
  Omit<WorkOrderExecutionDisplayMeta, "isActive">
> = {
  RESULT_UNKNOWN: EXECUTION_STATE_META.STATE_UNKNOWN,
  RESULT_PASSED: {
    label: "Passed",
    className:
      "border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200",
  },
  RESULT_FAILED: {
    label: "Failed",
    className: "border-red-200 bg-red-50 text-red-800 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-200",
  },
  RESULT_CANCELLED: {
    label: "Cancelled",
    className: "border-gray-200 bg-gray-50 text-gray-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300",
  },
};

/**
 * Build a link to the automation run that produced a piece of work-order
 * timeline content (a step execution, a comment, a status change, ...).
 * Centralizes the `appId` + `runId` → app-run-page URL construction so every
 * timeline surface links to runs the same way.
 */
export function getWorkOrderRunHref(
  organizationId: string,
  factoryKey: string,
  appId: string | undefined,
  runId: string | undefined,
  options?: { orderNumber?: string },
): string | null {
  if (!appId || !runId) {
    return null;
  }

  return factoryAppRunPath(organizationId, factoryKey, appId, runId, {
    from: "task",
    orderNumber: options?.orderNumber,
  });
}

export function getWorkOrderExecutionRunHref(
  organizationId: string,
  factoryKey: string,
  execution: FactoriesWorkOrderExecution,
  options?: { orderNumber?: string },
): string | null {
  return getWorkOrderRunHref(organizationId, factoryKey, execution.run?.appId, execution.run?.id, options);
}

export function getExecutionStepTimestamp(execution: FactoriesWorkOrderExecution): string {
  if (
    execution.state === "STATE_FINISHED" ||
    execution.result === "RESULT_PASSED" ||
    execution.result === "RESULT_FAILED" ||
    execution.result === "RESULT_CANCELLED"
  ) {
    return execution.updatedAt ?? execution.createdAt ?? "";
  }

  return execution.createdAt ?? execution.updatedAt ?? "";
}

export function isActiveWorkOrderExecution(execution: WorkOrderStepRow): boolean {
  return (
    isQueuedStepRow(execution) ||
    execution.state === "STATE_PENDING" ||
    execution.state === "STATE_STARTED" ||
    execution.state === "STATE_CANCELLING"
  );
}

export function getWorkOrderExecutionDisplayMeta(
  execution: FactoriesWorkOrderExecution,
): WorkOrderExecutionDisplayMeta {
  if (execution.state === "STATE_FINISHED" && execution.result && execution.result !== "RESULT_UNKNOWN") {
    return {
      ...EXECUTION_RESULT_META[execution.result],
      isActive: false,
    };
  }

  const state = execution.state ?? "STATE_UNKNOWN";
  return {
    ...EXECUTION_STATE_META[state],
    isActive: isActiveWorkOrderExecution(execution),
  };
}

/**
 * Flattens every step execution across every line dispatch on a work
 * order, for callers (like the event-driven timeline) that just need "all
 * step executions", not the structural per-traversal grouping.
 */
export function flattenWorkOrderExecutions(order: FactoriesWorkOrder): FactoriesWorkOrderExecution[] {
  return (order.lineDispatches ?? []).flatMap((dispatch) => dispatch.stepExecutions ?? []);
}
