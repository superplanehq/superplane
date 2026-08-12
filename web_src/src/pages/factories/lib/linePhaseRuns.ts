import type { FactoriesFactoryLine, FactoriesWorkOrder, FactoriesWorkOrderExecution } from "@/api-client";
import { isActiveWorkOrderExecution } from "./workOrderExecutions";

export type LinePhaseTick = "running" | "waiting" | "queued" | null;

export type LinePhaseRunCard = {
  executionId: string;
  workOrderId: string;
  title: string;
  execution: FactoriesWorkOrderExecution;
};

export type LinePhaseColumn = {
  stepName: string;
  stepIndex: number;
  /** Factory app id for this runApp step, when present. */
  appId?: string;
  runs: LinePhaseRunCard[];
  tick: LinePhaseTick;
};

/** Initial / step size for the scrollable phase-column run list. */
export const LINE_PHASE_RUNS_PAGE_SIZE = 3;

/**
 * Builds the Lines detail board: one column per line step, each with every
 * canvas-run execution for that step (matched by line id + step name), newest
 * first. The detail UI pages these into a scrollable column.
 */
export function buildLinePhaseBoard(line: FactoriesFactoryLine, workOrders: FactoriesWorkOrder[]): LinePhaseColumn[] {
  const lineId = line.id;
  const steps = line.steps ?? [];
  if (!lineId || steps.length === 0) {
    return [];
  }

  const runsByStep = collectRunsByStep(lineId, workOrders);

  return steps.map((step, stepIndex) => {
    const stepName = step.name?.trim() || `Phase ${stepIndex + 1}`;
    const runs = runsByStep.get(step.name ?? "") ?? [];
    const appId = step.app?.app?.trim() || undefined;
    return {
      stepName,
      stepIndex,
      appId,
      runs,
      tick: resolvePhaseTick(runs),
    };
  });
}

export function resolvePhaseRunStatus(execution: FactoriesWorkOrderExecution): {
  kind: "running" | "waiting" | "queued" | "idle";
  label: string;
} {
  if (execution.state === "STATE_STARTED") {
    return { kind: "running", label: "Executing" };
  }
  if (execution.state === "STATE_CANCELLING") {
    // In-flight like Automations (running tick), but keep Cancelling label.
    return { kind: "running", label: "Cancelling" };
  }
  if (execution.state === "STATE_PENDING") {
    return { kind: "queued", label: "Queued" };
  }
  if (execution.state === "STATE_FINISHED") {
    if (execution.result === "RESULT_PASSED") {
      return { kind: "idle", label: "Passed" };
    }
    if (execution.result === "RESULT_FAILED") {
      return { kind: "waiting", label: "Failed" };
    }
    if (execution.result === "RESULT_CANCELLED") {
      return { kind: "idle", label: "Cancelled" };
    }
  }
  return { kind: "idle", label: "Unknown" };
}

function collectRunsByStep(lineId: string, workOrders: FactoriesWorkOrder[]): Map<string, LinePhaseRunCard[]> {
  const runsByStep = new Map<string, LinePhaseRunCard[]>();

  for (const order of workOrders) {
    if (!order.id) {
      continue;
    }
    for (const execution of order.executions ?? []) {
      if (execution.line?.id !== lineId || !execution.step) {
        continue;
      }
      const card: LinePhaseRunCard = {
        executionId: execution.id ?? `${order.id}-${execution.step}-${execution.createdAt ?? ""}`,
        workOrderId: order.id,
        title: order.title?.trim() || "Untitled work order",
        execution,
      };
      const existing = runsByStep.get(execution.step);
      if (existing) {
        existing.push(card);
      } else {
        runsByStep.set(execution.step, [card]);
      }
    }
  }

  for (const runs of runsByStep.values()) {
    runs.sort(compareRunsNewestFirst);
  }

  return runsByStep;
}

function compareRunsNewestFirst(left: LinePhaseRunCard, right: LinePhaseRunCard): number {
  const leftTime = Date.parse(left.execution.updatedAt ?? left.execution.createdAt ?? "") || 0;
  const rightTime = Date.parse(right.execution.updatedAt ?? right.execution.createdAt ?? "") || 0;
  if (leftTime !== rightTime) {
    return rightTime - leftTime;
  }
  return (right.executionId ?? "").localeCompare(left.executionId ?? "");
}

function resolvePhaseTick(runs: LinePhaseRunCard[]): LinePhaseTick {
  let hasRunning = false;
  let hasWaiting = false;
  let hasQueued = false;

  for (const run of runs) {
    const { kind } = resolvePhaseRunStatus(run.execution);
    if (kind === "running") hasRunning = true;
    else if (kind === "waiting") hasWaiting = true;
    else if (kind === "queued") hasQueued = true;
    else if (isActiveWorkOrderExecution(run.execution)) hasQueued = true;
  }

  if (hasWaiting) return "waiting";
  if (hasRunning) return "running";
  if (hasQueued) return "queued";
  return null;
}
