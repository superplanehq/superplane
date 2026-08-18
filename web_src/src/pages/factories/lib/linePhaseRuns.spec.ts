import { describe, expect, it } from "vitest";
import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import { factoryAppPath, factoryAppRunPath, linesPath } from "./factoryPagePaths";
import { buildLinePhaseBoard, linePhaseRunHref, resolvePhaseRunStatus } from "./linePhaseRuns";

const LINE: FactoriesFactoryLine = {
  id: "line-1",
  name: "poc",
  steps: [{ name: "plan" }, { name: "build" }, { name: "demo" }],
};

function order(id: string, title: string, executions: FactoriesWorkOrder["executions"]): FactoriesWorkOrder {
  return { id, title, state: "STATE_OPEN", executions };
}

function workOrderIds(board: ReturnType<typeof buildLinePhaseBoard>): string[] {
  return board.flatMap((column) => column.runs.map((run) => run.workOrderId));
}

describe("buildLinePhaseBoard", () => {
  it("returns one column per step with current-step cards newest first", () => {
    const orders: FactoriesWorkOrder[] = [
      order("wo-a", "Alpha", [
        {
          id: "e1",
          line: { id: "line-1", name: "poc" },
          step: "plan",
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          createdAt: "2026-08-11T10:00:00.000Z",
          updatedAt: "2026-08-11T10:00:00.000Z",
        },
      ]),
      order("wo-b", "Beta", [
        {
          id: "e2",
          line: { id: "line-1", name: "poc" },
          step: "plan",
          state: "STATE_STARTED",
          createdAt: "2026-08-11T11:00:00.000Z",
          updatedAt: "2026-08-11T12:00:00.000Z",
        },
      ]),
      order("wo-c", "Gamma", [
        {
          id: "e3",
          line: { id: "line-1", name: "poc" },
          step: "plan",
          state: "STATE_PENDING",
          createdAt: "2026-08-11T11:30:00.000Z",
          updatedAt: "2026-08-11T11:30:00.000Z",
        },
      ]),
      order("wo-d", "Delta", [
        {
          id: "e4",
          line: { id: "line-1", name: "poc" },
          step: "plan",
          state: "STATE_PENDING",
          createdAt: "2026-08-11T09:00:00.000Z",
          updatedAt: "2026-08-11T09:00:00.000Z",
        },
      ]),
      order("wo-other-line", "Skip", [
        {
          id: "e5",
          line: { id: "line-other", name: "other" },
          step: "plan",
          state: "STATE_STARTED",
          createdAt: "2026-08-11T13:00:00.000Z",
          updatedAt: "2026-08-11T13:00:00.000Z",
        },
      ]),
    ];

    const board = buildLinePhaseBoard(LINE, orders);

    expect(board).toHaveLength(3);
    expect(board[0].stepName).toBe("plan");
    expect(board[0].appId).toBeUndefined();
    expect(board[0].runs.map((run) => run.order.title)).toEqual(["Beta", "Gamma", "Alpha", "Delta"]);
    expect(board[0].tick).toBe("running");
    expect(board[1].stepName).toBe("build");
    expect(board[1].runs).toEqual([]);
    expect(board[1].tick).toBeNull();
    expect(board[2].runs).toEqual([]);
    expect(workOrderIds(board)).toEqual(["wo-b", "wo-c", "wo-a", "wo-d"]);
  });

  it("places a multi-step work order only in its furthest active step", () => {
    const orders = [
      order("wo-progress", "Progressing", [
        {
          id: "e-plan",
          line: { id: "line-1", name: "poc" },
          step: "plan",
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          createdAt: "2026-08-11T10:00:00.000Z",
          updatedAt: "2026-08-11T10:00:00.000Z",
        },
        {
          id: "e-build",
          line: { id: "line-1", name: "poc" },
          step: "build",
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          createdAt: "2026-08-11T11:00:00.000Z",
          updatedAt: "2026-08-11T11:00:00.000Z",
        },
        {
          id: "e-demo",
          line: { id: "line-1", name: "poc" },
          step: "demo",
          state: "STATE_STARTED",
          createdAt: "2026-08-11T12:00:00.000Z",
          updatedAt: "2026-08-11T12:30:00.000Z",
        },
      ]),
    ];

    const board = buildLinePhaseBoard(LINE, orders);

    expect(board[0].runs).toEqual([]);
    expect(board[1].runs).toEqual([]);
    expect(board[2].runs).toHaveLength(1);
    expect(board[2].runs[0]).toMatchObject({ workOrderId: "wo-progress", executionId: "e-demo" });
    expect(board[2].tick).toBe("running");
    expect(workOrderIds(board)).toEqual(["wo-progress"]);
  });

  it("places a failed mid-line work order only on the failed step", () => {
    const orders = [
      order("wo-fail", "Failing", [
        {
          id: "e-plan",
          line: { id: "line-1", name: "poc" },
          step: "plan",
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          createdAt: "2026-08-11T09:00:00.000Z",
          updatedAt: "2026-08-11T09:00:00.000Z",
        },
        {
          id: "e-fail",
          line: { id: "line-1", name: "poc" },
          step: "build",
          state: "STATE_FINISHED",
          result: "RESULT_FAILED",
          createdAt: "2026-08-11T10:00:00.000Z",
          updatedAt: "2026-08-11T10:00:00.000Z",
        },
      ]),
    ];

    const board = buildLinePhaseBoard(LINE, orders);

    expect(board[0].runs).toEqual([]);
    expect(board[1].runs).toHaveLength(1);
    expect(board[1].runs[0]).toMatchObject({ workOrderId: "wo-fail", executionId: "e-fail" });
    expect(resolvePhaseRunStatus(board[1].runs[0].execution)).toEqual({ kind: "failed", label: "Failed" });
    expect(board[1].tick).toBeNull();
    expect(board[2].runs).toEqual([]);
    expect(workOrderIds(board)).toEqual(["wo-fail"]);
  });

  it("includes the step app id for configure navigation", () => {
    const line: FactoriesFactoryLine = {
      id: "line-1",
      name: "poc",
      steps: [{ name: "plan", app: { app: "app-planner", entrypoint: "start" } }],
    };

    const board = buildLinePhaseBoard(line, []);
    expect(board[0]).toMatchObject({ stepName: "plan", appId: "app-planner" });
  });

  it("keeps phase idle when only finished failed runs exist", () => {
    const orders = [
      order("wo-f", "Failing", [
        {
          id: "e-fail",
          line: { id: "line-1", name: "poc" },
          step: "build",
          state: "STATE_FINISHED",
          result: "RESULT_FAILED",
          createdAt: "2026-08-11T10:00:00.000Z",
          updatedAt: "2026-08-11T10:00:00.000Z",
        },
      ]),
    ];

    const board = buildLinePhaseBoard(LINE, orders);
    expect(board[1].tick).toBeNull();
  });
});

