import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

export function buildImageNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "OCI Image",
    componentName: "oci.createImage",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

export function buildImageExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date("2026-01-01T00:00:00Z").toISOString(),
    updatedAt: new Date("2026-01-01T00:01:00Z").toISOString(),
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

export function buildImageOutput(data: Record<string, unknown>) {
  return { type: "json", timestamp: new Date("2026-01-01T00:00:00Z").toISOString(), data };
}

export function buildImageDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildImageNode(overrides?.node);
  return { nodes: [node], node, execution: buildImageExecution(overrides?.execution) };
}

export function buildImageComponentCtx(nodeOverrides?: Partial<NodeInfo>): ComponentBaseContext {
  const node = buildImageNode(nodeOverrides);
  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: node.componentName,
      label: "OCI Image",
      description: "",
      icon: "oci",
      color: "red",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}
