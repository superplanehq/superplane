import type { CanvasesCanvasRun } from "@/api-client";
import { describe, expect, it } from "vitest";

import {
  analyzingWorkOrderIds,
  backlogAnalysisRuns,
  backlogAnalysisRunsByWorkOrder,
  findBacklogAnalyzerCanvasId,
  hasActiveBacklogAnalysisRun,
} from "./backlogAnalysis";

function analysisRun(overrides: {
  id: string;
  workOrderId?: string;
  state?: CanvasesCanvasRun["state"];
  createdAt?: string;
}): CanvasesCanvasRun {
  return {
    id: overrides.id,
    state: overrides.state ?? "STATE_STARTED",
    createdAt: overrides.createdAt ?? "2026-08-28T10:00:00Z",
    rootEvent: overrides.workOrderId
      ? { data: { type: "factory.workOrder", data: { workOrder: { id: overrides.workOrderId } } } }
      : undefined,
  };
}

describe("findBacklogAnalyzerCanvasId", () => {
  it("skips intake canvases that share the name", () => {
    const apps = [
      { id: "app-intake", name: "Backlog" },
      { id: "app-analyzer", name: "Backlog" },
    ];

    expect(findBacklogAnalyzerCanvasId(apps, ["app-intake"])).toBe("app-analyzer");
  });

  it("returns nothing when the factory has no analyzer", () => {
    expect(findBacklogAnalyzerCanvasId([{ id: "app-plan", name: "Plan" }])).toBeUndefined();
  });
});

describe("backlogAnalysisRuns", () => {
  it("reads the work order out of the trigger payload, oldest run first", () => {
    const runs = backlogAnalysisRuns("app-analyzer", [
      analysisRun({ id: "run-2", workOrderId: "wo-2", createdAt: "2026-08-28T11:00:00Z" }),
      analysisRun({ id: "run-1", workOrderId: "wo-1", createdAt: "2026-08-28T10:00:00Z" }),
    ]);

    expect(runs.map((entry) => entry.run.id)).toEqual(["run-1", "run-2"]);
    expect(runs.map((entry) => entry.workOrderId)).toEqual(["wo-1", "wo-2"]);
    expect(runs[0].canvasId).toBe("app-analyzer");
  });

  it("drops runs without a work order", () => {
    expect(backlogAnalysisRuns("app-analyzer", [analysisRun({ id: "run-1" })])).toEqual([]);
  });
});

describe("analyzingWorkOrderIds", () => {
  it("reports only work orders with a run in flight", () => {
    const runs = backlogAnalysisRuns("app-analyzer", [
      analysisRun({ id: "run-1", workOrderId: "wo-1", state: "STATE_STARTED" }),
      analysisRun({ id: "run-2", workOrderId: "wo-2", state: "STATE_FINISHED" }),
    ]);

    expect([...analyzingWorkOrderIds(runs)]).toEqual(["wo-1"]);
    expect(hasActiveBacklogAnalysisRun(runs)).toBe(true);
  });

  it("stops reporting once every run finished", () => {
    const runs = backlogAnalysisRuns("app-analyzer", [
      analysisRun({ id: "run-1", workOrderId: "wo-1", state: "STATE_FINISHED" }),
    ]);

    expect([...analyzingWorkOrderIds(runs)]).toEqual([]);
    expect(hasActiveBacklogAnalysisRun(runs)).toBe(false);
  });
});

describe("backlogAnalysisRunsByWorkOrder", () => {
  it("groups every run of the same work order", () => {
    const runs = backlogAnalysisRuns("app-analyzer", [
      analysisRun({ id: "run-1", workOrderId: "wo-1", createdAt: "2026-08-28T10:00:00Z" }),
      analysisRun({ id: "run-2", workOrderId: "wo-1", createdAt: "2026-08-28T11:00:00Z" }),
      analysisRun({ id: "run-3", workOrderId: "wo-2", createdAt: "2026-08-28T12:00:00Z" }),
    ]);

    const grouped = backlogAnalysisRunsByWorkOrder(runs);

    expect(grouped.get("wo-1")?.map((entry) => entry.run.id)).toEqual(["run-1", "run-2"]);
    expect(grouped.get("wo-2")?.map((entry) => entry.run.id)).toEqual(["run-3"]);
  });
});
