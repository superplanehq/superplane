import { describe, expect, it } from "vitest";
import type { CanvasesCanvasNodeExecutionRef, CanvasesCanvasRun } from "@/api-client";
import { makeComponentsNode } from "@/test/factories";
import { filterRuns, getAggregateStatus } from "./canvas-runs";

function makeExecutionRef(overrides: Partial<CanvasesCanvasNodeExecutionRef> = {}): CanvasesCanvasNodeExecutionRef {
  return {
    id: "execution-1",
    nodeId: "node-1",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    ...overrides,
  } as CanvasesCanvasNodeExecutionRef;
}

function makeRun(overrides: Partial<CanvasesCanvasRun> = {}): CanvasesCanvasRun {
  return {
    id: "run-1",
    canvasId: "canvas-1",
    createdAt: "2026-04-01T12:00:00.000Z",
    rootEvent: {
      id: "event-1",
      nodeId: "trigger-1",
      createdAt: "2026-04-01T12:00:00.000Z",
      data: {
        data: {},
        type: "event",
      },
    },
    executions: [makeExecutionRef()],
    ...overrides,
  } as CanvasesCanvasRun;
}

describe("getAggregateStatus", () => {
  it("returns running when any execution is pending", () => {
    const executions = [
      makeExecutionRef({ state: "STATE_PENDING", result: undefined }),
      makeExecutionRef({ id: "execution-2", result: "RESULT_FAILED" }),
    ];

    expect(getAggregateStatus(executions)).toBe("running");
  });

  it("returns running when any execution is started", () => {
    const executions = [
      makeExecutionRef({ state: "STATE_STARTED", result: undefined }),
      makeExecutionRef({ id: "execution-2", result: "RESULT_CANCELLED" }),
    ];

    expect(getAggregateStatus(executions)).toBe("running");
  });

  it("returns error before cancelled when a failed execution is present", () => {
    const executions = [
      makeExecutionRef({ result: "RESULT_CANCELLED" }),
      makeExecutionRef({ id: "execution-2", result: "RESULT_FAILED" }),
    ];

    expect(getAggregateStatus(executions)).toBe("error");
  });

  it("returns cancelled when no executions failed and one was cancelled", () => {
    const executions = [
      makeExecutionRef({ result: "RESULT_CANCELLED" }),
      makeExecutionRef({ id: "execution-2", result: "RESULT_PASSED" }),
    ];

    expect(getAggregateStatus(executions)).toBe("cancelled");
  });

  it("returns completed when every execution passed", () => {
    const executions = [makeExecutionRef(), makeExecutionRef({ id: "execution-2" })];

    expect(getAggregateStatus(executions)).toBe("completed");
  });

  it("returns completed when every execution finished even without passed results", () => {
    const executions = [
      makeExecutionRef({ result: undefined }),
      makeExecutionRef({ id: "execution-2", result: undefined }),
    ];

    expect(getAggregateStatus(executions)).toBe("completed");
  });

  it("returns queued when executions are neither running nor finished", () => {
    const executions = [
      makeExecutionRef({ state: "STATE_UNKNOWN", result: undefined }),
      makeExecutionRef({ id: "execution-2", state: "STATE_UNKNOWN", result: undefined }),
    ];

    expect(getAggregateStatus(executions)).toBe("queued");
  });
});

