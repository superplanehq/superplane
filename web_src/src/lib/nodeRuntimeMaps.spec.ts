import { describe, expect, it } from "bun:test";
import type { CanvasesCanvasNodeExecution } from "@/api-client";
import {
  buildNodeRuntimeMaps,
  EMPTY_NODE_RUNTIME_MAPS,
  nodeRuntimeMapsEqual,
  type NodeRuntimeStoreEntry,
} from "./nodeRuntimeMaps";

function execution(id: string): CanvasesCanvasNodeExecution {
  return { id, nodeId: "node-1" };
}

function entry(overrides: Partial<NodeRuntimeStoreEntry> = {}): NodeRuntimeStoreEntry {
  return {
    executions: [],
    queueItems: [],
    events: [],
    ...overrides,
  };
}

describe("buildNodeRuntimeMaps", () => {
  it("omits empty node collections", () => {
    const data = new Map<string, NodeRuntimeStoreEntry>([["idle", entry()]]);
    expect(buildNodeRuntimeMaps(data)).toEqual(EMPTY_NODE_RUNTIME_MAPS);
  });

  it("keeps execution array identity for unchanged nodes", () => {
    const executions = [execution("execution-1")];
    const data = new Map<string, NodeRuntimeStoreEntry>([["node-1", entry({ executions })]]);
    const first = buildNodeRuntimeMaps(data);
    const second = buildNodeRuntimeMaps(data);
    expect(first.nodeExecutionsMap["node-1"]).toBe(executions);
    expect(nodeRuntimeMapsEqual(first, second)).toBe(true);
  });

  it("detects a replaced node collection", () => {
    const first = buildNodeRuntimeMaps(new Map([["node-1", entry({ executions: [execution("execution-1")] })]]));
    const second = buildNodeRuntimeMaps(new Map([["node-1", entry({ executions: [execution("execution-1")] })]]));
    expect(nodeRuntimeMapsEqual(first, second)).toBe(false);
  });
});
