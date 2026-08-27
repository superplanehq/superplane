import { describe, expect, it } from "vitest";
import type {
  FactoriesFactoryLine,
  FactoriesLineRef,
  FactoriesWorkOrder,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
} from "@/api-client";
import { factoryAppPath, factoryAppRunPath, linesPath } from "./factoryPagePaths";
import { BOARD_IMPLEMENT_NOTIFY_ORDER } from "../__fixtures__/lineMetricsBoardOrders";
import { REFUND_LINE_PLAN_ID } from "../__fixtures__/factoryPageIds";
import {
  buildLinePhaseBoard,
  collectLineBacklogOrders,
  findBacklogAutomationApp,
  findClosureAutomationApp,
  isDoneLineColumn,
  linePhaseRunHref,
  resolvePhaseRunStatus,
} from "./linePhaseRuns";

const APPS = [
  { id: "app-plan", name: "plan" },
  { id: "app-build", name: "build" },
  { id: "app-demo", name: "demo" },
];

const LINE: FactoriesFactoryLine = {
  id: "line-1",
  name: "poc",
  steps: [{ app: { app: "app-plan" } }, { app: { app: "app-build" } }, { app: { app: "app-demo" } }],
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
          stepIndex: 0,
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
          stepIndex: 0,
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
          stepIndex: 0,
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
          stepIndex: 0,
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
          stepIndex: 0,
          state: "STATE_STARTED",
          createdAt: "2026-08-11T13:00:00.000Z",
          updatedAt: "2026-08-11T13:00:00.000Z",
        },
      ]),
    ];

    const board = buildLinePhaseBoard(LINE, orders, APPS);

    expect(board).toHaveLength(3);
    expect(board[0].stepName).toBe("plan");
    expect(board[0].appId).toBe("app-plan");
    expect(board[0].runs.map((run) => run.order.title)).toEqual(["Beta", "Gamma", "Alpha", "Delta"]);
    expect(board[0].tick).toBe("running");
    expect(board[1].stepName).toBe("build");
    expect(board[1].runs).toEqual([]);
    expect(board[1].tick).toBeNull();
    expect(board[2].runs).toEqual([]);
    expect(workOrderIds(board)).toEqual(["wo-b", "wo-c", "wo-a", "wo-d"]);
  });

  it("keeps closed work orders off the stage columns", () => {
    const closed = {
      ...order("wo-closed", "Closed", [
        {
          id: "e-closed",
          line: { id: "line-1", name: "poc" },
          step: "plan",
          stepIndex: 0,
          state: "STATE_FINISHED",
          result: "RESULT_FAILED",
          createdAt: "2026-08-11T12:00:00.000Z",
          updatedAt: "2026-08-11T12:00:00.000Z",
        },
      ]),
      state: "STATE_CLOSED" as const,
    };

    const board = buildLinePhaseBoard(LINE, [closed], APPS);

    expect(workOrderIds(board)).toEqual([]);
  });

  it("places a multi-step work order only in its furthest active step", () => {
    const orders = [
      order("wo-progress", "Progressing", [
        {
          id: "e-plan",
          line: { id: "line-1", name: "poc" },
          step: "plan",
          stepIndex: 0,
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          createdAt: "2026-08-11T10:00:00.000Z",
          updatedAt: "2026-08-11T10:00:00.000Z",
        },
        {
          id: "e-build",
          line: { id: "line-1", name: "poc" },
          step: "build",
          stepIndex: 1,
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          createdAt: "2026-08-11T11:00:00.000Z",
          updatedAt: "2026-08-11T11:00:00.000Z",
        },
        {
          id: "e-demo",
          line: { id: "line-1", name: "poc" },
          step: "demo",
          stepIndex: 2,
          state: "STATE_STARTED",
          createdAt: "2026-08-11T12:00:00.000Z",
          updatedAt: "2026-08-11T12:30:00.000Z",
        },
      ]),
    ];

    const board = buildLinePhaseBoard(LINE, orders, APPS);

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
          stepIndex: 0,
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          createdAt: "2026-08-11T09:00:00.000Z",
          updatedAt: "2026-08-11T09:00:00.000Z",
        },
        {
          id: "e-fail",
          line: { id: "line-1", name: "poc" },
          step: "build",
          stepIndex: 1,
          state: "STATE_FINISHED",
          result: "RESULT_FAILED",
          createdAt: "2026-08-11T10:00:00.000Z",
          updatedAt: "2026-08-11T10:00:00.000Z",
        },
      ]),
    ];

    const board = buildLinePhaseBoard(LINE, orders, APPS);

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
      steps: [{ app: { app: "app-planner", entrypoint: "start" } }],
    };

    const board = buildLinePhaseBoard(line, [], [{ id: "app-planner", name: "plan" }]);
    expect(board[0]).toMatchObject({ stepName: "plan", appId: "app-planner" });
  });

  it("places the notify card on Implement", () => {
    const line: FactoriesFactoryLine = {
      id: REFUND_LINE_PLAN_ID,
      name: "plan-and-implement",
      steps: [{ app: { app: "app-refund-implementer" } }, { app: { app: "app-refund-verifier" } }],
    };
    const board = buildLinePhaseBoard(
      line,
      [BOARD_IMPLEMENT_NOTIFY_ORDER],
      [
        { id: "app-refund-implementer", name: "Implement" },
        { id: "app-refund-verifier", name: "Verify" },
      ],
    );

    expect(board[0].runs.map((run) => run.order.title)).toEqual(["Notify on status change after a reopen"]);
    expect(board[1].runs).toEqual([]);
  });

  it("keeps two columns when the same automation appears twice", () => {
    const line: FactoriesFactoryLine = {
      id: "line-1",
      name: "poc",
      steps: [{ app: { app: "app-plan" } }, { app: { app: "app-plan" } }],
    };
    const orders = [
      order("wo-a", "Alpha", [
        {
          id: "e-second",
          line: { id: "line-1", name: "poc" },
          step: "plan",
          stepIndex: 1,
          state: "STATE_STARTED",
          createdAt: "2026-08-11T12:00:00.000Z",
          updatedAt: "2026-08-11T12:00:00.000Z",
        },
      ]),
    ];

    const board = buildLinePhaseBoard(line, orders, APPS);
    expect(board).toHaveLength(2);
    expect(board[0]).toMatchObject({ stepName: "plan", stepIndex: 0, appId: "app-plan" });
    expect(board[1]).toMatchObject({ stepName: "plan", stepIndex: 1, appId: "app-plan" });
    expect(board[0].runs).toEqual([]);
    expect(board[1].runs).toHaveLength(1);
    expect(board[1].runs[0].workOrderId).toBe("wo-a");
  });

  it("places a card on the live automation after a step is inserted ahead of it", () => {
    const line: FactoriesFactoryLine = {
      id: "line-1",
      name: "poc",
      steps: [{ app: { app: "app-new" } }, { app: { app: "app-plan" } }, { app: { app: "app-build" } }],
    };
    const orders = [
      order("wo-a", "Alpha", [
        {
          id: "e1",
          line: { id: "line-1", name: "poc" },
          step: "plan",
          stepIndex: 0,
          state: "STATE_STARTED",
          run: { appId: "app-plan" },
          createdAt: "2026-08-11T12:00:00.000Z",
          updatedAt: "2026-08-11T12:00:00.000Z",
        },
      ]),
    ];

    const board = buildLinePhaseBoard(line, orders, [...APPS, { id: "app-new", name: "new" }]);
    expect(board[0].runs).toEqual([]);
    expect(board[1].runs).toHaveLength(1);
    expect(board[1].runs[0].workOrderId).toBe("wo-a");
    expect(board[2].runs).toEqual([]);
  });

  it("keeps phase idle when only finished failed runs exist", () => {
    const orders = [
      order("wo-f", "Failing", [
        {
          id: "e-fail",
          line: { id: "line-1", name: "poc" },
          step: "build",
          stepIndex: 1,
          state: "STATE_FINISHED",
          result: "RESULT_FAILED",
          createdAt: "2026-08-11T10:00:00.000Z",
          updatedAt: "2026-08-11T10:00:00.000Z",
        },
      ]),
    ];

    const board = buildLinePhaseBoard(LINE, orders, APPS);
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
        orderNumber: "42",
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

describe("collectLineBacklogOrders", () => {
  it("returns only draft orders that are not on a line", () => {
    const onLine = order("wo-on-line", "On line", [
      {
        id: "e-on",
        line: { id: "line-1", name: "poc" },
        step: "plan",
        stepIndex: 0,
        state: "STATE_STARTED",
        createdAt: "2026-08-11T12:00:00.000Z",
        updatedAt: "2026-08-11T12:00:00.000Z",
      },
    ]);
    const otherLine = order("wo-other-line", "Other line", [
      {
        id: "e-other",
        line: { id: "line-other", name: "other" },
        step: "plan",
        stepIndex: 0,
        state: "STATE_STARTED",
        createdAt: "2026-08-11T13:00:00.000Z",
        updatedAt: "2026-08-11T13:00:00.000Z",
      },
    ]);
    const draft: FactoriesWorkOrder = {
      id: "wo-draft",
      title: "Draft",
      state: "STATE_DRAFT",
      updatedAt: "2026-08-11T14:00:00.000Z",
      lineDispatches: [],
    };
    const open: FactoriesWorkOrder = {
      id: "wo-open",
      title: "Open",
      state: "STATE_OPEN",
      updatedAt: "2026-08-11T11:00:00.000Z",
      lineDispatches: [],
    };
    const closed: FactoriesWorkOrder = {
      id: "wo-closed",
      title: "Closed",
      state: "STATE_CLOSED",
      lineDispatches: [],
    };

    const backlog = collectLineBacklogOrders([onLine, otherLine, draft, open, closed]);

    expect(backlog.map((entry) => entry.id)).toEqual(["wo-draft"]);
  });
});

describe("findBacklogAutomationApp", () => {
  it("returns the factory backlog automation", () => {
    expect(
      findBacklogAutomationApp([
        { id: "app-plan", name: "Plan" },
        { id: "app-refund-backlog", name: "Backlog" },
      ]),
    ).toEqual({ id: "app-refund-backlog", name: "Backlog" });
  });

  it("matches the Ingest app name", () => {
    expect(findBacklogAutomationApp([{ id: "app-refund-backlog", name: "Ingest" }])).toEqual({
      id: "app-refund-backlog",
      name: "Ingest",
    });
  });
});

describe("findClosureAutomationApp", () => {
  it("returns the factory PR Closure automation", () => {
    expect(
      findClosureAutomationApp([
        { id: "app-plan", name: "Plan" },
        { id: "app-pr-closure", name: "PR Closure" },
      ]),
    ).toEqual({ id: "app-pr-closure", name: "PR Closure" });
  });

  it("matches the refund done app id when the name is absent", () => {
    expect(findClosureAutomationApp([{ id: "app-refund-done" }])).toEqual({
      id: "app-refund-done",
      name: "PR Closure",
    });
  });
});

describe("isDoneLineColumn", () => {
  it("treats the Done name and the closure app id as special columns", () => {
    expect(isDoneLineColumn({ stepName: "Done", appId: "app-plan" })).toBe(true);
    expect(isDoneLineColumn({ stepName: "Phase 4", appId: "app-refund-done" })).toBe(true);
    expect(isDoneLineColumn({ stepName: "Plan", appId: "app-plan" })).toBe(false);
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
