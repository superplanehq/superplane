import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../../types";

export function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Workspace",
    componentName: "prometheus.getWorkspace",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

export function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date("2026-06-08T09:00:00Z").toISOString(),
    updatedAt: new Date("2026-06-08T09:01:00Z").toISOString(),
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

export function buildOutput(data: Record<string, unknown>) {
  return { type: "json", timestamp: new Date().toISOString(), data };
}

export function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

export function buildComponentCtx(nodeOverrides?: Partial<NodeInfo>): ComponentBaseContext {
  const node = buildNode(nodeOverrides);
  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: node.componentName,
      label: "Prometheus • Workspace",
      description: "",
      icon: "aws",
      color: "gray",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}
