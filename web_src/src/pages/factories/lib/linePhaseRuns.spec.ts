import { describe, expect, it } from "vitest";
import type {
  FactoriesFactoryLine,
  FactoriesLineRef,
  FactoriesWorkOrder,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
} from "@/api-client";
import { buildLinePhaseBoard, resolvePhaseRunStatus } from "./linePhaseRuns";

const LINE: FactoriesFactoryLine = {
  id: "line-1",
  name: "poc",
  steps: [{ name: "plan" }, { name: "build" }, { name: "demo" }],
};

/** Fixture-only shape: a step execution plus which line it ran on, before
 * it's grouped into a dispatch by `order()` below. */
type TestExecution = FactoriesWorkOrderExecution & { line?: FactoriesLineRef };

// Groups the given step executions into one dispatch per distinct line
// (matching what the real API returns), so test cases can list executions
// with an inline `line` ref without hand-building the nested shape.
function order(id: string, title: string, executions: TestExecution[]): FactoriesWorkOrder {
  const dispatchesByLineId = new Map<string, FactoriesWorkOrderLineDispatch>();
  for (const { line, ...execution } of executions) {
    const lineId = line?.id ?? "unknown";
    const dispatch = dispatchesByLineId.get(lineId);
    if (dispatch) {
      dispatch.stepExecutions = [...(dispatch.stepExecutions ?? []), execution];
      continue;
    }
    dispatchesByLineId.set(lineId, {
      id: `dispatch-${id}-${lineId}`,
      line,
      createdAt: execution.createdAt,
      stepExecutions: [execution],
    });
  }
  return { id, title, state: "STATE_OPEN", lineDispatches: [...dispatchesByLineId.values()] };
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
    expect(board[0].runs.map((run) => run.title)).toEqual(["Beta", "Gamma", "Alpha", "Delta"]);
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
    expect(board[2].runs[0]).toMatchObject({
      workOrderId: "wo-progress",
      title: "Progressing",
      executionId: "e-demo",
    });
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
