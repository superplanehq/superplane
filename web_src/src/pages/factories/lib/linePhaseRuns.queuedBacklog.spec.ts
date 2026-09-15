import { describe, expect, it } from "bun:test";
import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
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

describe("queued tasks on the line board", () => {
  it("keeps a first-step queued task in Backlog instead of the first phase", () => {
    const queued: FactoriesWorkOrder = {
      id: "wo-queued",
      title: "Queued start",
      state: "STATE_OPEN",
      updatedAt: "2026-08-11T16:00:00.000Z",
      lineDispatches: [
        {
          id: "dispatch-wo-queued",
          line: { id: "line-1", name: "poc" },
          state: "STATE_ACTIVE",
          createdAt: "2026-08-11T16:00:00.000Z",
          stepExecutions: [],
          queueItem: {
            id: "q-wo-queued",
            stepName: "plan",
            stepIndex: 0,
            position: 1,
            appId: "app-plan",
          },
        },
      ],
    };

    expect(collectLineBacklogOrders([queued], LINE.id).map((entry) => entry.id)).toEqual(["wo-queued"]);
    expect(buildLinePhaseBoard(LINE, [queued], APPS).flatMap((column) => column.runs)).toEqual([]);
  });

  it("places a later-step queued task on that phase column", () => {
    const queued: FactoriesWorkOrder = {
      id: "wo-queued-build",
      title: "Queued build",
      state: "STATE_OPEN",
      lineDispatches: [
        {
          id: "dispatch-wo-queued-build",
          line: { id: "line-1", name: "poc" },
          state: "STATE_ACTIVE",
          createdAt: "2026-08-11T16:00:00.000Z",
          stepExecutions: [
            {
              id: "e-plan",
              step: "plan",
              stepIndex: 0,
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-11T16:00:00.000Z",
              updatedAt: "2026-08-11T16:00:00.000Z",
              run: { appId: "app-plan" },
            },
          ],
          queueItem: {
            id: "q-wo-queued-build",
            stepName: "build",
            stepIndex: 1,
            position: 2,
            appId: "app-build",
          },
        },
      ],
    };

    expect(collectLineBacklogOrders([queued], LINE.id)).toEqual([]);
    const board = buildLinePhaseBoard(LINE, [queued], APPS);
    expect(board[0]?.runs.map((run) => run.workOrderId)).toEqual([]);
    expect(board[1]?.runs.map((run) => run.workOrderId)).toEqual(["wo-queued-build"]);
  });
});
