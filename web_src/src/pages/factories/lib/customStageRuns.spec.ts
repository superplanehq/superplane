import { describe, expect, it } from "bun:test";
import type { FactoriesWorkOrder, FactoriesWorkOrderExecution, FactoryLineStage } from "@/api-client";

import { buildLinePhaseBoard, collectCustomStageRuns } from "./linePhaseRuns";

function taskWithExecution(execution: FactoriesWorkOrderExecution): FactoriesWorkOrder {
  return {
    id: "wo-stage",
    state: "STATE_OPEN",
    lineDispatches: [{ line: { id: "line-1" }, stepExecutions: [{ state: "STATE_STARTED", ...execution }] }],
  };
}

describe("collectCustomStageRuns", () => {
  it.each([undefined, -1, 2])("uses the snapshot when the saved stage index is %s", (stepIndex) => {
    const task = taskWithExecution({ stepIndex: 1, stageKey: "phase-saved" });
    const board = buildLinePhaseBoard(
      {
        id: "line-1",
        steps: [{ app: { app: "app-plan" } }, { app: { app: "app-build" } }],
        stages: [{ key: "phase-saved", stepIndex }],
      },
      [task],
    );

    expect(board.map((column) => column.runs.length)).toEqual([0, 1]);
  });

  it.each([undefined, "   "])("accepts saved index zero with execution app ID %s", (appId) => {
    const task = taskWithExecution({ stepIndex: 1, stageKey: "phase-zero", run: { appId } });
    const board = buildLinePhaseBoard(
      {
        id: "line-1",
        steps: [{ app: { app: "app-plan" } }, { app: { app: "app-build" } }],
        stages: [{ key: "phase-zero", stepIndex: 0 }],
      },
      [task],
    );

    expect(board.map((column) => column.runs.length)).toEqual([1, 0]);
  });

  it.each<FactoryLineStage>([
    { key: "qa", kind: "STAGE_KIND_CUSTOM", stepIndex: 0 },
    { key: "custom-qa", kind: "STAGE_KIND_PHASE", stepIndex: 0 },
  ])("excludes custom stage $key even when the snapshot matches", (stage) => {
    const task = taskWithExecution({ stepIndex: 0, stageKey: stage.key, run: { appId: "app-plan" } });
    const board = buildLinePhaseBoard({ id: "line-1", steps: [{ app: { app: "app-plan" } }], stages: [stage] }, [task]);

    expect(board.flatMap((column) => column.runs)).toEqual([]);
  });

  it.each([
    { snapshotIndex: 3, expectedRuns: [0, 0, 1, 0, 0] },
    { snapshotIndex: 5, expectedRuns: [0, 0, 0, 0, 1] },
  ])("selects the closest repeated app from snapshot $snapshotIndex", ({ snapshotIndex, expectedRuns }) => {
    const task = taskWithExecution({ stepIndex: snapshotIndex, stageKey: "phase-stale", run: { appId: "app-plan" } });
    const board = buildLinePhaseBoard(
      {
        id: "line-1",
        steps: ["app-plan", "app-build", "app-plan", "app-build", "app-plan"].map((app) => ({ app: { app } })),
        stages: [{ key: "phase-stale", stepIndex: 1 }],
      },
      [task],
    );

    expect(board.map((column) => column.runs.length)).toEqual(expectedRuns);
  });

  it("uses the snapshot when the saved stage is missing", () => {
    const task = taskWithExecution({ stepIndex: 1, stageKey: "phase-missing", run: { appId: "app-build" } });
    const board = buildLinePhaseBoard(
      { id: "line-1", steps: [{ app: { app: "app-plan" } }, { app: { app: "app-build" } }] },
      [task],
    );

    expect(board.map((column) => column.runs.length)).toEqual([0, 1]);
  });

  it("places a task in its current custom stage", () => {
    const task: FactoriesWorkOrder = {
      id: "wo-custom",
      title: "Custom",
      state: "STATE_OPEN",
      lineDispatches: [
        {
          id: "dispatch-custom",
          line: { id: "line-1", name: "Main" },
          createdAt: "2026-08-11T10:00:00.000Z",
          stepExecutions: [
            {
              id: "execution-custom",
              step: "Lint",
              stepIndex: 0,
              stageKey: "custom-qa",
              state: "STATE_STARTED",
              createdAt: "2026-08-11T10:00:00.000Z",
            },
          ],
        },
      ],
    };

    expect(collectCustomStageRuns("line-1", "custom-qa", [task]).map((run) => run.workOrderId)).toEqual(["wo-custom"]);
  });

  it("shows a failed parallel automation as the stage result", () => {
    const task: FactoriesWorkOrder = {
      id: "wo-failed",
      state: "STATE_OPEN",
      lineDispatches: [
        {
          line: { id: "line-1" },
          stepExecutions: [
            {
              id: "execution-failed",
              stepIndex: 0,
              stageKey: "custom-qa",
              state: "STATE_FINISHED",
              result: "RESULT_FAILED",
            },
            {
              id: "execution-passed",
              stepIndex: 1,
              stageKey: "custom-qa",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
            },
          ],
        },
      ],
    };

    expect(collectCustomStageRuns("line-1", "custom-qa", [task])[0]?.execution.id).toBe("execution-failed");
  });

  it("does not leave the task in its previous phase lane", () => {
    const task: FactoriesWorkOrder = {
      id: "wo-moving",
      state: "STATE_OPEN",
      lineDispatches: [
        {
          line: { id: "line-1" },
          stepExecutions: [
            {
              id: "phase-passed",
              stepIndex: 0,
              stageKey: "phase-0",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              run: { appId: "app-implement" },
            },
            {
              id: "custom-running",
              stepIndex: 1,
              stageKey: "custom-qa",
              state: "STATE_STARTED",
            },
          ],
        },
      ],
    };
    const board = buildLinePhaseBoard(
      { id: "line-1", steps: [{ app: { app: "app-implement" } }] },
      [task],
      [{ id: "app-implement", name: "Implement" }],
    );

    expect(board[0]?.runs).toHaveLength(0);
    expect(collectCustomStageRuns("line-1", "custom-qa", [task])).toHaveLength(1);
  });

  it("places a waiting task in an empty current stage", () => {
    const task: FactoriesWorkOrder = {
      id: "wo-empty",
      state: "STATE_OPEN",
      lineDispatches: [
        {
          line: { id: "line-1" },
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          currentStageKey: "custom-empty",
          stepExecutions: [
            {
              id: "phase-passed",
              stepIndex: 0,
              stageKey: "phase-0",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              run: { appId: "app-implement" },
            },
          ],
        },
      ],
    };
    const board = buildLinePhaseBoard(
      { id: "line-1", steps: [{ app: { app: "app-implement" } }] },
      [task],
      [{ id: "app-implement", name: "Implement" }],
    );

    expect(board[0]?.runs).toHaveLength(0);
    expect(collectCustomStageRuns("line-1", "custom-empty", [task])).toHaveLength(1);
  });

  it("places a repeated app by its saved phase key, not the shifted step index", () => {
    const task: FactoriesWorkOrder = {
      id: "wo-repeated",
      state: "STATE_OPEN",
      lineDispatches: [
        {
          line: { id: "line-1" },
          stepExecutions: [
            {
              id: "phase-0-running",
              stepIndex: 1,
              stageKey: "phase-0",
              state: "STATE_STARTED",
              run: { appId: "app-repeated" },
            },
          ],
        },
      ],
    };
    const board = buildLinePhaseBoard(
      {
        id: "line-1",
        steps: [{ app: { app: "app-repeated" } }, { app: { app: "app-repeated" } }],
        stages: [
          { key: "custom-qa", kind: "STAGE_KIND_CUSTOM" },
          { key: "phase-0", kind: "STAGE_KIND_PHASE", stepIndex: 0 },
          { key: "phase-1", kind: "STAGE_KIND_PHASE", stepIndex: 1 },
        ],
      },
      [task],
      [{ id: "app-repeated", name: "Repeated" }],
    );

    expect(board.map((column) => column.runs.length)).toEqual([1, 0]);
  });

  it("keeps a running task with its app after the steps are reordered", () => {
    const task: FactoriesWorkOrder = {
      id: "wo-reordered",
      state: "STATE_OPEN",
      lineDispatches: [
        {
          line: { id: "line-1" },
          stepExecutions: [
            {
              id: "app-a-running",
              stepIndex: 0,
              stageKey: "phase-0",
              state: "STATE_STARTED",
              run: { appId: "app-a" },
            },
          ],
        },
      ],
    };
    const board = buildLinePhaseBoard(
      {
        id: "line-1",
        steps: [{ app: { app: "app-b" } }, { app: { app: "app-a" } }],
        stages: [
          { key: "phase-0", kind: "STAGE_KIND_PHASE", stepIndex: 0 },
          { key: "phase-1", kind: "STAGE_KIND_PHASE", stepIndex: 1 },
        ],
      },
      [task],
      [
        { id: "app-a", name: "A" },
        { id: "app-b", name: "B" },
      ],
    );

    expect(board.map((column) => column.runs.length)).toEqual([0, 1]);
  });
});
