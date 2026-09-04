import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";

export function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "opencodego.chatCompletion",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

export function buildContext(node: NodeInfo, definition?: Partial<ComponentDefinition>): ComponentBaseContext {
  return {
    nodes: [],
    node,
    componentDefinition: {
      name: "opencodego.chatCompletion",
      label: "Chat Completion",
      description: "",
      icon: "sparkles",
      color: "gray",
      ...definition,
    },
    lastExecutions: [],
    currentUser: { id: "user-1", name: "Test User", email: "test@example.com", roles: [], groups: [] },
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

export function buildOutput(type: string, data: unknown): OutputPayload {
  return {
    type,
    timestamp: "2026-08-18T12:00:00.000Z",
    data,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: "2026-08-18T12:00:00.000Z",
    updatedAt: "2026-08-18T12:00:00.000Z",
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

export function buildDetailsCtx(payload?: OutputPayload): ExecutionDetailsContext {
  const node = buildNode();
  return {
    nodes: [node],
    node,
    execution: buildExecution(payload ? { outputs: { default: [payload] } } : undefined),
  };
}
