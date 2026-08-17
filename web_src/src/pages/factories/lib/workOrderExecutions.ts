import type {
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderExecutionResult,
  FactoriesWorkOrderExecutionState,
} from "@/api-client";
import { factoryAppRunPath } from "./factoryPagePaths";

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
    from: "work-order",
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

export function isActiveWorkOrderExecution(execution: FactoriesWorkOrderExecution): boolean {
  return (
    execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED" || execution.state === "STATE_CANCELLING"
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

export interface WorkOrderExecutionLineGroup {
  lineId: string;
  lineName: string;
  executions: FactoriesWorkOrderExecution[];
}

export function groupWorkOrderExecutionsByLine(
  executions: FactoriesWorkOrderExecution[] | undefined,
): WorkOrderExecutionLineGroup[] {
  if (!executions?.length) {
    return [];
  }

  const groups = new Map<string, WorkOrderExecutionLineGroup>();
  for (const execution of executions) {
    const lineId = execution.line?.id ?? "unknown";
    const lineName = execution.line?.name?.trim() || "Unnamed line";
    const existing = groups.get(lineId);
    if (existing) {
      existing.executions.push(execution);
      continue;
    }

    groups.set(lineId, {
      lineId,
      lineName,
      executions: [execution],
    });
  }

  return [...groups.values()].map((group) => ({
    ...group,
    executions: [...group.executions].sort(compareExecutionsChronologically),
  }));
}

function compareExecutionsChronologically(
  left: FactoriesWorkOrderExecution,
  right: FactoriesWorkOrderExecution,
): number {
  const leftTime = Date.parse(left.createdAt ?? "") || 0;
  const rightTime = Date.parse(right.createdAt ?? "") || 0;
  if (leftTime !== rightTime) {
    return leftTime - rightTime;
  }

  return (left.id ?? "").localeCompare(right.id ?? "");
}

export function hasFailedWorkOrderExecution(executions: FactoriesWorkOrderExecution[] | undefined): boolean {
  return (executions ?? []).some((execution) => execution.result === "RESULT_FAILED");
}

function executionTimestamp(execution: FactoriesWorkOrderExecution): number {
  return Date.parse(execution.updatedAt ?? execution.createdAt ?? "") || 0;
}

// Most recent finished execution (by `updatedAt`, fallback `createdAt`).
// Callers fence against `order.updatedAt` to handle reopens.
export function latestFinishedWorkOrderExecution(
  executions: FactoriesWorkOrderExecution[] | undefined,
): FactoriesWorkOrderExecution | null {
  let latest: FactoriesWorkOrderExecution | null = null;
  let latestAt = -Infinity;
  for (const execution of executions ?? []) {
    if (execution.state !== "STATE_FINISHED") {
      continue;
    }
    if (!execution.result || execution.result === "RESULT_UNKNOWN") {
      continue;
    }
    const at = executionTimestamp(execution);
    if (at >= latestAt) {
      latest = execution;
      latestAt = at;
    }
  }
  return latest;
}

export function hasActiveWorkOrderExecution(executions: FactoriesWorkOrderExecution[] | undefined): boolean {
  return (executions ?? []).some(isActiveWorkOrderExecution);
}
