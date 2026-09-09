import { describe, expect, it } from "vitest";

import { firstRunAnalysisProgress } from "./firstRunAnalysisProgress";

function run(placement: string) {
  return { placement } as never;
}

describe("firstRunAnalysisProgress", () => {
  it("is at the import stage while runs are unknown or empty", () => {
    expect(firstRunAnalysisProgress(undefined)).toEqual({ total: 0, scored: 0, stageIndex: 0 });
    expect(firstRunAnalysisProgress([])).toEqual({ total: 0, scored: 0, stageIndex: 0 });
  });

  it("is at the scoring stage while any run is analyzing", () => {
    const runs = [run("PLACEMENT_ANALYZING"), run("PLACEMENT_BACKLOG")];
    expect(firstRunAnalysisProgress(runs)).toEqual({ total: 2, scored: 1, stageIndex: 1 });
  });

  it("is at the board stage when every run is scored", () => {
    const runs = [run("PLACEMENT_BACKLOG"), run("PLACEMENT_BELOW_THRESHOLD")];
    expect(firstRunAnalysisProgress(runs)).toEqual({ total: 2, scored: 2, stageIndex: 2 });
  });
});
