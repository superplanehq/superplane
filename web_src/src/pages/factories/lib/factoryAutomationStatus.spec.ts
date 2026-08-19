import { describe, expect, it } from "vitest";
import type { CanvasesCanvasRun, FactoriesWorkOrder, FactoriesWorkOrderExecution } from "@/api-client";
import {
  findWorkOrderForAutomationRun,
  listFactoryAutomationRuns,
  resolveFactoryAutomationStatus,
  resolveFactoryAutomationStatusFromCanvasRuns,
} from "./factoryAutomationStatus";

function order(id: string, title: string, executions: FactoriesWorkOrderExecution[]): FactoriesWorkOrder {
  return {
    id,
    title,
    state: "STATE_OPEN",
    lineDispatches: executions.length > 0 ? [{ id: `dispatch-${id}`, stepExecutions: executions }] : [],
  };
}

function canvasRun(overrides: CanvasesCanvasRun): CanvasesCanvasRun {
  return overrides;
}

describe("resolveFactoryAutomationStatus", () => {
  it("returns Idle when app has no executions", () => {
    expect(resolveFactoryAutomationStatus("app-1", [order("wo-1", "A", [])])).toEqual({
      tick: null,
      label: "Idle",
    });
  });

  it("returns Executing when a run is started", () => {
    expect(
      resolveFactoryAutomationStatus("app-1", [
        order("wo-1", "A", [
          {
            id: "e1",
            state: "STATE_STARTED",
            run: { id: "r1", appId: "app-1" },
          },
        ]),
      ]),
    ).toEqual({ tick: "running", label: "Executing" });
  });

  it("returns Idle when only finished failed executions exist", () => {
    expect(
      resolveFactoryAutomationStatus("app-1", [
        order("wo-1", "A", [
          {
            id: "e2",
            state: "STATE_FINISHED",
            result: "RESULT_FAILED",
            run: { id: "r2", appId: "app-1" },
          },
        ]),
      ]),
    ).toEqual({ tick: null, label: "Idle" });
  });

  it("prefers Executing over finished failed siblings", () => {
    expect(
      resolveFactoryAutomationStatus("app-1", [
        order("wo-1", "A", [
          {
            id: "e1",
            state: "STATE_STARTED",
            run: { id: "r1", appId: "app-1" },
          },
          {
            id: "e2",
            state: "STATE_FINISHED",
            result: "RESULT_FAILED",
            run: { id: "r2", appId: "app-1" },
          },
        ]),
      ]),
    ).toEqual({ tick: "running", label: "Executing" });
  });

  it("ignores executions for other apps", () => {
    expect(
      resolveFactoryAutomationStatus("app-1", [
        order("wo-1", "A", [
          {
            id: "e1",
            state: "STATE_STARTED",
            run: { id: "r1", appId: "app-other" },
          },
        ]),
      ]),
    ).toEqual({ tick: null, label: "Idle" });
  });
});

describe("resolveFactoryAutomationStatusFromCanvasRuns", () => {
  it("returns Idle when there are no active runs", () => {
    expect(
      resolveFactoryAutomationStatusFromCanvasRuns([
        canvasRun({
          id: "r-pass",
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
        }),
      ]),
    ).toEqual({ tick: null, label: "Idle" });
  });

  it("returns Idle when canvas runs only failed", () => {
    expect(
      resolveFactoryAutomationStatusFromCanvasRuns([
        canvasRun({ id: "r2", state: "STATE_FINISHED", result: "RESULT_FAILED" }),
      ]),
    ).toEqual({ tick: null, label: "Idle" });
  });

  it("prefers Executing when a canvas run is started even if another failed", () => {
    expect(
      resolveFactoryAutomationStatusFromCanvasRuns([
        canvasRun({ id: "r1", state: "STATE_STARTED" }),
        canvasRun({ id: "r2", state: "STATE_FINISHED", result: "RESULT_FAILED" }),
      ]),
    ).toEqual({ tick: "running", label: "Executing" });
  });
});

describe("listFactoryAutomationRuns", () => {
  it("maps canvas runs newest first with run titles and statuses", () => {
    const runs = listFactoryAutomationRuns([
      canvasRun({
        id: "aaaaaaaa-old",
        state: "STATE_FINISHED",
        result: "RESULT_FAILED",
        updatedAt: "2026-01-01T00:00:00.000Z",
        rootEvent: { customName: "Old failure" },
      }),
      canvasRun({
        id: "bbbbbbbb-new",
        state: "STATE_STARTED",
        updatedAt: "2026-01-02T00:00:00.000Z",
        rootEvent: { customName: "New executing" },
      }),
      canvasRun({
        id: "cccccccc-pass",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        updatedAt: "2026-01-03T00:00:00.000Z",
      }),
    ]);

    expect(runs.map((run) => run.runId)).toEqual(["cccccccc-pass", "bbbbbbbb-new", "aaaaaaaa-old"]);
    expect(runs[0]).toMatchObject({ title: "Run cccccccc", label: "Passed", tick: "passed" });
    expect(runs[1]).toMatchObject({ title: "New executing", label: "Executing", tick: "running" });
    expect(runs[2]).toMatchObject({ title: "Old failure", label: "Failed", tick: "failed" });
  });

  it("labels STATE_CANCELLING as Cancelling with a running tick", () => {
    const runs = listFactoryAutomationRuns([
      canvasRun({
        id: "dddddddd-cancel",
        state: "STATE_CANCELLING",
        updatedAt: "2026-01-04T00:00:00.000Z",
      }),
    ]);

    expect(runs).toEqual([
      expect.objectContaining({
        runId: "dddddddd-cancel",
        tick: "running",
        label: "Cancelling",
      }),
    ]);
  });
});

describe("findWorkOrderForAutomationRun", () => {
  const orders: FactoriesWorkOrder[] = [
    order("wo-other", "Other", [{ id: "e0", run: { id: "run-x", appId: "app-other" } }]),
    order("wo-match", "Matched", [{ id: "e1", run: { id: "run-1", appId: "app-1" } }]),
  ];

  it("returns the work order whose execution run matches this canvas run", () => {
    expect(findWorkOrderForAutomationRun(orders, "run-1")?.id).toBe("wo-match");
  });

  it("matches by run id even when the execution app differs", () => {
    expect(findWorkOrderForAutomationRun(orders, "run-x")?.id).toBe("wo-other");
  });

  it("returns undefined when no execution matches the run", () => {
    expect(findWorkOrderForAutomationRun(orders, "run-missing")).toBeUndefined();
  });
});
