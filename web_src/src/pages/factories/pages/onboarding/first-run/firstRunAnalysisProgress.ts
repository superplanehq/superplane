import type { FactoriesFactoryIntake, FactoriesFactoryIntakeRun, FactoryIntakeInitialImportStatus } from "@/api-client";

export type FirstRunAnalysisProgress = {
  total: number;
  scored: number;
  /** Scored and on the board (or already on a line), so it can run. */
  ready: number;
  stageIndex: 0 | 1 | 2;
  /** The import finished and the ticket source had no open tickets. */
  empty?: boolean;
};

export type FirstRunInitialImport = {
  status?: FactoryIntakeInitialImportStatus;
  itemCount?: number;
};

// Intake decisions that only exist after a score: the item was gated out,
// or its work order already started on a line.
const TERMINAL = ["PLACEMENT_BELOW_THRESHOLD", "PLACEMENT_REJECTED", "PLACEMENT_PROGRESSED"];
const READY = ["PLACEMENT_BACKLOG", "PLACEMENT_PROGRESSED"];

export function githubIssuesIntake(intakes: FactoriesFactoryIntake[] | undefined): FactoriesFactoryIntake | undefined {
  return intakes?.find((intake) => intake.source === "SOURCE_GITHUB_ISSUES");
}

export function initialImportFailed(
  status: FactoryIntakeInitialImportStatus | undefined,
  importSettled: boolean,
): boolean {
  if (status === "INITIAL_IMPORT_STATUS_FAILED" || status === "INITIAL_IMPORT_STATUS_SKIPPED") return true;
  return status === "INITIAL_IMPORT_STATUS_PENDING" && importSettled;
}

function completedImportItemCount(initialImport: FirstRunInitialImport): number | undefined {
  if (initialImport.status !== "INITIAL_IMPORT_STATUS_COMPLETED") return undefined;
  return initialImport.itemCount;
}

function legacyImportIsEmpty(initialImport: FirstRunInitialImport, importSettled: boolean): boolean {
  const legacyStatus = !initialImport.status || initialImport.status === "INITIAL_IMPORT_STATUS_UNSPECIFIED";
  return legacyStatus && importSettled;
}

function initialProgress(
  runsLoaded: boolean,
  importedItemCount: number | undefined,
  legacyEmpty: boolean,
): FirstRunAnalysisProgress {
  const empty = runsLoaded && (importedItemCount === 0 || (importedItemCount === undefined && legacyEmpty));
  return {
    total: importedItemCount ?? 0,
    scored: 0,
    ready: 0,
    stageIndex: importedItemCount !== undefined && importedItemCount > 0 ? 1 : 0,
    empty,
  };
}

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
  importSettled = false,
  initialImport: FirstRunInitialImport = {},
): FirstRunAnalysisProgress {
  const importedItemCount = completedImportItemCount(initialImport);

  if (!runs || runs.length === 0) {
    return initialProgress(Boolean(runs), importedItemCount, legacyImportIsEmpty(initialImport, importSettled));
  }
  const scored = runs.filter((run) => isScored(run, scoredOrderIds)).length;
  // Ready means scored and on the board, whichever path produced the score:
  // an intake confidence percentage or a finished Backlog analysis run.
  const ready = runs.filter((run) => READY.includes(String(run.placement)) && isScored(run, scoredOrderIds)).length;
  const total = Math.max(runs.length, importedItemCount ?? 0);
  return { total, scored, ready, stageIndex: scored === total ? 2 : 1 };
}
