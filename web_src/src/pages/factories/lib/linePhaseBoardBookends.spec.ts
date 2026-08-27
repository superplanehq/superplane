import { describe, expect, it } from "vitest";
import type {
  FactoriesFactoryLine,
  FactoriesLineRef,
  FactoriesWorkOrder,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
} from "@/api-client";
import { buildLinePhaseBoard, collectLineDoneOrders, lineStageColumns, type LinePhaseColumn } from "./linePhaseRuns";

const APPS = [
  { id: "app-plan", name: "Planning" },
  { id: "app-refund-done", name: "Done" },
];

const LINE: FactoriesFactoryLine = {
  id: "line-1",
  name: "poc",
  steps: [{ app: { app: "app-plan" } }, { app: { app: "app-refund-done" } }],
};

type TestExecution = FactoriesWorkOrderExecution & { line?: FactoriesLineRef };

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

function closedOrder(id: string, updatedAt: string, lineId?: string): FactoriesWorkOrder {
  return {
    id,
    title: id,
    state: "STATE_CLOSED",
    updatedAt,
    lineDispatches: lineId ? [{ id: `d-${id}`, line: { id: lineId }, stepExecutions: [] }] : [],
  };
}

describe("collectLineDoneOrders", () => {
  it("returns closed work orders for this line, newest first", () => {
    const closedOld = closedOrder("wo-closed-old", "2026-08-11T10:00:00.000Z", "line-1");
    const closedNew = closedOrder("wo-closed-new", "2026-08-11T14:00:00.000Z", "line-1");
    const closedOtherLine = closedOrder("wo-other-line", "2026-08-11T16:00:00.000Z", "line-other");
    const closedNoLine = closedOrder("wo-no-line", "2026-08-11T17:00:00.000Z");
    const draft: FactoriesWorkOrder = {
      id: "wo-draft",
      title: "Draft",
      state: "STATE_DRAFT",
      updatedAt: "2026-08-11T15:00:00.000Z",
      lineDispatches: [],
    };
    const open = order("wo-open", "Open", [
      {
        id: "e-open",
        line: { id: "line-1", name: "poc" },
        step: "plan",
        stepIndex: 0,
        state: "STATE_STARTED",
        createdAt: "2026-08-11T13:00:00.000Z",
      },
    ]);

    const done = collectLineDoneOrders([closedOld, draft, open, closedNew, closedOtherLine, closedNoLine], LINE);

    expect(done.map((entry) => entry.id)).toEqual(["wo-no-line", "wo-closed-new", "wo-closed-old"]);
  });

  it("keeps open work that is still on a Done step after the stage column is dropped", () => {
    const openOnDone = order("wo-open-done", "Closing", [
      {
        id: "e-done",
        line: { id: "line-1", name: "poc" },
        step: "Done",
        stepIndex: 1,
        state: "STATE_STARTED",
        createdAt: "2026-08-11T13:00:00.000Z",
        updatedAt: "2026-08-11T13:00:00.000Z",
        run: { id: "run-done", appId: "app-refund-done" },
      },
    ]);
    const board = buildLinePhaseBoard(LINE, [openOnDone], APPS);

    expect(lineStageColumns(board).flatMap((column) => column.runs.map((run) => run.workOrderId))).toEqual([]);

    const done = collectLineDoneOrders([openOnDone], LINE, board);

    expect(done.map((entry) => entry.id)).toEqual(["wo-open-done"]);
  });
});

describe("lineStageColumns", () => {
  it("drops Done-named and PR-closure columns from the stage row", () => {
    const columns: LinePhaseColumn[] = [
      { stepName: "Planning", stepIndex: 0, appId: "app-plan", maxParallelism: 10, runs: [], tick: null },
      { stepName: "Open PR", stepIndex: 1, appId: "app-pr", maxParallelism: 10, runs: [], tick: null },
      { stepName: "Done", stepIndex: 2, appId: "app-refund-done", maxParallelism: 10, runs: [], tick: null },
    ];

    expect(lineStageColumns(columns).map((column) => column.stepName)).toEqual(["Planning", "Open PR"]);
  });
});
