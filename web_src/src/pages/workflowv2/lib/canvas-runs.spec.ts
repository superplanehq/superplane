import { describe, expect, it } from "vitest";
import type { CanvasesCanvasEventWithExecutions, CanvasesCanvasNodeExecutionRef, ComponentsNode } from "@/api-client";
import { filterRunEvents, getAggregateStatus } from "./canvas-runs";

function makeExecutionRef(overrides: Partial<CanvasesCanvasNodeExecutionRef> = {}): CanvasesCanvasNodeExecutionRef {
  return {
    id: "execution-1",
    nodeId: "node-1",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    ...overrides,
  } as CanvasesCanvasNodeExecutionRef;
}

function makeNode(overrides: Partial<ComponentsNode> = {}): ComponentsNode {
  return {
    id: "node-1",
    name: "Node 1",
    type: "TYPE_COMPONENT",
    component: {
      name: "noop",
    },
    ...overrides,
  } as ComponentsNode;
}

function makeEvent(overrides: Partial<CanvasesCanvasEventWithExecutions> = {}): CanvasesCanvasEventWithExecutions {
  return {
    id: "event-1",
    nodeId: "trigger-1",
    createdAt: "2026-04-01T12:00:00.000Z",
    data: {
      data: {},
      type: "event",
    },
    executions: [makeExecutionRef()],
    ...overrides,
  } as CanvasesCanvasEventWithExecutions;
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

describe("filterRunEvents", () => {
  const nodes = [
    makeNode({
      id: "trigger-1",
      name: "Deploy Trigger",
      type: "TYPE_TRIGGER",
      trigger: {
        name: "unknown-trigger",
      },
      component: undefined,
    }),
    makeNode({
      id: "node-success",
      name: "Ship Build",
    }),
    makeNode({
      id: "node-failed",
      name: "Run Checks",
    }),
    makeNode({
      id: "node-running",
      name: "Wait for Deploy",
    }),
  ];

  it("returns all events when the status filter is all", () => {
    const events = [
      makeEvent({
        id: "event-success",
        executions: [makeExecutionRef({ nodeId: "node-success" })],
      }),
      makeEvent({
        id: "event-failed",
        executions: [makeExecutionRef({ nodeId: "node-failed", result: "RESULT_FAILED" })],
      }),
      makeEvent({
        id: "event-queued",
        executions: [],
      }),
    ];

    expect(filterRunEvents(events, nodes, "all", "")).toEqual(events);
  });

  it("matches completed filter for both completed and cancelled runs", () => {
    const completedEvent = makeEvent({
      id: "event-completed",
      executions: [makeExecutionRef({ nodeId: "node-success", result: "RESULT_PASSED" })],
    });
    const cancelledEvent = makeEvent({
      id: "event-cancelled",
      executions: [makeExecutionRef({ nodeId: "node-success", result: "RESULT_CANCELLED" })],
    });
    const failedEvent = makeEvent({
      id: "event-failed",
      executions: [makeExecutionRef({ nodeId: "node-failed", result: "RESULT_FAILED" })],
    });

    expect(filterRunEvents([completedEvent, cancelledEvent, failedEvent], nodes, "completed", "")).toEqual([
      completedEvent,
      cancelledEvent,
    ]);
  });

  it("matches error, running, and queued filters from aggregate statuses", () => {
    const failedEvent = makeEvent({
      id: "event-failed",
      executions: [makeExecutionRef({ nodeId: "node-failed", result: "RESULT_FAILED" })],
    });
    const runningEvent = makeEvent({
      id: "event-running",
      executions: [makeExecutionRef({ nodeId: "node-running", state: "STATE_STARTED", result: undefined })],
    });
    const queuedEvent = makeEvent({
      id: "event-queued",
      executions: [],
    });

    expect(filterRunEvents([failedEvent, runningEvent, queuedEvent], nodes, "errors", "")).toEqual([failedEvent]);
    expect(filterRunEvents([failedEvent, runningEvent, queuedEvent], nodes, "running", "")).toEqual([runningEvent]);
    expect(filterRunEvents([failedEvent, runningEvent, queuedEvent], nodes, "queued", "")).toEqual([queuedEvent]);
  });

  it("matches search queries against event ids, node names, and execution messages", () => {
    const searchableEvent = makeEvent({
      id: "event-searchable",
      executions: [
        makeExecutionRef({
          nodeId: "node-failed",
          result: "RESULT_FAILED",
          resultMessage: "Approval timed out",
        }),
      ],
    });
    const otherEvent = makeEvent({
      id: "event-other",
      executions: [makeExecutionRef({ nodeId: "node-success", resultMessage: "Everything passed" })],
    });

    expect(filterRunEvents([searchableEvent, otherEvent], nodes, "all", "searchABLE")).toEqual([searchableEvent]);
    expect(filterRunEvents([searchableEvent, otherEvent], nodes, "all", "run checks")).toEqual([searchableEvent]);
    expect(filterRunEvents([searchableEvent, otherEvent], nodes, "all", "TIMED OUT")).toEqual([searchableEvent]);
  });

  it("applies status filtering before search matching", () => {
    const failedEvent = makeEvent({
      id: "event-failed",
      executions: [makeExecutionRef({ nodeId: "node-failed", result: "RESULT_FAILED" })],
    });
    const completedEvent = makeEvent({
      id: "event-completed",
      executions: [makeExecutionRef({ nodeId: "node-success", resultMessage: "Run checks finished" })],
    });

    expect(filterRunEvents([failedEvent, completedEvent], nodes, "errors", "run checks")).toEqual([failedEvent]);
  });
});
