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

/**
 * Optimistic "an analysis is expected" store, keyed by work order id.
 *
 * A freshly created draft has no Backlog run yet — the run is created
 * asynchronously after the create RPC returns — so the board cannot learn
 * "analyzing" from real run data alone. This tiny external store lets the
 * create mutation say "show analyzing now" the moment it succeeds, while
 * `useBacklogAnalysisRuns` keeps polling until the real run (or its result)
 * shows up. Entries self-clean via a TTL backstop so a card can never get
 * stuck in "Analyzing" if a run never appears.
 */
const PENDING_ANALYSIS_TTL_MS = 60_000;

const pendingAnalysis = new Map<string, number>();
const pendingAnalysisListeners = new Set<() => void>();
let pendingAnalysisSnapshot: ReadonlySet<string> = new Set();

function notifyPendingAnalysisListeners(): void {
  pendingAnalysisSnapshot = new Set(pendingAnalysis.keys());
  for (const listener of pendingAnalysisListeners) {
    listener();
  }
}

/** Mark a work order as expecting a Backlog analysis run. */
export function markBacklogAnalysisPending(workOrderId: string | undefined | null): void {
  if (!workOrderId) {
    return;
  }
  pendingAnalysis.set(workOrderId, Date.now() + PENDING_ANALYSIS_TTL_MS);
  notifyPendingAnalysisListeners();
}

/** Clear a work order once its real run (or result) is known. */
export function clearBacklogAnalysisPending(workOrderId: string | undefined | null): void {
  if (!workOrderId || !pendingAnalysis.delete(workOrderId)) {
    return;
  }
  notifyPendingAnalysisListeners();
}

/** Live pending ids, pruning anything past its TTL. */
export function pendingBacklogAnalysisIds(now = Date.now()): ReadonlySet<string> {
  let pruned = false;
  for (const [workOrderId, expiresAt] of pendingAnalysis) {
    if (expiresAt <= now) {
      pendingAnalysis.delete(workOrderId);
      pruned = true;
    }
  }
  if (pruned) {
    notifyPendingAnalysisListeners();
  }
  return pendingAnalysisSnapshot;
}

/** Subscribe to pending-set changes; returns an unsubscribe function. */
export function subscribeBacklogAnalysisPending(listener: () => void): () => void {
  pendingAnalysisListeners.add(listener);
  return () => {
    pendingAnalysisListeners.delete(listener);
  };
}
