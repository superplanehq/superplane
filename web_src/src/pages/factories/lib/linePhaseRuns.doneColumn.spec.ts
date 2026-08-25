import { describe, expect, it } from "vitest";
import type {
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
} from "@/api-client";
import { buildLinePhaseBoard, collectLineDoneOrders, lineBoardEndsWithDoneStep } from "./linePhaseRuns";

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

function execution(stepIndex: number, result: FactoriesWorkOrderExecution["result"]): FactoriesWorkOrderExecution {
  return {
    id: `e-${stepIndex}`,
    stepIndex,
    state: "STATE_FINISHED",
    result,
    createdAt: "2026-08-11T10:00:00.000Z",
    updatedAt: "2026-08-11T10:00:00.000Z",
  };
}

function closedOrder(args: {
  id: string;
  result: FactoriesWorkOrder["result"];
  lineId?: string;
  stepIndex?: number;
  stepResult?: FactoriesWorkOrderExecution["result"];
  dispatchResult?: FactoriesWorkOrderLineDispatch["result"];
  updatedAt?: string;
}): FactoriesWorkOrder {
  return {
    id: args.id,
    title: args.id,
    state: "STATE_CLOSED",
    result: args.result,
    updatedAt: args.updatedAt,
    lineDispatches: args.lineId
      ? [
          {
            id: `dispatch-${args.id}`,
            line: { id: args.lineId, name: args.lineId },
            result: args.dispatchResult,
            stepExecutions: [execution(args.stepIndex ?? 2, args.stepResult ?? "RESULT_PASSED")],
          },
        ]
      : [],
  };
}

describe("buildLinePhaseBoard with a board Done column", () => {
  it("takes a completed work order off the phase columns", () => {
    const done = closedOrder({ id: "wo-done", result: "RESULT_COMPLETED", lineId: "line-1" });

    const board = buildLinePhaseBoard(LINE, [done], APPS);

    expect(lineBoardEndsWithDoneStep(board)).toBe(false);
    expect(board.flatMap((column) => column.runs)).toEqual([]);
  });

  it("keeps a failed work order on the phase where it stopped", () => {
    const failed = closedOrder({
      id: "wo-failed",
      result: "RESULT_FAILED",
      lineId: "line-1",
      stepIndex: 1,
      stepResult: "RESULT_FAILED",
    });

    const board = buildLinePhaseBoard(LINE, [failed], APPS);

    expect(board[1].runs).toHaveLength(1);
    expect(board[1].runs[0].workOrderId).toBe("wo-failed");
  });

  it("keeps completed work orders on a line that ends with its own Done automation", () => {
    const line: FactoriesFactoryLine = {
      id: "line-1",
      name: "poc",
      steps: [{ app: { app: "app-plan" } }, { app: { app: "app-done" } }],
    };
    const done = closedOrder({ id: "wo-done", result: "RESULT_COMPLETED", lineId: "line-1", stepIndex: 1 });

    const board = buildLinePhaseBoard(line, [done], [...APPS, { id: "app-done", name: "Done" }]);

    expect(lineBoardEndsWithDoneStep(board)).toBe(true);
    expect(board[1].runs).toHaveLength(1);
  });
});

describe("collectLineDoneOrders", () => {
  it("returns completed, rejected, and canceled orders of this line, newest first", () => {
    const completed = closedOrder({
      id: "wo-completed",
      result: "RESULT_COMPLETED",
      lineId: "line-1",
      updatedAt: "2026-08-11T12:00:00.000Z",
    });
    const rejected = closedOrder({
      id: "wo-rejected",
      result: "RESULT_REJECTED",
      lineId: "line-1",
      updatedAt: "2026-08-11T14:00:00.000Z",
    });
    const canceled = closedOrder({
      id: "wo-canceled",
      result: "RESULT_UNSPECIFIED",
      lineId: "line-1",
      dispatchResult: "RESULT_CANCELLED",
      updatedAt: "2026-08-11T13:00:00.000Z",
    });

    const done = collectLineDoneOrders("line-1", [completed, rejected, canceled]);

    expect(done.map((entry) => entry.id)).toEqual(["wo-rejected", "wo-canceled", "wo-completed"]);
  });

  it("leaves out failed orders, open orders, and orders of another line", () => {
    const failed = closedOrder({ id: "wo-failed", result: "RESULT_FAILED", lineId: "line-1" });
    const otherLine = closedOrder({ id: "wo-other", result: "RESULT_COMPLETED", lineId: "line-other" });
    const open: FactoriesWorkOrder = { id: "wo-open", title: "Open", state: "STATE_OPEN", lineDispatches: [] };

    const done = collectLineDoneOrders("line-1", [failed, otherLine, open]);

    expect(done).toEqual([]);
  });

  it("includes an order that closed before it reached a line", () => {
    const closedInBacklog = closedOrder({ id: "wo-backlog-closed", result: "RESULT_COMPLETED" });

    const done = collectLineDoneOrders("line-1", [closedInBacklog]);

    expect(done.map((entry) => entry.id)).toEqual(["wo-backlog-closed"]);
  });
});
