import { describe, expect, it, vi } from "vitest";
import type { CanvasesCanvasNodeExecutionRef } from "@/api-client";
import { makeComponentsNode } from "@/test/factories";
import type { LogEntry } from "@/ui/CanvasLogSidebar";
import { buildRunItemFromExecutionRef, mapCanvasNodesToLogEntries, mergeWorkflowLogEntries } from "./utils";

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

describe("buildRunItemFromExecutionRef", () => {
  it("marks failed execution refs as error when no resolved state is provided", () => {
    const runItem = buildRunItemFromExecutionRef({
      execution: makeExecutionRef({ result: "RESULT_FAILED" }),
      nodes: [makeComponentsNode()],
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
      nodes: [makeComponentsNode()],
      onNodeSelect: vi.fn(),
    });

    expect(runItem.type).toBe("resolved-error");
  });
});

describe("mergeWorkflowLogEntries", () => {
  it("keeps canvas warnings visible outside live mode", () => {
    const canvasEntries = mapCanvasNodesToLogEntries({
      nodes: [
        makeComponentsNode({
          id: "draft-node-newer",
          name: "Draft Node Newer",
          warningMessage: "Newer warning",
        }),
        makeComponentsNode({
          id: "draft-node-older",
          name: "Draft Node Older",
          warningMessage: "Older warning",
        }),
      ],
      workflowUpdatedAt: "2026-04-03T12:00:00Z",
      onNodeSelect: vi.fn(),
    });

    const result = mergeWorkflowLogEntries({
      isViewingLiveVersion: false,
      runEntries: [
        {
          id: "run-1",
          source: "runs",
          timestamp: "2026-04-04T12:00:00Z",
          title: "Live run",
          type: "run",
        } satisfies LogEntry,
      ],
      liveRunEntries: [],
      canvasEntries: [canvasEntries[0]!],
      liveCanvasEntries: [
        {
          ...canvasEntries[1]!,
          timestamp: "2026-04-02T12:00:00Z",
        },
      ],
      resolvedExecutionIds: new Set(["execution-1"]),
    });

    expect(result).toHaveLength(2);
    expect(result.map((entry) => entry.id)).toEqual(["warning-2", "warning-1"]);
    expect(result[0]?.type).toBe("warning");
    expect(result[0]?.source).toBe("canvas");
    expect(result[0]?.searchText).toContain("Older warning");
  });

  it("preserves resolved run item state in live mode", () => {
    const result = mergeWorkflowLogEntries({
      isViewingLiveVersion: true,
      runEntries: [
        {
          id: "run-1",
          source: "runs",
          timestamp: "2026-04-04T12:00:00Z",
          title: "Live run",
          type: "run",
          runItems: [
            {
              id: "execution-1",
              type: "error",
              title: "Execution failed",
              timestamp: "2026-04-04T12:00:00Z",
            },
          ],
        } satisfies LogEntry,
      ],
      liveRunEntries: [],
      canvasEntries: [],
      liveCanvasEntries: [],
      resolvedExecutionIds: new Set(["execution-1"]),
    });

    expect(result[0]?.runItems?.[0]?.type).toBe("resolved-error");
  });
});
