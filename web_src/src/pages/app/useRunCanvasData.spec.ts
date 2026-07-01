import { describe, expect, it } from "vitest";
import type { CanvasesCanvasNodeExecution, CanvasesCanvasRun } from "@/api-client";
import { getRunCanvasFitKey, mergeRunExecutionsForCanvas, type RunCanvasData } from "./useRunCanvasData";

function makeRunExecutionRef(id: string, nodeId: string): NonNullable<CanvasesCanvasRun["executions"]>[number] {
  return {
    id,
    nodeId,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
  };
}

function makeFullExecution(id: string, nodeId: string): CanvasesCanvasNodeExecution {
  return {
    id,
    nodeId,
    canvasId: "canvas-1",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    outputs: { result: "ok" },
  };
}

describe("mergeRunExecutionsForCanvas", () => {
  it("uses full executions when the selected run only has summary data", () => {
    const executions = mergeRunExecutionsForCanvas(
      [],
      [makeFullExecution("execution-1", "trigger-node"), makeFullExecution("execution-2", "selected-node")],
    );

    expect(executions.map((execution) => execution.nodeId)).toEqual(["trigger-node", "selected-node"]);
  });

  it("keeps summary execution order and appends full executions missing from the summary", () => {
    const executions = mergeRunExecutionsForCanvas(
      [makeRunExecutionRef("execution-1", "trigger-node")],
      [makeFullExecution("execution-1", "trigger-node"), makeFullExecution("execution-2", "selected-node")],
    );

    expect(executions.map((execution) => execution.id)).toEqual(["execution-1", "execution-2"]);
    expect(executions.map((execution) => execution.nodeId)).toEqual(["trigger-node", "selected-node"]);
  });
});

describe("getRunCanvasFitKey", () => {
  const runCanvasData: RunCanvasData = {
    nodes: [],
    edges: [],
    participantNodeIds: ["selected-node", "trigger-node"],
  };

  it("does not request a run canvas fit while execution data is loading", () => {
    expect(
      getRunCanvasFitKey({
        isRunInspectionMode: true,
        selectedRunId: "run-1",
        runCanvasData,
        runCanvasLoading: true,
      }),
    ).toBeNull();
  });

  it("builds a stable fit key after the run canvas finishes loading", () => {
    expect(
      getRunCanvasFitKey({
        isRunInspectionMode: true,
        selectedRunId: "run-1",
        runCanvasData,
        runCanvasLoading: false,
      }),
    ).toBe("run-1|selected-node|trigger-node");
  });
});
