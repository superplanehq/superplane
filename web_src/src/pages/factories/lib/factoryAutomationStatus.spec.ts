import { describe, expect, it } from "vitest";
import type { CanvasesCanvasRun, FactoriesWorkOrder } from "@/api-client";
import {
  listFactoryAutomationRuns,
  resolveFactoryAutomationStatus,
  resolveFactoryAutomationStatusFromCanvasRuns,
} from "./factoryAutomationStatus";

function order(id: string, title: string, executions: FactoriesWorkOrder["executions"]): FactoriesWorkOrder {
  return { id, title, state: "STATE_OPEN", executions };
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

  it("prefers Needs input over Executing", () => {
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
    ).toEqual({ tick: "waiting", label: "Needs input" });
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
  it("returns Idle when there are no active or failed runs", () => {
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

  it("prefers Needs input when a canvas run failed", () => {
    expect(
      resolveFactoryAutomationStatusFromCanvasRuns([
        canvasRun({ id: "r1", state: "STATE_STARTED" }),
        canvasRun({ id: "r2", state: "STATE_FINISHED", result: "RESULT_FAILED" }),
      ]),
    ).toEqual({ tick: "waiting", label: "Needs input" });
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
