import type { FactoriesWorkOrder, FactoriesWorkOrderExecution } from "@/api-client";
import { formatRelativeTime } from "@/lib/timezone";

import { flattenWorkOrderExecutions, getExecutionStepTimestamp } from "./workOrderExecutions";

export type ColumnAutomationLastRunStatus = "passed" | "failed";

export type ColumnAutomationActivity = {
  lastRunStatus?: ColumnAutomationLastRunStatus;
  lastRunWhen?: string;
  runningCount: number;
};

type FinishedOutcome = {
  status: ColumnAutomationLastRunStatus;
  at: number;
  when: string;
};

export function emptyColumnAutomationActivity(runningCount = 0): ColumnAutomationActivity {
  return { runningCount };
}

export function buildColumnAutomationActivity(input: {
  canvasId?: string;
  workOrders: FactoriesWorkOrder[];
  runningCount: number;
}): ColumnAutomationActivity {
  const canvasId = input.canvasId?.trim();
  if (!canvasId) {
    return emptyColumnAutomationActivity(input.runningCount);
  }

  let lastRun: FinishedOutcome | undefined;
  for (const execution of executionsForApp(canvasId, input.workOrders)) {
    const outcome = finishedOutcome(execution);
    if (!outcome) {
      continue;
    }
    if (!lastRun || outcome.at > lastRun.at) {
      lastRun = outcome;
    }
  }

  return {
    lastRunStatus: lastRun?.status,
    lastRunWhen: lastRun?.when,
    runningCount: input.runningCount,
  };
}

function executionsForApp(canvasId: string, workOrders: FactoriesWorkOrder[]): FactoriesWorkOrderExecution[] {
  return workOrders.flatMap((order) =>
    flattenWorkOrderExecutions(order).filter((execution) => execution.run?.appId === canvasId),
  );
}

function finishedOutcome(execution: FactoriesWorkOrderExecution): FinishedOutcome | undefined {
  if (execution.state !== "STATE_FINISHED") {
    return undefined;
  }
  const status = resultStatus(execution.result);
  if (!status) {
    return undefined;
  }
  const when = execution.finishedAt ?? getExecutionStepTimestamp(execution);
  const at = Date.parse(when);
  if (!Number.isFinite(at)) {
    return undefined;
  }
  return { status, at, when: formatRelativeTime(when) };
}

function resultStatus(result: FactoriesWorkOrderExecution["result"]): ColumnAutomationLastRunStatus | undefined {
  if (result === "RESULT_PASSED") {
    return "passed";
  }
  if (result === "RESULT_FAILED") {
    return "failed";
  }
  return undefined;
}
