import type {
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
} from "@/api-client";
import { automationNameForLineStep, lineStepParallelism } from "./factoryLineFormShared";
import { factoryAppPath, factoryAppRunPath, linesPath } from "./factoryPagePaths";
import {
  dispatchStepRows,
  isActiveWorkOrderExecution,
  isQueuedStepRow,
  type WorkOrderStepRow,
} from "./workOrderExecutions";

export type LinePhaseTick = "running" | "waiting" | "queued" | "failed" | null;

/** Phase status expressed with a distinct glyph shape, not colour alone. */
export type PhaseGlyphKind = "running" | "waiting" | "queued" | "failed" | "passed" | "pending";

export type LinePhaseRunCard = {
  executionId: string;
  workOrderId: string;
  /** Raw work order, so the board can build the shared work order card model. */
  order: FactoriesWorkOrder;
  execution: WorkOrderStepRow;
};

export type LinePhaseColumn = {
  stepName: string;
  stepIndex: number;
  /** Factory app id for this runApp step, when present. */
  appId?: string;
  /** In-flight cap for this step. Defaults to 10 when the line omits it. */
  maxParallelism: number;
  runs: LinePhaseRunCard[];
  tick: LinePhaseTick;
};

/** Initial / step size for the scrollable phase-column run list. */
export const LINE_PHASE_RUNS_PAGE_SIZE = 3;

/**
 * Destination for a phase-board card: the split-run page for this phase.
 * Never the work order page — that destination stays on the Work Orders list.
 */
export function linePhaseRunHref(
  organizationId: string,
  factoryKey: string,
  lineId: string | undefined,
  run: LinePhaseRunCard,
  stepAppId?: string,
): string {
  const appId = run.execution.run?.appId || stepAppId;
  const runId = run.execution.run?.id;
  if (appId && runId) {
    return factoryAppRunPath(organizationId, factoryKey, appId, runId, {
      from: "lines",
      lineId,
      orderNumber: run.order.number,
    });
  }
  if (appId) {
    return factoryAppPath(organizationId, factoryKey, appId, { from: "lines", lineId });
  }
  return linesPath(organizationId, factoryKey);
}

/**
 * Builds the Lines detail board: one column per line step. Each work order
 * appears once, in the column for its current (furthest active, else furthest
 * finished) step on this line — newest cards first within a column.
 */
export function buildLinePhaseBoard(
  line: FactoriesFactoryLine,
  workOrders: FactoriesWorkOrder[],
  apps: Array<{ id?: string; name?: string }> = [],
): LinePhaseColumn[] {
  const lineId = line.id;
  const steps = line.steps ?? [];
  if (!lineId || steps.length === 0) {
    return [];
  }

  const runsByStep = collectCurrentRunsByStep(lineId, steps, workOrders);

  return steps.map((step, stepIndex) => {
    const runs = runsByStep.get(stepIndex) ?? [];
    const appId = step.app?.app?.trim() || undefined;
    return {
      stepName: automationNameForLineStep(step, apps, stepIndex),
      stepIndex,
      appId,
      maxParallelism: lineStepParallelism(step),
      runs,
      tick: resolvePhaseTick(runs),
    };
  });
}

/**
 * Draft work orders that are not on a line yet. Newest updated drafts
 * come first.
 */
export function collectLineBacklogOrders(workOrders: FactoriesWorkOrder[]): FactoriesWorkOrder[] {
  return workOrders.filter(isLineBacklogOrder).sort(compareOrdersNewestFirst);
}

/** Closed work orders. Newest closed orders come first. */
export function collectLineDoneOrders(workOrders: FactoriesWorkOrder[]): FactoriesWorkOrder[] {
  return workOrders.filter((order) => order.state === "STATE_CLOSED").sort(compareOrdersNewestFirst);
}

/** Stage columns only. Done is a fixed bookend, not a line step. */
export function lineStageColumns(columns: LinePhaseColumn[]): LinePhaseColumn[] {
  return columns.filter((column) => !isDoneLineColumn(column));
}

