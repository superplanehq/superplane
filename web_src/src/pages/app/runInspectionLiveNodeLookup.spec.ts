import { describe, expect, it } from "vitest";
import type { CanvasesCanvasEvent, CanvasesCanvasNodeExecution } from "@/api-client";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import {
  resolveCachedNodeRunId,
  resolveLiveCanvasNodeClickSyncAction,
  resolveRunLookupEventForNodeActivity,
  shouldDeferRunInspectionForLiveNodeClick,
} from "./runInspectionLiveNodeLookup";

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

  it("resolves cached runs directly from execution activity run id", () => {
    useNodeExecutionStore.getState().clear();
    useNodeExecutionStore.getState().updateNodeExecution(
      "action-1",
      execution({
        id: "latest-execution",
        nodeId: "action-1",
        runId: "run-from-execution",
        createdAt: "2026-07-07T10:20:00Z",
      }),
    );

    const runId = resolveCachedNodeRunId("action-1", { id: "action-1", type: "TYPE_ACTION" }, () => null);

    expect(runId).toBe("run-from-execution");
  });

  it("resolves cached runs directly from trigger event run id", () => {
    useNodeExecutionStore.getState().clear();
    useNodeExecutionStore.getState().updateNodeEvent(
      "trigger-1",
      event({
        id: "latest-event",
        nodeId: "trigger-1",
        runId: "run-from-trigger",
        createdAt: "2026-07-07T10:20:00Z",
      }),
    );

    const runId = resolveCachedNodeRunId("trigger-1", { id: "trigger-1", type: "TYPE_TRIGGER" }, () => null);

    expect(runId).toBe("run-from-trigger");
  });
});

describe("resolveLiveCanvasNodeClickSyncAction", () => {
  it("looks up a run from the server when the node has no cached activity", () => {
    useNodeExecutionStore.getState().clear();

    expect(
      resolveLiveCanvasNodeClickSyncAction("action-1", { id: "action-1", type: "TYPE_ACTION" }, () => null),
    ).toEqual({ kind: "lookupRun" });
  });

  it("inspects the cached run when one is already resolved", () => {
    useNodeExecutionStore.getState().clear();
    useNodeExecutionStore.getState().updateNodeExecution(
      "action-1",
      execution({
        id: "latest-execution",
        nodeId: "action-1",
        runId: "run-from-execution",
        createdAt: "2026-07-07T10:20:00Z",
      }),
    );

    expect(
      resolveLiveCanvasNodeClickSyncAction("action-1", { id: "action-1", type: "TYPE_ACTION" }, () => null),
    ).toEqual({ kind: "inspectRun", runId: "run-from-execution" });
  });

  it("looks up a run when activity exists but no cached run id is available", () => {
    useNodeExecutionStore.getState().clear();
    useNodeExecutionStore.getState().updateNodeExecution(
      "action-1",
      execution({
        id: "latest-execution",
        nodeId: "action-1",
        createdAt: "2026-07-07T10:20:00Z",
      }),
    );

    expect(
      resolveLiveCanvasNodeClickSyncAction("action-1", { id: "action-1", type: "TYPE_ACTION" }, () => null),
    ).toEqual({ kind: "lookupRun" });
  });
});

describe("shouldDeferRunInspectionForLiveNodeClick", () => {
  it("defers run inspection for approval nodes waiting on input", () => {
    useNodeExecutionStore.getState().clear();
    useNodeExecutionStore.getState().updateNodeExecution(
      "approval-1",
      execution({
        id: "approval-execution",
        nodeId: "approval-1",
        state: "STATE_STARTED",
        createdAt: "2026-07-07T10:20:00Z",
      }),
    );

    expect(
      shouldDeferRunInspectionForLiveNodeClick(
        { id: "approval-1", type: "TYPE_ACTION", component: "approval" },
        useNodeExecutionStore.getState().getNodeData("approval-1"),
      ),
    ).toBe(true);
  });

  it("does not defer run inspection for finished approval executions", () => {
    useNodeExecutionStore.getState().clear();
    useNodeExecutionStore.getState().updateNodeExecution(
      "approval-1",
      execution({
        id: "approval-execution",
        nodeId: "approval-1",
        state: "STATE_FINISHED",
        createdAt: "2026-07-07T10:20:00Z",
      }),
    );

    expect(
      shouldDeferRunInspectionForLiveNodeClick(
        { id: "approval-1", type: "TYPE_ACTION", component: "approval" },
        useNodeExecutionStore.getState().getNodeData("approval-1"),
      ),
    ).toBe(false);
  });

  it("does not defer run inspection for wait nodes with active executions", () => {
    useNodeExecutionStore.getState().clear();
    useNodeExecutionStore.getState().updateNodeExecution(
      "wait-1",
      execution({
        id: "wait-execution",
        nodeId: "wait-1",
        state: "STATE_STARTED",
        createdAt: "2026-07-07T10:20:00Z",
      }),
    );

    expect(
      shouldDeferRunInspectionForLiveNodeClick(
        { id: "wait-1", type: "TYPE_ACTION", component: "wait" },
        useNodeExecutionStore.getState().getNodeData("wait-1"),
      ),
    ).toBe(false);
  });

  it("does not defer run inspection for unrelated components", () => {
    useNodeExecutionStore.getState().clear();
    useNodeExecutionStore.getState().updateNodeExecution(
      "noop-1",
      execution({
        id: "noop-execution",
        nodeId: "noop-1",
        state: "STATE_STARTED",
        createdAt: "2026-07-07T10:20:00Z",
      }),
    );

    expect(
      shouldDeferRunInspectionForLiveNodeClick(
        { id: "noop-1", type: "TYPE_ACTION", component: "noop" },
        useNodeExecutionStore.getState().getNodeData("noop-1"),
      ),
    ).toBe(false);
  });
});
