import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "../types";

export function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test VM Node",
    componentName: "gcp.manageVMInstancePower",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

export function buildOutput(data: unknown, type = "gcp.result"): OutputPayload {
  return {
    type,
    timestamp: new Date().toISOString(),
    data,
  };
}

export function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
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

export function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

export function buildComponentCtx(
  nodeOverrides?: Partial<NodeInfo>,
  componentName = "gcp.compute.createVM",
): ComponentBaseContext {
  const node = buildNode({ componentName, ...nodeOverrides });
  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: node.componentName,
      label: "GCP",
      description: "",
      icon: "gcp",
      color: "gray",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}
