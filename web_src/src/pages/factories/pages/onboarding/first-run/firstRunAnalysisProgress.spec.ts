import { describe, expect, it } from "vitest";

import { firstRunAnalysisProgress } from "./firstRunAnalysisProgress";

function run(placement: string, extra: { confidencePct?: number; workOrderId?: string } = {}) {
  return { placement, ...extra } as never;
}

describe("firstRunAnalysisProgress", () => {
  it("is at the import stage while runs are unknown or empty", () => {
    expect(firstRunAnalysisProgress(undefined)).toEqual({ total: 0, scored: 0, ready: 0, stageIndex: 0, empty: false });
    expect(firstRunAnalysisProgress([])).toEqual({ total: 0, scored: 0, ready: 0, stageIndex: 0, empty: false });
  });

  // Zero runs on a settled import means the source had no open tickets, but
  // a run list that has not loaded yet says nothing about the source.
  it("reports an empty source only when the import settled and runs loaded", () => {
    expect(firstRunAnalysisProgress([], new Set(), true)).toEqual({
      total: 0,
      scored: 0,
      ready: 0,
      stageIndex: 0,
      empty: true,
    });
    expect(firstRunAnalysisProgress(undefined, new Set(), true)).toEqual({
      total: 0,
      scored: 0,
      ready: 0,
      stageIndex: 0,
      empty: false,
    });
  });

  // An intake without an analysis node places items on the backlog the
  // moment the import creates them. That placement is not a score.
  it("does not count a backlog placement without a score signal as scored", () => {
    const runs = [run("PLACEMENT_BACKLOG", { workOrderId: "wo-1" }), run("PLACEMENT_BACKLOG", { workOrderId: "wo-2" })];
    expect(firstRunAnalysisProgress(runs)).toEqual({ total: 2, scored: 0, ready: 0, stageIndex: 1 });
  });

  // The Backlog automation scores tickets that the import already placed on
  // the board, so a finished analysis run makes the ticket ready to run.
  it("counts a ticket as scored and ready once its Backlog analysis run finished", () => {
    const runs = [run("PLACEMENT_BACKLOG", { workOrderId: "wo-1" }), run("PLACEMENT_BACKLOG", { workOrderId: "wo-2" })];
    expect(firstRunAnalysisProgress(runs, new Set(["wo-1"]))).toEqual({ total: 2, scored: 1, ready: 1, stageIndex: 1 });
    expect(firstRunAnalysisProgress(runs, new Set(["wo-1", "wo-2"]))).toEqual({
      total: 2,
      scored: 2,
      ready: 2,
      stageIndex: 2,
    });
  });

  it("counts a ticket with an intake confidence score as scored and ready", () => {
    const runs = [run("PLACEMENT_BACKLOG", { confidencePct: 82, workOrderId: "wo-1" }), run("PLACEMENT_ANALYZING")];
    expect(firstRunAnalysisProgress(runs)).toEqual({ total: 2, scored: 1, ready: 1, stageIndex: 1 });
  });

  it("counts terminal placements as scored without a confidence score", () => {
    // Progressed work already moved to a line, so it also counts as ready.
    const runs = [run("PLACEMENT_BELOW_THRESHOLD"), run("PLACEMENT_REJECTED"), run("PLACEMENT_PROGRESSED")];
    expect(firstRunAnalysisProgress(runs)).toEqual({ total: 3, scored: 3, ready: 1, stageIndex: 2 });
  });
});
