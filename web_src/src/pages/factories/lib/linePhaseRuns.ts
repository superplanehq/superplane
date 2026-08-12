import type { FactoriesFactoryLine, FactoriesWorkOrder, FactoriesWorkOrderExecution } from "@/api-client";
import { isActiveWorkOrderExecution } from "./workOrderExecutions";

export type LinePhaseTick = "running" | "waiting" | "queued" | "failed" | null;

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
 * Builds the Lines detail board: one column per line step. Each work order
 * appears once, in the column for its current (furthest active, else furthest
 * finished) step on this line — newest cards first within a column.
 */
export function buildLinePhaseBoard(line: FactoriesFactoryLine, workOrders: FactoriesWorkOrder[]): LinePhaseColumn[] {
  const lineId = line.id;
  const steps = line.steps ?? [];
  if (!lineId || steps.length === 0) {
    return [];
  }

  const runsByStep = collectCurrentRunsByStep(lineId, steps, workOrders);

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
  kind: "running" | "waiting" | "queued" | "failed" | "idle";
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
      return { kind: "failed", label: "Failed" };
    }
    if (execution.result === "RESULT_CANCELLED") {
      return { kind: "idle", label: "Cancelled" };
    }
  }
  return { kind: "idle", label: "Unknown" };
}

function collectCurrentRunsByStep(
  lineId: string,
  steps: NonNullable<FactoriesFactoryLine["steps"]>,
  workOrders: FactoriesWorkOrder[],
): Map<string, LinePhaseRunCard[]> {
  const stepIndexByName = new Map<string, number>();
  for (const [index, step] of steps.entries()) {
    if (step.name) {
      stepIndexByName.set(step.name, index);
    }
  }

  const runsByStep = new Map<string, LinePhaseRunCard[]>();

  for (const order of workOrders) {
    if (!order.id) {
      continue;
    }

    const lineExecutions = (order.executions ?? []).filter(
      (execution) => execution.line?.id === lineId && Boolean(execution.step) && stepIndexByName.has(execution.step!),
    );
    if (lineExecutions.length === 0) {
      continue;
    }

    const currentExecution = pickCurrentLineExecution(lineExecutions, stepIndexByName);
    if (!currentExecution?.step) {
      continue;
    }

    const card: LinePhaseRunCard = {
      executionId: currentExecution.id ?? `${order.id}-${currentExecution.step}-${currentExecution.createdAt ?? ""}`,
      workOrderId: order.id,
      title: order.title?.trim() || "Untitled work order",
      execution: currentExecution,
    };
    const existing = runsByStep.get(currentExecution.step);
    if (existing) {
      existing.push(card);
    } else {
      runsByStep.set(currentExecution.step, [card]);
    }
  }

  for (const runs of runsByStep.values()) {
    runs.sort(compareRunsNewestFirst);
  }

  return runsByStep;
}

function pickCurrentLineExecution(
  executions: FactoriesWorkOrderExecution[],
  stepIndexByName: Map<string, number>,
): FactoriesWorkOrderExecution | null {
  const active = executions.filter(isActiveWorkOrderExecution);
  const candidates = active.length > 0 ? active : executions;
  let best: FactoriesWorkOrderExecution | null = null;
  let bestStepIndex = -1;
  let bestTime = -1;

  for (const execution of candidates) {
    const stepIndex = stepIndexByName.get(execution.step ?? "") ?? -1;
    if (stepIndex < 0) {
      continue;
    }
    const time = Date.parse(execution.updatedAt ?? execution.createdAt ?? "") || 0;
    if (
      !best ||
      stepIndex > bestStepIndex ||
      (stepIndex === bestStepIndex && time > bestTime) ||
      (stepIndex === bestStepIndex && time === bestTime && (execution.id ?? "") > (best.id ?? ""))
    ) {
      best = execution;
      bestStepIndex = stepIndex;
      bestTime = time;
    }
  }

  return best;
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
    // Finished failed runs are row-level only — do not drive the phase aggregate.
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
