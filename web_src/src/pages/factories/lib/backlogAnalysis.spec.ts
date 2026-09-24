import type { CanvasesCanvasRun } from "@/api-client";
import { describe, expect, it, vi } from "bun:test";

import {
  analyzingWorkOrderIds,
  backlogAnalysisRuns,
  backlogAnalysisRunsByWorkOrder,
  clearBacklogAnalysisPending,
  findBacklogAnalyzerCanvasId,
  hasActiveBacklogAnalysisRun,
  markBacklogAnalysisPending,
  mergeBacklogAnalysisRunSnapshots,
  pendingBacklogAnalysisIds,
  subscribeBacklogAnalysisPending,
  upsertBacklogAnalysisRun,
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
  it("reads the task out of the trigger payload, oldest run first", () => {
    const runs = backlogAnalysisRuns("app-analyzer", [
      analysisRun({ id: "run-2", workOrderId: "wo-2", createdAt: "2026-08-28T11:00:00Z" }),
      analysisRun({ id: "run-1", workOrderId: "wo-1", createdAt: "2026-08-28T10:00:00Z" }),
    ]);

    expect(runs.map((entry) => entry.run.id)).toEqual(["run-1", "run-2"]);
    expect(runs.map((entry) => entry.workOrderId)).toEqual(["wo-1", "wo-2"]);
    expect(runs[0].canvasId).toBe("app-analyzer");
  });

  it("drops runs without a task", () => {
    expect(backlogAnalysisRuns("app-analyzer", [analysisRun({ id: "run-1" })])).toEqual([]);
  });

  it("adds and updates a run from websocket events", () => {
    const started = analysisRun({ id: "run-1", workOrderId: "wo-1", state: "STATE_STARTED" });
    const current = upsertBacklogAnalysisRun([], "app-analyzer", started);

    const finished = upsertBacklogAnalysisRun(current, "app-analyzer", {
      ...started,
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      updatedAt: "2026-08-28T10:01:00Z",
    });

    expect(finished).toHaveLength(1);
    expect(finished[0].run).toMatchObject({
      id: "run-1",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
    });
  });

  it("keeps a websocket event that arrives before an older REST snapshot", () => {
    const finished = {
      ...analysisRun({ id: "run-1", workOrderId: "wo-1", state: "STATE_FINISHED" }),
      result: "RESULT_PASSED" as const,
      updatedAt: "2026-08-28T10:01:00Z",
    };
    const live = upsertBacklogAnalysisRun(undefined, "app-analyzer", finished);
    const staleSnapshot = backlogAnalysisRuns("app-analyzer", [
      {
        ...finished,
        state: "STATE_STARTED",
        result: undefined,
        updatedAt: "2026-08-28T10:00:00Z",
      },
    ]);

    const merged = mergeBacklogAnalysisRunSnapshots(live, staleSnapshot, "app-analyzer");

    expect(merged[0]?.run.state).toBe("STATE_FINISHED");
    expect(merged[0]?.run.result).toBe("RESULT_PASSED");
  });
});

describe("analyzingWorkOrderIds", () => {
  it("reports only tasks with a run in flight", () => {
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
  it("groups every run of the same task", () => {
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

describe("pending backlog analysis store", () => {
  it("adds an id when marked pending", () => {
    markBacklogAnalysisPending("wo-pending-1");

    expect(pendingBacklogAnalysisIds().has("wo-pending-1")).toBe(true);

    clearBacklogAnalysisPending("wo-pending-1");
  });

  it("ignores empty ids", () => {
    markBacklogAnalysisPending("");
    markBacklogAnalysisPending(undefined);
    markBacklogAnalysisPending(null);

    expect(pendingBacklogAnalysisIds().size).toBe(0);
  });

  it("removes an id when cleared", () => {
    markBacklogAnalysisPending("wo-pending-2");
    clearBacklogAnalysisPending("wo-pending-2");

    expect(pendingBacklogAnalysisIds().has("wo-pending-2")).toBe(false);
  });

  it("drops an id once its TTL expires", () => {
    const markedAt = Date.now();
    markBacklogAnalysisPending("wo-pending-3");

    expect(pendingBacklogAnalysisIds(markedAt).has("wo-pending-3")).toBe(true);
    expect(pendingBacklogAnalysisIds(markedAt + 61_000).has("wo-pending-3")).toBe(false);
  });

  it("notifies listeners on mark and clear", () => {
    const listener = vi.fn();
    const unsubscribe = subscribeBacklogAnalysisPending(listener);

    markBacklogAnalysisPending("wo-pending-4");
    expect(listener).toHaveBeenCalledTimes(1);

    clearBacklogAnalysisPending("wo-pending-4");
    expect(listener).toHaveBeenCalledTimes(2);

    unsubscribe();
    markBacklogAnalysisPending("wo-pending-5");
    expect(listener).toHaveBeenCalledTimes(2);
    clearBacklogAnalysisPending("wo-pending-5");
  });
});
