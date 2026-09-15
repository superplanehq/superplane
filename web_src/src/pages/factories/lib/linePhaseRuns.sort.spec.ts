import { describe, expect, it } from "bun:test";
import type {
  FactoriesFactoryLine,
  FactoriesLineRef,
  FactoriesWorkOrder,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
} from "@/api-client";
import { buildLinePhaseBoard, collectLineBacklogOrders } from "./linePhaseRuns";

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

describe("linePhaseRuns column sort", () => {
  it("sorts a phase by work order created time, not execution time", () => {
    const olderCreated = {
      ...order("wo-old", "Older created", [
        {
          id: "e-late",
          line: { id: "line-1", name: "poc" },
          step: "plan",
          stepIndex: 0,
          state: "STATE_STARTED",
          createdAt: "2026-08-11T16:00:00.000Z",
          updatedAt: "2026-08-11T16:00:00.000Z",
        },
      ]),
      createdAt: "2026-08-11T09:00:00.000Z",
    };
    const newerCreated = {
      ...order("wo-new", "Newer created", [
        {
          id: "e-early",
          line: { id: "line-1", name: "poc" },
          step: "plan",
          stepIndex: 0,
          state: "STATE_STARTED",
          createdAt: "2026-08-11T10:00:00.000Z",
          updatedAt: "2026-08-11T10:00:00.000Z",
        },
      ]),
      createdAt: "2026-08-11T12:00:00.000Z",
    };

    const byActivity = buildLinePhaseBoard(LINE, [olderCreated, newerCreated], APPS);
    expect(byActivity[0].runs.map((run) => run.workOrderId)).toEqual(["wo-old", "wo-new"]);

    const byCreated = buildLinePhaseBoard(LINE, [olderCreated, newerCreated], APPS, { 0: "created" });
    expect(byCreated[0].runs.map((run) => run.workOrderId)).toEqual(["wo-new", "wo-old"]);
    expect(byCreated[1].runs).toEqual([]);
  });

  it("sorts drafts by work order created time when requested", () => {
    const older: FactoriesWorkOrder = {
      id: "wo-older",
      title: "Older",
      state: "STATE_DRAFT",
      createdAt: "2026-08-11T10:00:00.000Z",
      updatedAt: "2026-08-11T18:00:00.000Z",
    };
    const newer: FactoriesWorkOrder = {
      id: "wo-newer",
      title: "Newer",
      state: "STATE_DRAFT",
      createdAt: "2026-08-11T12:00:00.000Z",
      updatedAt: "2026-08-11T13:00:00.000Z",
    };

    expect(collectLineBacklogOrders([older, newer]).map((entry) => entry.id)).toEqual(["wo-older", "wo-newer"]);
    expect(collectLineBacklogOrders([older, newer], "created").map((entry) => entry.id)).toEqual([
      "wo-newer",
      "wo-older",
    ]);
  });

  it("sorts drafts by confidence when scores are ready", () => {
    const low: FactoriesWorkOrder = { id: "wo-low", title: "Low", state: "STATE_DRAFT" };
    const high: FactoriesWorkOrder = { id: "wo-high", title: "High", state: "STATE_DRAFT" };
    const missing: FactoriesWorkOrder = { id: "wo-missing", title: "Missing", state: "STATE_DRAFT" };
    const scores = new Map<string, number | undefined>([
      ["wo-high", 5],
      ["wo-low", 2],
    ]);

    expect(collectLineBacklogOrders([missing, low, high], "confidence", scores).map((entry) => entry.id)).toEqual([
      "wo-high",
      "wo-low",
      "wo-missing",
    ]);
  });
});