/** Factory-level intake automation. It is not a line step. */
export function findBacklogAutomationApp(
  apps: Array<{ id?: string; name?: string }>,
): { id: string; name: string } | undefined {
  const match = apps.find(
    (app) => app.id && (app.name === "Backlog" || app.name === "Ingest" || app.id === "app-refund-backlog"),
  );
  if (!match?.id) {
    return undefined;
  }
  return { id: match.id, name: match.name ?? "Ingest" };
}

/** Factory-level PR Closure automation. It is not a line step. */
export function findClosureAutomationApp(
  apps: Array<{ id?: string; name?: string }>,
): { id: string; name: string } | undefined {
  const match = apps.find(
    (app) =>
      Boolean(app.id) &&
      (app.name === "PR Closure" || app.id === "app-refund-done" || (app.id ?? "").includes("pr-closure")),
  );
  if (!match?.id) {
    return undefined;
  }
  return { id: match.id, name: match.name ?? "PR Closure" };
}

/** Backlog and Done are not canvas-backed columns. */
export function isDoneLineColumn(column: Pick<LinePhaseColumn, "stepName" | "appId">): boolean {
  if (column.stepName.trim().toLowerCase() === "done") {
    return true;
  }
  if (!column.appId) {
    return false;
  }
  return column.appId === "app-refund-done" || column.appId.includes("pr-closure");
}

function isLineBacklogOrder(order: FactoriesWorkOrder): boolean {
  if (!order.id || order.state !== "STATE_DRAFT") {
    return false;
  }
  return (order.lineDispatches ?? []).length === 0;
}

function compareOrdersNewestFirst(left: FactoriesWorkOrder, right: FactoriesWorkOrder): number {
  const leftTime = Date.parse(left.updatedAt ?? left.createdAt ?? "") || 0;
  const rightTime = Date.parse(right.updatedAt ?? right.createdAt ?? "") || 0;
  if (leftTime !== rightTime) {
    return rightTime - leftTime;
  }
  return (right.id ?? "").localeCompare(left.id ?? "");
}

