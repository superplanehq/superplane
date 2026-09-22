import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "bun:test";
import type { CanvasesCanvasNodeExecution } from "@/api-client";
import { EMPTY_NODE_RUNTIME_MAPS } from "@/lib/nodeRuntimeMaps";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { useNodeRuntimeMaps } from "./useNodeRuntimeMaps";

function execution(id: string): CanvasesCanvasNodeExecution {
  return { id, nodeId: "node-1" };
}

describe("useNodeRuntimeMaps", () => {
  beforeEach(() => {
    act(() => {
      useNodeExecutionStore.getState().clear();
    });
  });

  it("keeps a stable empty snapshot when runtime maps are disabled", () => {
    const { result } = renderHook(() => useNodeRuntimeMaps(false));
    const first = result.current;

    act(() => {
      useNodeExecutionStore.getState().updateNodeExecution("node-1", execution("execution-1"));
    });

    expect(result.current).toBe(EMPTY_NODE_RUNTIME_MAPS);
    expect(result.current).toBe(first);
  });

  it("rebuilds maps only when a node collection changes", () => {
    const { result } = renderHook(() => useNodeRuntimeMaps(true));

    act(() => {
      useNodeExecutionStore.getState().updateNodeExecution("node-1", execution("execution-1"));
    });

    const afterFirstUpdate = result.current;
    expect(afterFirstUpdate.nodeExecutionsMap["node-1"]?.at(-1)?.id).toBe("execution-1");

    act(() => {
      useNodeExecutionStore.getState().updateNodeExecution("node-2", execution("execution-2"));
    });

    expect(result.current).not.toBe(afterFirstUpdate);
    expect(result.current.nodeExecutionsMap["node-1"]).toBe(afterFirstUpdate.nodeExecutionsMap["node-1"]);
    expect(result.current.nodeExecutionsMap["node-2"]?.at(-1)?.id).toBe("execution-2");
  });
});
