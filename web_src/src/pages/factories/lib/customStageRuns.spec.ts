import { describe, expect, it } from "bun:test";
import type { FactoriesWorkOrder } from "@/api-client";

import { buildLinePhaseBoard, collectCustomStageRuns } from "./linePhaseRuns";

describe("collectCustomStageRuns", () => {
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
});
