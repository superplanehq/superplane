import type { FactoriesFactoryIntakeRun } from "@/api-client";

export type FirstRunAnalysisProgress = {
  total: number;
  scored: number;
  /** Scored above the threshold: on the backlog or already on a line. */
  ready: number;
  stageIndex: 0 | 1 | 2;
};

const ANALYZING = "PLACEMENT_ANALYZING";
const READY = ["PLACEMENT_BACKLOG", "PLACEMENT_PROGRESSED"];

export function firstRunAnalysisProgress(runs: FactoriesFactoryIntakeRun[] | undefined): FirstRunAnalysisProgress {
  if (!runs || runs.length === 0) return { total: 0, scored: 0, ready: 0, stageIndex: 0 };
  const scored = runs.filter((item) => item.placement && String(item.placement) !== ANALYZING).length;
  const ready = runs.filter((item) => READY.includes(String(item.placement))).length;
  return { total: runs.length, scored, ready, stageIndex: scored === runs.length ? 2 : 1 };
}
