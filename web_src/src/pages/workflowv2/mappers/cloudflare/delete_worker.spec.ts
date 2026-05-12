import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { deleteWorkerMapper } from "./delete_worker";
import { eventStateRegistry } from "./index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "cloudflare.deleteWorker",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    ...overrides,
  };
}

function buildDetailsCtx(overrides?: { execution?: Partial<ExecutionInfo> }): ExecutionDetailsContext {
  const node = buildNode();
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

describe("deleteWorkerMapper.getExecutionDetails", () => {
  it("includes script name and deleted flag", () => {
    const data = { workerScript: "gone", deleted: true };
    const outputs = { default: [{ type: "cloudflare.worker.deleted", timestamp: new Date().toISOString(), data }] };
    const details = deleteWorkerMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs } }));
    expect(details["Script"]).toBe("gone");
    expect(details["Deleted"]).toBe("true");
  });
});

describe("eventStateRegistry.deleteWorker", () => {
  it("maps finished success to deleted", () => {
    expect(eventStateRegistry.deleteWorker.getState(buildExecution())).toBe("deleted");
  });
});
