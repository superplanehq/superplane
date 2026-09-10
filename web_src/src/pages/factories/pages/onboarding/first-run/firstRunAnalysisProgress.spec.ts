import { describe, expect, it } from "vitest";
import type { FactoriesFactoryIntake } from "@/api-client";

import { firstRunAnalysisProgress, githubIssuesIntake, type FirstRunInitialImport } from "./firstRunAnalysisProgress";

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

  it("reports an empty source immediately after a completed zero-item import", () => {
    const initialImport: FirstRunInitialImport = { status: "INITIAL_IMPORT_STATUS_COMPLETED", itemCount: 0 };

    expect(firstRunAnalysisProgress([], new Set(), false, initialImport)).toEqual({
      total: 0,
      scored: 0,
      ready: 0,
      stageIndex: 0,
      empty: true,
    });
  });

  it("uses the completed import count while queued runs become visible", () => {
    const initialImport: FirstRunInitialImport = { status: "INITIAL_IMPORT_STATUS_COMPLETED", itemCount: 2 };

    expect(firstRunAnalysisProgress([], new Set(), false, initialImport)).toEqual({
      total: 2,
      scored: 0,
      ready: 0,
      stageIndex: 1,
      empty: false,
    });
  });

  it("does not report failed or skipped imports as empty", () => {
    for (const status of ["INITIAL_IMPORT_STATUS_FAILED", "INITIAL_IMPORT_STATUS_SKIPPED"] as const) {
      expect(firstRunAnalysisProgress([], new Set(), true, { status })).toEqual({
        total: 0,
        scored: 0,
        ready: 0,
        stageIndex: 0,
        empty: false,
      });
    }
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

describe("githubIssuesIntake", () => {
  it("selects the GitHub intake when another source appears first", () => {
    const intakes: FactoriesFactoryIntake[] = [
      { id: "sentry-1", source: "SOURCE_SENTRY_EXCEPTIONS" },
      { id: "github-1", source: "SOURCE_GITHUB_ISSUES" },
    ];

    expect(githubIssuesIntake(intakes)?.id).toBe("github-1");
  });
});
