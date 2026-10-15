import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../../types";

interface EC2FixtureOptions {
  nodeName: string;
  componentName: string;
  label: string;
  icon: string;
  updatedAt?: string;
}

export function createEC2Fixtures(options: EC2FixtureOptions) {
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

  function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
    return {
      id: "exec-1",
      createdAt: new Date("2026-05-21T12:00:00Z").toISOString(),
      updatedAt: new Date(options.updatedAt ?? "2026-05-21T12:01:00Z").toISOString(),
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

  function buildOutput(data: Record<string, unknown>) {
    return { type: "json", timestamp: new Date().toISOString(), data };
  }

  function buildDetailsCtx(overrides?: {
    node?: Partial<NodeInfo>;
    execution?: Partial<ExecutionInfo>;
  }): ExecutionDetailsContext {
    const node = buildNode(overrides?.node);
    return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
  }

  function buildComponentCtx(nodeOverrides?: Partial<NodeInfo>): ComponentBaseContext {
    const node = buildNode(nodeOverrides);
    return {
      nodes: [node],
      node,
      componentDefinition: {
        name: options.componentName,
        label: options.label,
        description: "",
        icon: options.icon,
        color: "gray",
      },
      lastExecutions: [],
      currentUser: undefined,
      actions: { invokeNodeExecutionHook: async () => {} },
    };
  }

  return { buildNode, buildExecution, buildOutput, buildDetailsCtx, buildComponentCtx };
}
