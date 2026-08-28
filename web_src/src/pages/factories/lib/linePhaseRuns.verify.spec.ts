import { describe, expect, it } from "vitest";
import type {
  FactoriesFactoryLine,
  FactoriesLineRef,
  FactoriesWorkOrder,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
} from "@/api-client";
import {
  buildLinePhaseBoard,
  collectLineVerifyOrders,
  lineStageColumns,
  visibleLineStageColumns,
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

describe("collectLineVerifyOrders", () => {
  it("moves a waiting order that passed the last stage into Verify", () => {
    const waiting = order("wo-wait", "Waiting review", [
      {
        id: "e-last",
        line: { id: "line-1", name: "poc" },
        step: "demo",
        stepIndex: 2,
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        createdAt: "2026-08-11T12:00:00.000Z",
        updatedAt: "2026-08-11T12:00:00.000Z",
      },
    ]);

    const board = buildLinePhaseBoard(LINE, [waiting], APPS);
    const verify = collectLineVerifyOrders(board);

    expect(verify.map((entry) => entry.id)).toEqual(["wo-wait"]);
    expect(visibleLineStageColumns(board, verify).flatMap((column) => column.runs)).toEqual([]);
  });

  it("keeps a failed last stage on the stage column", () => {
    const failed = order("wo-failed", "Failed implement", [
      {
        id: "e-fail",
        line: { id: "line-1", name: "poc" },
        step: "demo",
        stepIndex: 2,
        state: "STATE_FINISHED",
        result: "RESULT_FAILED",
        createdAt: "2026-08-11T12:00:00.000Z",
        updatedAt: "2026-08-11T12:00:00.000Z",
      },
    ]);

    const board = buildLinePhaseBoard(LINE, [failed], APPS);
    const verify = collectLineVerifyOrders(board);

    expect(verify).toEqual([]);
    expect(lineStageColumns(board)[2]?.runs.map((run) => run.workOrderId)).toEqual(["wo-failed"]);
  });

  it("leaves a closed order off Verify so Done can collect it", () => {
    const closed: FactoriesWorkOrder = {
      ...order("wo-closed", "Closed after last stage", [
        {
          id: "e-last",
          line: { id: "line-1", name: "poc" },
          step: "demo",
          stepIndex: 2,
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          createdAt: "2026-08-11T12:00:00.000Z",
          updatedAt: "2026-08-11T12:00:00.000Z",
        },
      ]),
      state: "STATE_CLOSED",
    };

    const board = buildLinePhaseBoard(LINE, [closed], APPS);
    expect(collectLineVerifyOrders(board)).toEqual([]);
  });
});
