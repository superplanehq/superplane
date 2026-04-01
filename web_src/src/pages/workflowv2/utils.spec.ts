import { describe, expect, it, vi } from "vitest";
import type { CanvasesCanvasNodeExecutionRef, ComponentsNode } from "@/api-client";
import { buildRunItemFromExecutionRef } from "./utils";

function makeExecutionRef(overrides: Partial<CanvasesCanvasNodeExecutionRef> = {}): CanvasesCanvasNodeExecutionRef {
  return {
    id: "execution-1",
    nodeId: "node-1",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    createdAt: "2026-04-01T12:00:00Z",
    updatedAt: "2026-04-01T12:00:01Z",
    ...overrides,
  } as CanvasesCanvasNodeExecutionRef;
}

function makeNode(overrides: Partial<ComponentsNode> = {}): ComponentsNode {
  return {
    id: "node-1",
    name: "Node 1",
    type: "TYPE_COMPONENT",
    ...overrides,
  } as ComponentsNode;
}

describe("buildRunItemFromExecutionRef", () => {
  it("marks failed execution refs as error when no resolved state is provided", () => {
    const runItem = buildRunItemFromExecutionRef({
      execution: makeExecutionRef({ result: "RESULT_FAILED" }),
      nodes: [makeNode()],
      onNodeSelect: vi.fn(),
    });

    expect(runItem.type).toBe("error");
  });

  it("preserves resolved-error typing for resolved failures", () => {
    const runItem = buildRunItemFromExecutionRef({
      execution: makeExecutionRef({
        result: "RESULT_FAILED",
        resultReason: "RESULT_REASON_ERROR_RESOLVED",
      }),
      nodes: [makeNode()],
      onNodeSelect: vi.fn(),
    });

    expect(runItem.type).toBe("resolved-error");
  });
});
