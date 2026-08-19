import type { CanvasesCanvasRun, FactoriesWorkOrder, FactoriesWorkOrderExecution } from "@/api-client";
import { shortId } from "@/ui/Runs/runPresentation";
import { flattenWorkOrderExecutions } from "./workOrderExecutions";

export type FactoryAutomationTick = "running" | "waiting" | "queued" | "passed" | "failed" | "cancelled" | null;

export type FactoryAutomationStatus = {
  tick: Extract<FactoryAutomationTick, "running" | "waiting" | "queued" | null>;
  label: "Executing" | "Needs input" | "Queued" | "Idle";
};

export type FactoryAutomationRunCard = {
  runId: string;
  title: string;
  tick: Exclude<FactoryAutomationTick, null>;
  label: "Executing" | "Cancelling" | "Needs input" | "Queued" | "Passed" | "Failed" | "Cancelled";
  updatedAt?: string;
  createdAt?: string;
  finishedAt?: string;
};

/**
 * Aggregates line-step executions for a factory app into a list-card status.
 * Priority matches v3 Automations: Needs input → Executing → Queued → Idle.
 */
export function resolveFactoryAutomationStatus(
  appId: string | undefined,
  workOrders: FactoriesWorkOrder[],
): FactoryAutomationStatus {
  if (!appId) {
    return { tick: null, label: "Idle" };
  }

  let hasRunning = false;
  let hasWaiting = false;
  let hasQueued = false;

  for (const order of workOrders) {
    for (const dispatch of order.lineDispatches ?? []) {
      for (const execution of dispatch.stepExecutions ?? []) {
        if (execution.run?.appId !== appId) {
          continue;
        }
        const kind = classifyWorkOrderExecution(execution);
        if (kind === "running") hasRunning = true;
        else if (kind === "waiting") hasWaiting = true;
        else if (kind === "queued") hasQueued = true;
      }
    }
  }

  if (hasWaiting) return { tick: "waiting", label: "Needs input" };
  if (hasRunning) return { tick: "running", label: "Executing" };
  if (hasQueued) return { tick: "queued", label: "Queued" };
  return { tick: null, label: "Idle" };
}

/**
 * Status for an automation detail header from canvas ListRuns results.
 * Same priority as the work-order aggregate: Needs input → Executing → Queued → Idle.
 */
export function resolveFactoryAutomationStatusFromCanvasRuns(runs: CanvasesCanvasRun[]): FactoryAutomationStatus {
  let hasRunning = false;
  let hasWaiting = false;
  let hasQueued = false;

  for (const run of runs) {
    const kind = classifyCanvasRunTick(run);
    // Finished failed runs are not "Needs input" — only a true waiting signal.
    if (kind === "waiting") hasWaiting = true;
    else if (kind === "running") hasRunning = true;
    else if (kind === "queued") hasQueued = true;
  }

  if (hasWaiting) return { tick: "waiting", label: "Needs input" };
  if (hasRunning) return { tick: "running", label: "Executing" };
  if (hasQueued) return { tick: "queued", label: "Queued" };
  return { tick: null, label: "Idle" };
}

/**
 * Maps canvas ListRuns payloads into Automations detail Run cards (newest first).
 * Callers own pagination (infinite ListRuns); this maps the already-loaded page set.
 */
export function listFactoryAutomationRuns(runs: CanvasesCanvasRun[]): FactoryAutomationRunCard[] {
  const cards: FactoryAutomationRunCard[] = [];

  for (const run of runs) {
    if (!run.id) {
      continue;
    }
    const classified = classifyCanvasRun(run);
    if (!classified) {
      continue;
    }
    cards.push({
      runId: run.id,
      title: canvasRunTitle(run),
      tick: classified.tick,
      label: classified.label,
      updatedAt: run.updatedAt,
      createdAt: run.createdAt,
      finishedAt: run.finishedAt,
    });
  }

  cards.sort(compareRunsNewestFirst);
  return cards;
}

function canvasRunTitle(run: CanvasesCanvasRun): string {
  const customName = run.rootEvent?.customName?.trim();
  if (customName) {
    return customName;
  }
  return `Run ${shortId(run.id)}`;
}

function classifyCanvasRun(
  run: CanvasesCanvasRun,
): { tick: Exclude<FactoryAutomationTick, null>; label: FactoryAutomationRunCard["label"] } | null {
  if (run.state === "STATE_CANCELLING") {
    return { tick: "running", label: "Cancelling" };
  }
  const tick = classifyCanvasRunTick(run);
  if (!tick) {
    return null;
  }
  return { tick, label: tickLabel(tick) };
}

function classifyCanvasRunTick(run: CanvasesCanvasRun): Exclude<FactoryAutomationTick, null> | null {
  if (run.state === "STATE_PENDING") {
    return "queued";
  }
  if (run.state === "STATE_STARTED" || run.state === "STATE_CANCELLING") {
    return "running";
  }
  if (run.result === "RESULT_FAILED") {
    return "failed";
  }
  if (run.result === "RESULT_CANCELLED") {
    return "cancelled";
  }
  if (run.result === "RESULT_PASSED" || run.state === "STATE_FINISHED") {
    return "passed";
  }
  return null;
}

function tickLabel(tick: Exclude<FactoryAutomationTick, null>): FactoryAutomationRunCard["label"] {
  if (tick === "running") return "Executing";
  if (tick === "waiting") return "Needs input";
  if (tick === "failed") return "Failed";
  if (tick === "queued") return "Queued";
  if (tick === "passed") return "Passed";
  return "Cancelled";
}

function classifyWorkOrderExecution(execution: FactoriesWorkOrderExecution): FactoryAutomationTick {
  if (execution.state === "STATE_STARTED" || execution.state === "STATE_CANCELLING") {
    return "running";
  }
  if (execution.state === "STATE_PENDING") {
    return "queued";
  }
  // Finished results (passed/failed/cancelled) do not drive the aggregate card —
  // Idle when nothing is actively executing, queued, or waiting for input.
  return null;
}

function compareRunsNewestFirst(left: FactoryAutomationRunCard, right: FactoryAutomationRunCard): number {
  const leftTime = Date.parse(left.updatedAt ?? left.finishedAt ?? left.createdAt ?? "") || 0;
  const rightTime = Date.parse(right.updatedAt ?? right.finishedAt ?? right.createdAt ?? "") || 0;
  if (leftTime !== rightTime) {
    return rightTime - leftTime;
  }
  return right.runId.localeCompare(left.runId);
}

/**
 * Finds the work order that produced this canvas run, if any.
 * Trigger-only runs have no work order. Match on run id alone: ids are unique
 * and Storybook ListRuns is shared across automations.
 */
export function findWorkOrderForAutomationRun(
  workOrders: FactoriesWorkOrder[],
  runId: string,
): FactoriesWorkOrder | undefined {
  if (!runId) {
    return undefined;
  }

  return workOrders.find((order) => flattenWorkOrderExecutions(order).some((execution) => execution.run?.id === runId));
}
