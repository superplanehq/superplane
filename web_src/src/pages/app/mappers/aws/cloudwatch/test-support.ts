import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../../types";

interface CloudWatchFixtureOptions {
  nodeName: string;
  componentName: string;
  label: string;
  outputType: string;
  createdAt: string;
  updatedAt: string;
}

export function createCloudWatchFixtures(options: CloudWatchFixtureOptions) {
  function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
    return {
      id: "node-1",
      name: options.nodeName,
      componentName: options.componentName,
      isCollapsed: false,
      configuration: {},
      metadata: {},
      ...overrides,
    };
  }

  function buildOutput(data: unknown): OutputPayload {
    return { type: options.outputType, timestamp: new Date().toISOString(), data };
  }

  function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
    return {
      id: "exec-1",
      createdAt: options.createdAt,
      updatedAt: options.updatedAt,
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

  function buildDetailsCtx(overrides?: {
    node?: Partial<NodeInfo>;
    execution?: Partial<ExecutionInfo>;
  }): ExecutionDetailsContext {
    const node = buildNode(overrides?.node);
    return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
  }

  const defaultDefinition: ComponentDefinition = {
    name: options.componentName,
    label: options.label,
    description: "",
    icon: "aws.cloudwatch",
    color: "gray",
  };

  function buildPropsContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
    return {
      nodes: [],
      node: buildNode(),
      componentDefinition: defaultDefinition,
      lastExecutions: [],
      currentUser: undefined,
      actions: { invokeNodeExecutionHook: async () => {} },
      ...overrides,
    };
  }

  return { buildNode, buildOutput, buildExecution, buildDetailsCtx, buildPropsContext };
}
