import { describe, expect, it } from "bun:test";

import { loopMapper } from "./loop";
import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "./types";

function buildDetailsContext(overrides: Partial<ExecutionInfo> = {}): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Loop",
    componentName: "loop",
    isCollapsed: false,
  };
  return {
    nodes: [node],
    node,
    execution: {
      id: "exec-1",
      createdAt: "",
      updatedAt: new Date().toISOString(),
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      resultReason: "RESULT_REASON_OK",
      resultMessage: "",
      metadata: undefined,
      configuration: {},
      rootEvent: undefined,
      ...overrides,
    },
  };
}

function buildOutput(data: unknown): OutputPayload {
  return { type: "loop.done", timestamp: new Date().toISOString(), data };
}

const baseDetails = {
  "Evaluated at": "-",
  "Until expression": "-",
  "Max iterations": 10,
  "Timeout (s)": 3600,
};

describe("loopMapper.getExecutionDetails", () => {
  it.each([undefined, null, {}, { iteration: 0 }, { iteration: 1, waitingBetweenIterations: true }])(
    "keeps defaults and omits progress for metadata without active: %j",
    (metadata) => {
      expect(loopMapper.getExecutionDetails(buildDetailsContext({ metadata }))).toStrictEqual(baseDetails);
    },
  );

  it.each([{}, { done: [] }, { done: [buildOutput({})] }, { done: [buildOutput({ done: {} })] }])(
    "omits completion details for missing or incomplete output: %j",
    (outputs) => {
      expect(loopMapper.getExecutionDetails(buildDetailsContext({ outputs }))).toStrictEqual(baseDetails);
    },
  );

  it("preserves false and zero values while omitting false waiting status and empty stop reason", () => {
    const details = loopMapper.getExecutionDetails(
      buildDetailsContext({
        metadata: { active: false, iteration: 0, waitingBetweenIterations: false },
        outputs: { done: [buildOutput({ done: { iterations: 0, elapsedMs: 0, stopReason: "" } })] },
        configuration: { delayBetweenIterations: { enabled: false, strategy: "exponential", intervalSeconds: 5 } },
      }),
    );

    expect(details).toStrictEqual({
      ...baseDetails,
      "Current iteration": 0,
      Active: false,
      Iterations: 0,
      "Elapsed (ms)": 0,
    });
    for (const key of ["Waiting between iterations", "Stop reason", "Delay strategy", "Delay interval (s)"]) {
      expect(details).not.toHaveProperty(key);
    }
  });

  it("omits optional progress and completion fields with nonnumeric or nonboolean values", () => {
    const details = loopMapper.getExecutionDetails(
      buildDetailsContext({
        metadata: { active: "false", iteration: "0" },
        outputs: { done: [buildOutput({ done: { iterations: "0", elapsedMs: null } })] },
      }),
    );

    expect(details).toStrictEqual(baseDetails);
  });

  it("preserves labels and insertion order and reads only the first done output", () => {
    const details = loopMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          untilExpression: "iteration > 2",
          maxIterations: 3,
          timeoutSeconds: 60,
          delayBetweenIterations: { enabled: true },
        },
        metadata: { active: true, iteration: 2, waitingBetweenIterations: true },
        outputs: {
          done: [
            buildOutput({ done: { iterations: 2, stopReason: "condition_met", elapsedMs: 100 } }),
            buildOutput({ done: { iterations: 3, stopReason: "max_iterations", elapsedMs: 200 } }),
          ],
        },
      }),
    );

    expect(Object.entries(details)).toEqual([
      ["Evaluated at", "-"],
      ["Until expression", "iteration > 2"],
      ["Max iterations", 3],
      ["Timeout (s)", 60],
      ["Current iteration", 2],
      ["Active", true],
      ["Waiting between iterations", true],
      ["Iterations", 2],
      ["Stop reason", "condition_met"],
      ["Elapsed (ms)", 100],
      ["Delay strategy", "fixed"],
      ["Delay interval (s)", 0],
    ]);
  });

  it("does not search later done outputs when the first lacks completion data", () => {
    const details = loopMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          done: [
            buildOutput({}),
            buildOutput({ done: { iterations: 2, elapsedMs: 100, stopReason: "condition_met" } }),
          ],
        },
      }),
    );

    expect(details).toStrictEqual(baseDetails);
  });
});