describe("linePhaseRunHref", () => {
  it("opens the canvas run when the phase execution has an app run", () => {
    const href = linePhaseRunHref("org-1", "RF", "line-1", {
      executionId: "e1",
      workOrderId: "wo-1",
      order: { id: "wo-1", number: "42", title: "Ship retries" },
      execution: { id: "e1", run: { id: "run-implement", appId: "app-refund-implementer" } },
    });

    expect(href).toBe(
      factoryAppRunPath("org-1", "RF", "app-refund-implementer", "run-implement", {
        from: "lines",
        lineId: "line-1",
      }),
    );
  });

  it("opens the phase canvas when the execution has an app but no run id", () => {
    const href = linePhaseRunHref(
      "org-1",
      "RF",
      "line-1",
      {
        executionId: "e1",
        workOrderId: "wo-1",
        order: { id: "wo-1", number: "42", title: "Ship retries" },
        execution: { id: "e1" },
      },
      "app-refund-implementer",
    );

    expect(href).toBe(factoryAppPath("org-1", "RF", "app-refund-implementer", { from: "lines", lineId: "line-1" }));
  });

  it("falls back to the lines list when the phase has no canvas", () => {
    const href = linePhaseRunHref("org-1", "RF", "line-1", {
      executionId: "e1",
      workOrderId: "wo-1",
      order: { id: "wo-1", title: "Ship retries" },
      execution: { id: "e1" },
    });

    expect(href).toBe(linesPath("org-1", "RF"));
  });
});

describe("resolvePhaseRunStatus", () => {
  it("maps execution states to board labels", () => {
    expect(resolvePhaseRunStatus({ state: "STATE_STARTED" })).toEqual({ kind: "running", label: "Executing" });
    expect(resolvePhaseRunStatus({ state: "STATE_CANCELLING" })).toEqual({ kind: "running", label: "Cancelling" });
    expect(resolvePhaseRunStatus({ state: "STATE_PENDING" })).toEqual({ kind: "queued", label: "Queued" });
    expect(resolvePhaseRunStatus({ state: "STATE_FINISHED", result: "RESULT_PASSED" })).toEqual({
      kind: "idle",
      label: "Passed",
    });
    expect(resolvePhaseRunStatus({ state: "STATE_FINISHED", result: "RESULT_FAILED" })).toEqual({
      kind: "failed",
      label: "Failed",
    });
  });
});
