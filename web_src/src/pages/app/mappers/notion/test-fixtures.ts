import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

export function notionNode(componentName: string, configuration: unknown = {}): NodeInfo {
  return { id: "notion-node", name: "Notion action", componentName, configuration, isCollapsed: false };
}

export function notionExecution(data?: unknown, overrides: Partial<ExecutionInfo> = {}): ExecutionInfo {
  const timestamp = new Date().toISOString();
  return {
    id: "execution-1",
    createdAt: timestamp,
    updatedAt: timestamp,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    configuration: {},
    metadata: {},
    rootEvent: undefined,
    outputs: data === undefined ? undefined : { default: [{ data, timestamp, type: "notion.result" }] },
    ...overrides,
  };
}

export function notionDetails(componentName: string, data?: unknown): ExecutionDetailsContext {
  const node = notionNode(componentName);
  return { node, nodes: [node], execution: notionExecution(data) };
}

export function notionContext(componentName: string, configuration?: unknown): ComponentBaseContext {
  const node = notionNode(componentName, configuration);
  return {
    node,
    nodes: [node],
    componentDefinition: {
      name: componentName,
      label: "Notion action",
      description: "",
      icon: "notion",
      color: "gray",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

export function notionExecutedContext(componentName: string, data: unknown): ComponentBaseContext {
  const context = notionContext(componentName);
  const trigger = { ...notionNode("start"), id: "start", name: "Start" };
  const createdAt = new Date(Date.now() - 125_000).toISOString();
  const execution = notionExecution(data, {
    createdAt,
    rootEvent: { id: "event-1", nodeId: trigger.id, type: "start", createdAt, data: { scenario: "Notion QA" } },
  });
  return { ...context, nodes: [trigger, context.node], lastExecutions: [execution] };
}
