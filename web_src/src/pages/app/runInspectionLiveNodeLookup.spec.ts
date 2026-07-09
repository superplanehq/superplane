import { describe, expect, it } from "vitest";
import type { CanvasesCanvasEvent, CanvasesCanvasNodeExecution } from "@/api-client";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { resolveCachedNodeRunId, resolveRunLookupEventForNodeActivity } from "./runInspectionLiveNodeLookup";

function execution(overrides: Partial<CanvasesCanvasNodeExecution>): CanvasesCanvasNodeExecution {
  return overrides as CanvasesCanvasNodeExecution;
}

function event(overrides: Partial<CanvasesCanvasEvent>): CanvasesCanvasEvent {
  return overrides as CanvasesCanvasEvent;
}

describe("resolveRunLookupEventForNodeActivity", () => {
  it("uses the latest execution for action nodes even when a newer event is cached", () => {
    const lookupEvent = resolveRunLookupEventForNodeActivity("action-1", "TYPE_ACTION", {
      executions: [
        execution({ id: "older-execution", createdAt: "2026-07-07T10:00:00Z" }),
        execution({ id: "latest-execution", createdAt: "2026-07-07T10:05:00Z" }),
      ],
      events: [event({ id: "newer-event", createdAt: "2026-07-07T10:10:00Z" })],
    });

    expect(lookupEvent).toMatchObject({
      id: "latest-execution",
      executionId: "latest-execution",
      kind: "execution",
      nodeId: "action-1",
    });
  });

  it("uses the latest event for trigger nodes", () => {
    const lookupEvent = resolveRunLookupEventForNodeActivity("trigger-1", "TYPE_TRIGGER", {
      executions: [execution({ id: "execution-1", createdAt: "2026-07-07T10:10:00Z" })],
      events: [
        event({ id: "older-event", createdAt: "2026-07-07T10:00:00Z" }),
        event({ id: "latest-event", createdAt: "2026-07-07T10:05:00Z" }),
      ],
    });

    expect(lookupEvent).toMatchObject({
      id: "latest-event",
      triggerEventId: "latest-event",
      kind: "trigger",
      nodeId: "trigger-1",
    });
  });

  it("resolves cached runs from the current node activity store", () => {
    useNodeExecutionStore.getState().clear();
    useNodeExecutionStore
      .getState()
      .updateNodeExecution(
        "action-1",
        execution({ id: "latest-execution", nodeId: "action-1", createdAt: "2026-07-07T10:20:00Z" }),
      );

    const runId = resolveCachedNodeRunId("action-1", { id: "action-1", type: "TYPE_ACTION" }, (event) =>
      event.executionId === "latest-execution" ? "run-from-latest-execution" : null,
    );

    expect(runId).toBe("run-from-latest-execution");
  });
});
