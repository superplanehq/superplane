import type { FactoriesFactoryIntakeRun } from "@/api-client";

export type FirstRunAnalysisProgress = { total: number; scored: number; stageIndex: 0 | 1 | 2 };

const ANALYZING = "PLACEMENT_ANALYZING";

export function firstRunAnalysisProgress(runs: FactoriesFactoryIntakeRun[] | undefined): FirstRunAnalysisProgress {
  if (!runs || runs.length === 0) return { total: 0, scored: 0, stageIndex: 0 };
  const scored = runs.filter((item) => item.placement && String(item.placement) !== ANALYZING).length;
  return { total: runs.length, scored, stageIndex: scored === runs.length ? 2 : 1 };
}
