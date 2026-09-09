import type { FactoriesFactoryIntakeRun } from "@/api-client";

export type FirstRunAnalysisProgress = {
  total: number;
  scored: number;
  /** Scored above the threshold: on the backlog or already on a line. */
  ready: number;
  stageIndex: 0 | 1 | 2;
};

// Intake decisions that only exist after a score: the item was gated out,
// or its work order already started on a line.
const TERMINAL = ["PLACEMENT_BELOW_THRESHOLD", "PLACEMENT_REJECTED", "PLACEMENT_PROGRESSED"];
const READY = ["PLACEMENT_BACKLOG", "PLACEMENT_PROGRESSED"];

/**
 * An intake without an analysis node places items on the backlog the moment
 * the import creates them, and the factory Backlog automation scores them
 * afterwards. Placement alone therefore says nothing about scoring; a ticket
 * counts as scored only when a score signal exists: a terminal placement, a
 * confidence percentage from the intake, or a finished Backlog analysis run
 * for its work order (`scoredOrderIds`, from useFactoryBacklogAnalysis).
 */
function isScored(run: FactoriesFactoryIntakeRun, scoredOrderIds: ReadonlySet<string>): boolean {
  const placement = String(run.placement ?? "");
  if (TERMINAL.includes(placement)) return true;
  if (run.confidencePct != null) return true;
  return Boolean(run.workOrderId && scoredOrderIds.has(run.workOrderId));
}

export function firstRunAnalysisProgress(
  runs: FactoriesFactoryIntakeRun[] | undefined,
  scoredOrderIds: ReadonlySet<string> = new Set(),
): FirstRunAnalysisProgress {
  if (!runs || runs.length === 0) return { total: 0, scored: 0, ready: 0, stageIndex: 0 };
  const scored = runs.filter((run) => isScored(run, scoredOrderIds)).length;
  // Only an intake analysis reports the score with the run, so `ready` stays
  // zero when the Backlog automation scores, and the result line omits it.
  const ready = runs.filter((run) => run.confidencePct != null && READY.includes(String(run.placement))).length;
  return { total: runs.length, scored, ready, stageIndex: scored === runs.length ? 2 : 1 };
}
