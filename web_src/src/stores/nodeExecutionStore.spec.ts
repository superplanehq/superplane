import { describe, it, expect, beforeEach } from "vitest";
import type { CanvasesCanvasNodeExecution } from "@/api-client";
import { useNodeExecutionStore } from "./nodeExecutionStore";

const nodeId = "node-1";

function execution(overrides: Partial<CanvasesCanvasNodeExecution>): CanvasesCanvasNodeExecution {
  return {
    id: "execution-1",
    nodeId,
    ...overrides,
  };
}

describe("nodeExecutionStore.updateNodeExecution", () => {
  beforeEach(() => {
    useNodeExecutionStore.getState().clear();
  });

  it("keeps the finished state when a stale started event arrives out of order", () => {
    const store = useNodeExecutionStore.getState();

    store.updateNodeExecution(
      nodeId,
      execution({ state: "STATE_FINISHED", result: "RESULT_PASSED", updatedAt: "2026-06-01T12:00:01.000Z" }),
    );

    // Late "started" event for the same execution (older updatedAt) must not
    // downgrade the node back to a running state.
    store.updateNodeExecution(nodeId, execution({ state: "STATE_STARTED", updatedAt: "2026-06-01T12:00:00.000Z" }));

    const data = useNodeExecutionStore.getState().getNodeData(nodeId);
    expect(data.executions).toHaveLength(1);
    expect(data.executions[0].state).toBe("STATE_FINISHED");
  });

  it("applies a newer state update", () => {
    const store = useNodeExecutionStore.getState();

    store.updateNodeExecution(nodeId, execution({ state: "STATE_STARTED", updatedAt: "2026-06-01T12:00:00.000Z" }));
    store.updateNodeExecution(
      nodeId,
      execution({ state: "STATE_FINISHED", result: "RESULT_PASSED", updatedAt: "2026-06-01T12:00:01.000Z" }),
    );

    const data = useNodeExecutionStore.getState().getNodeData(nodeId);
    expect(data.executions).toHaveLength(1);
    expect(data.executions[0].state).toBe("STATE_FINISHED");
  });

  it("prepends a different execution rather than replacing", () => {
    const store = useNodeExecutionStore.getState();

    store.updateNodeExecution(
      nodeId,
      execution({ id: "execution-1", state: "STATE_FINISHED", updatedAt: "2026-06-01T12:00:00.000Z" }),
    );
    store.updateNodeExecution(
      nodeId,
      execution({ id: "execution-2", state: "STATE_STARTED", updatedAt: "2026-06-01T12:01:00.000Z" }),
    );

    const data = useNodeExecutionStore.getState().getNodeData(nodeId);
    expect(data.executions.map((e) => e.id)).toEqual(["execution-2", "execution-1"]);
  });
});
