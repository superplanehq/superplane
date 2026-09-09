import { describe, expect, it } from "vitest";

import { firstRunAnalysisProgress } from "./firstRunAnalysisProgress";

function run(placement: string) {
  return { placement } as never;
}

describe("firstRunAnalysisProgress", () => {
  it("is at the import stage while runs are unknown or empty", () => {
    expect(firstRunAnalysisProgress(undefined)).toEqual({ total: 0, scored: 0, ready: 0, stageIndex: 0 });
    expect(firstRunAnalysisProgress([])).toEqual({ total: 0, scored: 0, ready: 0, stageIndex: 0 });
  });

  it("is at the scoring stage while any run is analyzing", () => {
    const runs = [run("PLACEMENT_ANALYZING"), run("PLACEMENT_BACKLOG")];
    expect(firstRunAnalysisProgress(runs)).toEqual({ total: 2, scored: 1, ready: 1, stageIndex: 1 });
  });

  it("is done when every run is scored", () => {
    const runs = [run("PLACEMENT_BACKLOG"), run("PLACEMENT_BELOW_THRESHOLD")];
    expect(firstRunAnalysisProgress(runs)).toEqual({ total: 2, scored: 2, ready: 1, stageIndex: 2 });
  });

  // Backlog items and items already dispatched to a line both scored above
  // the threshold, so both count as ready.
  it("counts progressed runs as ready", () => {
    const runs = [run("PLACEMENT_PROGRESSED"), run("PLACEMENT_REJECTED")];
    expect(firstRunAnalysisProgress(runs)).toEqual({ total: 2, scored: 2, ready: 1, stageIndex: 2 });
  });
});
