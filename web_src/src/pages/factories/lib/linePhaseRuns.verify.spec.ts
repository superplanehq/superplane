import { describe, expect, it } from "bun:test";
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
    const verify = collectLineVerifyOrders(board, [waiting]);

    expect(verify.map((entry) => entry.id)).toEqual(["wo-wait"]);
    expect(visibleLineStageColumns(board, verify).flatMap((column) => column.runs)).toEqual([]);
    expect(
      visibleLineStageColumns(board, verify, false)
        .flatMap((column) => column.runs)
        .map((run) => run.workOrderId),
    ).toEqual(["wo-wait"]);
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
    const verify = collectLineVerifyOrders(board, [failed]);

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
    expect(collectLineVerifyOrders(board, [closed])).toEqual([]);
  });

  describe("when a custom stage is the last line stage", () => {
    const CUSTOM_KEY = "custom-docs";
    const CUSTOM_LAST_LINE: FactoriesFactoryLine = {
      id: "line-1",
      name: "poc",
      steps: [{ app: { app: "app-plan" } }, { app: { app: "app-docs" } }],
      stages: [
        { id: "backlog", key: "backlog", name: "Backlog", kind: "STAGE_KIND_BACKLOG" },
        { id: "phase-0", key: "phase-0", name: "Plan", kind: "STAGE_KIND_PHASE", stepIndex: 0 },
        { id: "docs", key: CUSTOM_KEY, name: "Update docs", kind: "STAGE_KIND_CUSTOM" },
        { id: "verify", key: "verify", name: "Verify", kind: "STAGE_KIND_VERIFY" },
        { id: "done", key: "done", name: "Done", kind: "STAGE_KIND_DONE" },
      ],
    };

    function orderWithCustomStage(id: string, customStep: Partial<TestExecution>): FactoriesWorkOrder {
      return order(id, id, [
        {
          id: `${id}-plan`,
          line: { id: "line-1", name: "poc" },
          step: "plan",
          stepIndex: 0,
          stageKey: "phase-0",
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          createdAt: "2026-08-11T12:00:00.000Z",
        },
        {
          id: `${id}-docs`,
          line: { id: "line-1", name: "poc" },
          step: "Update Docs",
          stepIndex: 1,
          stageKey: CUSTOM_KEY,
          createdAt: "2026-08-11T12:10:00.000Z",
          ...customStep,
        },
      ]);
    }

    it("moves a waiting order that passed the custom stage into Verify", () => {
      const waiting = orderWithCustomStage("wo-docs-passed", { state: "STATE_FINISHED", result: "RESULT_PASSED" });
      const board = buildLinePhaseBoard(CUSTOM_LAST_LINE, [waiting], APPS);

      const verify = collectLineVerifyOrders(board, [waiting], CUSTOM_LAST_LINE);

      expect(verify.map((entry) => entry.id)).toEqual(["wo-docs-passed"]);
    });

    it("moves a waiting order to Verify when the custom stage has no automation", () => {
      const waiting = order("wo-plan-passed", "wo-plan-passed", [
        {
          id: "wo-plan-passed-plan",
          line: { id: "line-1", name: "poc" },
          step: "plan",
          stepIndex: 0,
          stageKey: "phase-0",
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          createdAt: "2026-08-11T12:00:00.000Z",
        },
      ]);
      const line = { ...CUSTOM_LAST_LINE, steps: [{ app: { app: "app-plan" } }] };
      const board = buildLinePhaseBoard(line, [waiting], APPS);

      const verify = collectLineVerifyOrders(board, [waiting], line);

      expect(verify.map((entry) => entry.id)).toEqual(["wo-plan-passed"]);
    });

    it("keeps a running custom stage off Verify", () => {
      const running = orderWithCustomStage("wo-docs-running", { state: "STATE_STARTED" });
      const board = buildLinePhaseBoard(CUSTOM_LAST_LINE, [running], APPS);

      expect(collectLineVerifyOrders(board, [running], CUSTOM_LAST_LINE)).toEqual([]);
    });
  });

  it("keeps a task in its source-list place when updated_at changes", () => {
    const first = order("wo-first", "First review", [
      {
        id: "e-first",
        line: { id: "line-1", name: "poc" },
        step: "demo",
        stepIndex: 2,
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        createdAt: "2026-08-11T12:00:00.000Z",
        updatedAt: "2026-08-11T12:00:00.000Z",
      },
    ]);
    const second = order("wo-second", "Second review", [
      {
        id: "e-second",
        line: { id: "line-1", name: "poc" },
        step: "demo",
        stepIndex: 2,
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        createdAt: "2026-08-11T12:00:00.000Z",
        updatedAt: "2026-08-11T12:00:00.000Z",
      },
    ]);
    first.updatedAt = "2026-01-01T00:00:00.000Z";
    second.updatedAt = "2026-02-01T00:00:00.000Z";
    const bumpedSecond = { ...second, updatedAt: "2026-03-01T00:00:00.000Z" };
    const board = buildLinePhaseBoard(LINE, [first, bumpedSecond], APPS);

    expect(collectLineVerifyOrders(board, [first, bumpedSecond]).map((entry) => entry.id)).toEqual([
      "wo-first",
      "wo-second",
    ]);
  });
});
