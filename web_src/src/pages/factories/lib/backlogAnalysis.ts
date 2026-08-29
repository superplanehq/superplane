import type { CanvasesCanvasRun } from "@/api-client";

import { isActiveCanvasRun } from "./workOrderPullRequest";

/**
 * One run of the factory Backlog automation and the task it analyzes.
 * The `factory.onWorkOrder` trigger payload is the only place the work
 * order id exists, so the run carries it in its root event.
 */
export type BacklogAnalysisRun = {
  canvasId: string;
  workOrderId: string;
  run: CanvasesCanvasRun;
};

/**
 * Backlog analyzer of a factory. Intake automations can carry the same
 * name, so the caller excludes their canvases.
 */
export function findBacklogAnalyzerCanvasId(
  apps: Array<{ id?: string; name?: string }>,
  intakeCanvasIds: Iterable<string> = [],
): string | undefined {
  const intakeIds = new Set(intakeCanvasIds);
  const match = apps.find((app) => app.id && !intakeIds.has(app.id) && isBacklogAnalyzerName(app.name));
  return match?.id;
}

function isBacklogAnalyzerName(name?: string): boolean {
  return (name ?? "").trim().toLowerCase().startsWith("backlog");
}

/** Analysis runs of one canvas, oldest first, keyed to their task. */
export function backlogAnalysisRuns(canvasId: string, runs: CanvasesCanvasRun[]): BacklogAnalysisRun[] {
  return runs
    .flatMap((run) => {
      const workOrderId = analyzedWorkOrderId(run);
      if (!run.id || !workOrderId) {
        return [];
      }
      return [{ canvasId, workOrderId, run }];
    })
    .sort((left, right) => Date.parse(left.run.createdAt ?? "") - Date.parse(right.run.createdAt ?? ""));
}

export function backlogAnalysisRunsByWorkOrder(runs: BacklogAnalysisRun[]): Map<string, BacklogAnalysisRun[]> {
  const byWorkOrder = new Map<string, BacklogAnalysisRun[]>();
  for (const entry of runs) {
    const existing = byWorkOrder.get(entry.workOrderId);
    if (existing) {
      existing.push(entry);
      continue;
    }
    byWorkOrder.set(entry.workOrderId, [entry]);
  }
  return byWorkOrder;
}

/** Tasks whose score is still on the way. */
export function analyzingWorkOrderIds(runs: BacklogAnalysisRun[]): ReadonlySet<string> {
  const ids = new Set<string>();
  for (const entry of runs) {
    if (isActiveCanvasRun(entry.run)) {
      ids.add(entry.workOrderId);
    }
  }
  return ids;
}

export function hasActiveBacklogAnalysisRun(runs: BacklogAnalysisRun[]): boolean {
  return runs.some((entry) => isActiveCanvasRun(entry.run));
}

function analyzedWorkOrderId(run: CanvasesCanvasRun): string | undefined {
  const envelope = asRecord(run.rootEvent?.data);
  const payload = asRecord(envelope?.data) ?? envelope;
  const id = asRecord(payload?.workOrder)?.id;
  if (typeof id !== "string") {
    return undefined;
  }
  return id.trim() || undefined;
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }
  return value as Record<string, unknown>;
}