describe("filterRuns", () => {
  const nodes = [
    makeComponentsNode({
      id: "trigger-1",
      name: "Deploy Trigger",
      type: "TYPE_TRIGGER",
      component: "unknown-trigger",
    }),
    makeComponentsNode({
      id: "node-success",
      name: "Ship Build",
    }),
    makeComponentsNode({
      id: "node-failed",
      name: "Run Checks",
    }),
    makeComponentsNode({
      id: "node-running",
      name: "Wait for Deploy",
    }),
  ];

  it("returns all runs when the status filter is all", () => {
    const runs = [
      makeRun({
        id: "run-success",
        rootEvent: { id: "event-success", nodeId: "trigger-1" },
        executions: [makeExecutionRef({ nodeId: "node-success" })],
      }),
      makeRun({
        id: "run-failed",
        rootEvent: { id: "event-failed", nodeId: "trigger-1" },
        executions: [makeExecutionRef({ nodeId: "node-failed", result: "RESULT_FAILED" })],
      }),
      makeRun({
        id: "run-queued",
        rootEvent: { id: "event-queued", nodeId: "trigger-1" },
        executions: [],
      }),
    ];

    expect(filterRuns(runs, nodes, "all", "")).toEqual(runs);
  });

  it("matches completed filter for both completed and cancelled runs", () => {
    const completedRun = makeRun({
      id: "run-completed",
      rootEvent: { id: "event-completed", nodeId: "trigger-1" },
      executions: [makeExecutionRef({ nodeId: "node-success", result: "RESULT_PASSED" })],
    });
    const cancelledRun = makeRun({
      id: "run-cancelled",
      rootEvent: { id: "event-cancelled", nodeId: "trigger-1" },
      executions: [makeExecutionRef({ nodeId: "node-success", result: "RESULT_CANCELLED" })],
    });
    const failedRun = makeRun({
      id: "run-failed",
      rootEvent: { id: "event-failed", nodeId: "trigger-1" },
      executions: [makeExecutionRef({ nodeId: "node-failed", result: "RESULT_FAILED" })],
    });

    expect(filterRuns([completedRun, cancelledRun, failedRun], nodes, "completed", "")).toEqual([
      completedRun,
      cancelledRun,
    ]);
  });

  it("matches error, running, and queued filters from aggregate statuses", () => {
    const failedRun = makeRun({
      id: "run-failed",
      rootEvent: { id: "event-failed", nodeId: "trigger-1" },
      executions: [makeExecutionRef({ nodeId: "node-failed", result: "RESULT_FAILED" })],
    });
    const runningRun = makeRun({
      id: "run-running",
      rootEvent: { id: "event-running", nodeId: "trigger-1" },
      executions: [makeExecutionRef({ nodeId: "node-running", state: "STATE_STARTED", result: undefined })],
    });
    const queuedRun = makeRun({
      id: "run-queued",
      rootEvent: { id: "event-queued", nodeId: "trigger-1" },
      executions: [],
    });

    expect(filterRuns([failedRun, runningRun, queuedRun], nodes, "errors", "")).toEqual([failedRun]);
    expect(filterRuns([failedRun, runningRun, queuedRun], nodes, "running", "")).toEqual([runningRun]);
    expect(filterRuns([failedRun, runningRun, queuedRun], nodes, "queued", "")).toEqual([queuedRun]);
  });

  it("matches search queries against event ids, node names, and execution messages", () => {
    const searchableRun = makeRun({
      id: "run-searchable",
      rootEvent: { id: "event-searchable", nodeId: "trigger-1" },
      executions: [
        makeExecutionRef({
          nodeId: "node-failed",
          result: "RESULT_FAILED",
          resultMessage: "Approval timed out",
        }),
      ],
    });
    const otherRun = makeRun({
      id: "run-other",
      rootEvent: { id: "event-other", nodeId: "trigger-1" },
      executions: [makeExecutionRef({ nodeId: "node-success", resultMessage: "Everything passed" })],
    });

    expect(filterRuns([searchableRun, otherRun], nodes, "all", "searchABLE")).toEqual([searchableRun]);
    expect(filterRuns([searchableRun, otherRun], nodes, "all", "run checks")).toEqual([searchableRun]);
    expect(filterRuns([searchableRun, otherRun], nodes, "all", "TIMED OUT")).toEqual([searchableRun]);
  });

  it("applies status filtering before search matching", () => {
    const failedRun = makeRun({
      id: "run-failed",
      rootEvent: { id: "event-failed", nodeId: "trigger-1" },
      executions: [makeExecutionRef({ nodeId: "node-failed", result: "RESULT_FAILED" })],
    });
    const completedRun = makeRun({
      id: "run-completed",
      rootEvent: { id: "event-completed", nodeId: "trigger-1" },
      executions: [makeExecutionRef({ nodeId: "node-success", resultMessage: "Run checks finished" })],
    });

    expect(filterRuns([failedRun, completedRun], nodes, "errors", "run checks")).toEqual([failedRun]);
  });
});
