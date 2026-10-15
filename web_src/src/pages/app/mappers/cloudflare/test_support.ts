import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";

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

export function buildOutput(type: string, data: unknown): OutputPayload {
  return { type, timestamp: new Date().toISOString(), data };
}

export function createNodeFixtures(componentName: string, nodeName = "Test Node") {
  function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
    return {
      id: "node-1",
      name: nodeName,
      componentName,
      isCollapsed: false,
      configuration: {},
      metadata: {},
      ...overrides,
    };
  }

  function buildDetailsCtx(overrides?: {
    node?: Partial<NodeInfo>;
    execution?: Partial<ExecutionInfo>;
  }): ExecutionDetailsContext {
    const node = buildNode(overrides?.node);
    return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
  }

  return { buildNode, buildDetailsCtx };
}

export function createComponentFixtures(componentDefinition: ComponentDefinition, nodeName = "Test Node") {
  const fixtures = createNodeFixtures(componentDefinition.name, nodeName);

  function buildPropsContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
    return {
      nodes: [],
      node: fixtures.buildNode(),
      componentDefinition,
      lastExecutions: [],
      currentUser: undefined,
      actions: { invokeNodeExecutionHook: async () => {} },
      ...overrides,
    };
  }

  return { ...fixtures, buildPropsContext };
}
