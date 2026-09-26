import { describe, expect, it } from "vitest";

import {
  WAIT_FOR_ENDPOINT_STATE_REGISTRY,
  waitForEndpointMapper,
  waitForEndpointStateFunction,
} from "./waitForEndpoint";
import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload, SubtitleContext } from "./types";

function node(): NodeInfo {
  return {
    id: "endpoint-node",
    name: "Wait for API",
    componentName: "waitForEndpoint",
    isCollapsed: false,
    configuration: {
      url: "https://api.example.com/ready",
      method: "GET",
      expectedStatus: "2xx",
      intervalSeconds: 10,
      timeoutSeconds: 300,
    },
  };
}

function execution(overrides: Partial<ExecutionInfo> = {}): ExecutionInfo {
  return {
    id: "execution-1",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    state: "STATE_STARTED",
    result: "RESULT_UNKNOWN",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: node().configuration,
    rootEvent: undefined,
    outputs: undefined,
    ...overrides,
  };
}

function payload(data: Record<string, unknown>): OutputPayload {
  return {
    type: "endpoint.test",
    timestamp: new Date().toISOString(),
    data,
  };
}

describe("waitForEndpointStateFunction", () => {
  it("shows running while checks are scheduled", () => {
    expect(waitForEndpointStateFunction(execution())).toBe("running");
  });

  it("shows ready for the ready channel", () => {
    expect(
      waitForEndpointStateFunction(
        execution({
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          outputs: { ready: [payload({ status: 200 })] },
        }),
      ),
    ).toBe("ready");
  });

  it("shows timeout for the timeout channel", () => {
    expect(
      waitForEndpointStateFunction(
        execution({
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          outputs: { timeout: [payload({ attempts: 5 })] },
        }),
      ),
    ).toBe("timeout");
  });

  it("registers the endpoint state map", () => {
    expect(WAIT_FOR_ENDPOINT_STATE_REGISTRY.getState).toBe(waitForEndpointStateFunction);
  });
});

describe("waitForEndpointMapper", () => {
  it("shows attempts and last status while running", () => {
    const context: SubtitleContext = {
      node: node(),
      execution: execution({
        metadata: { attempts: 3, lastStatus: 503 },
      }),
    };

    expect(waitForEndpointMapper.subtitle(context)).toBe("Attempt 3 - last status 503");
  });

  it("shows timeout details", () => {
    const currentNode = node();
    const currentExecution = execution({
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      metadata: { attempts: 4, lastStatus: 503 },
      outputs: {
        timeout: [payload({ attempts: 4, lastStatus: 503, elapsedMs: 30000, reason: "deadline_exceeded" })],
      },
    });
    const context: ExecutionDetailsContext = {
      nodes: [currentNode],
      node: currentNode,
      execution: currentExecution,
    };

    expect(waitForEndpointMapper.getExecutionDetails(context)).toMatchObject({
      Endpoint: "GET https://api.example.com/ready",
      "Expected Status": "2xx",
      Attempts: "4",
      "Last Status": "503",
      Elapsed: "30s",
    });
    expect(waitForEndpointMapper.subtitle({ node: currentNode, execution: currentExecution })).toBe(
      "Timed out after 4 attempts",
    );
  });
});