export function resolvePhaseRunStatus(execution: WorkOrderStepRow): {
  kind: "running" | "waiting" | "queued" | "failed" | "idle";
  label: string;
} {
  if (isQueuedStepRow(execution)) {
    const position = execution.queuePosition ?? 0;
    return { kind: "queued", label: position > 0 ? `Queued #${position}` : "Queued" };
  }
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

/** Board-level status for a phase column header. */
export function resolveColumnGlyph(column: LinePhaseColumn): PhaseGlyphKind {
  if (column.tick) {
    return column.tick;
  }
  return column.runs.length > 0 ? "passed" : "pending";
}

/** Row-level status for a single run card. */
export function resolveRunGlyph(run: LinePhaseRunCard): PhaseGlyphKind {
  const { kind, label } = resolvePhaseRunStatus(run.execution);
  if (kind === "idle") {
    return label === "Passed" ? "passed" : "pending";
  }
  return kind;
}

function collectCurrentRunsByStep(
  lineId: string,
  steps: NonNullable<FactoriesFactoryLine["steps"]>,
  workOrders: FactoriesWorkOrder[],
): Map<number, LinePhaseRunCard[]> {
  const runsByStep = new Map<number, LinePhaseRunCard[]>();
  for (const order of workOrders) {
    appendCurrentRunForOrder(order, lineId, steps, runsByStep);
  }
  for (const runs of runsByStep.values()) {
    runs.sort(compareRunsNewestFirst);
  }
  return runsByStep;
}

function executionStepIndex(execution: FactoriesWorkOrderExecution): number | undefined {
  if (execution.stepIndex == null || execution.stepIndex < 0) {
    return undefined;
  }
  return execution.stepIndex;
}

function liveColumnIndexForExecution(
  steps: NonNullable<FactoriesFactoryLine["steps"]>,
  execution: FactoriesWorkOrderExecution,
): number | undefined {
  const stepIndex = executionStepIndex(execution);
  if (stepIndex == null) {
    return undefined;
  }

  // Dispatch stepIndex is a snapshot. A later line edit can move that
  // index onto a different automation. Keep the snapshot index when the
  // app still matches; otherwise place the card on the live column with
  // the same app.
  const executionAppId = execution.run?.appId?.trim();
  const liveAppId = steps[stepIndex]?.app?.app?.trim();
  if (stepIndex < steps.length && (!executionAppId || liveAppId === executionAppId)) {
    return stepIndex;
  }

  if (!executionAppId) {
    return undefined;
  }

  return closestAppColumnIndex(steps, executionAppId, stepIndex);
}

function closestAppColumnIndex(
  steps: NonNullable<FactoriesFactoryLine["steps"]>,
  executionAppId: string,
  stepIndex: number,
): number | undefined {
  const matches: number[] = [];
  for (let index = 0; index < steps.length; index++) {
    if (steps[index]?.app?.app?.trim() === executionAppId) {
      matches.push(index);
    }
  }
  if (matches.length === 0) {
    return undefined;
  }
  if (matches.length === 1) {
    return matches[0];
  }
  return matches.reduce((best, index) => (Math.abs(index - stepIndex) < Math.abs(best - stepIndex) ? index : best));
}

function appendCurrentRunForOrder(
  order: FactoriesWorkOrder,
  lineId: string,
  steps: NonNullable<FactoriesFactoryLine["steps"]>,
  runsByStep: Map<number, LinePhaseRunCard[]>,
): void {
  if (!order.id || order.state === "STATE_CLOSED") {
    return;
  }

  // A line can have more than one dispatch (traversal) for this order; the
  // board is a compact status summary, so it shows the most recent one —
  // the full history of every dispatch lives in WorkOrderExecutionsList.
  const dispatchesForLine = (order.lineDispatches ?? []).filter((dispatch) => dispatch.line?.id === lineId);
  if (dispatchesForLine.length === 0) {
    return;
  }
  const currentDispatch = pickMostRecentDispatch(dispatchesForLine);

  const lineExecutions = dispatchStepRows(currentDispatch).filter(
    (execution) => liveColumnIndexForExecution(steps, execution) != null,
  );
  if (lineExecutions.length === 0) {
    return;
  }

  const currentExecution = pickCurrentLineExecution(lineExecutions);
  const stepIndex = currentExecution ? liveColumnIndexForExecution(steps, currentExecution) : undefined;
  if (!currentExecution || stepIndex == null) {
    return;
  }

  const card: LinePhaseRunCard = {
    executionId: currentExecution.id ?? `${order.id}-${stepIndex}-${currentExecution.createdAt ?? ""}`,
    workOrderId: order.id,
    order,
    execution: currentExecution,
  };
  const existing = runsByStep.get(stepIndex);
  if (existing) {
    existing.push(card);
    return;
  }
  runsByStep.set(stepIndex, [card]);
}

function pickMostRecentDispatch(dispatches: FactoriesWorkOrderLineDispatch[]): FactoriesWorkOrderLineDispatch {
  return dispatches.reduce((latest, candidate) => {
    const latestAt = Date.parse(latest.createdAt ?? "") || 0;
    const candidateAt = Date.parse(candidate.createdAt ?? "") || 0;
    return candidateAt >= latestAt ? candidate : latest;
  });
}

function executionTimestamp(execution: WorkOrderStepRow): number {
  return Date.parse(execution.updatedAt ?? execution.createdAt ?? "") || 0;
}

function isPreferableCurrentExecution(candidate: WorkOrderStepRow, incumbent: WorkOrderStepRow): boolean {
  const candidateStep = executionStepIndex(candidate) ?? -1;
  const incumbentStep = executionStepIndex(incumbent) ?? -1;
  if (candidateStep !== incumbentStep) {
    return candidateStep > incumbentStep;
  }
  const candidateTime = executionTimestamp(candidate);
  const incumbentTime = executionTimestamp(incumbent);
  if (candidateTime !== incumbentTime) {
    return candidateTime > incumbentTime;
  }
  return (candidate.id ?? "") > (incumbent.id ?? "");
}

function pickCurrentLineExecution(executions: WorkOrderStepRow[]): WorkOrderStepRow | null {
  const active = executions.filter(isActiveWorkOrderExecution);
  const candidates = active.length > 0 ? active : executions;
  let best: WorkOrderStepRow | null = null;

  for (const execution of candidates) {
    if (executionStepIndex(execution) == null) {
      continue;
    }
    if (!best || isPreferableCurrentExecution(execution, best)) {
      best = execution;
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
