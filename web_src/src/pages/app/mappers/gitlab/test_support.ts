import type { ComponentDefinition, EventInfo, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

export function buildGitLabDetailsContext(
  nodeOverrides: Pick<NodeInfo, "name" | "componentName"> & Partial<NodeInfo>,
  execution: Partial<ExecutionInfo>,
): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    isCollapsed: false,
    metadata: {
      project: {
        id: 123,
        name: "felixgateru/hello-world",
        url: "https://gitlab.com/felixgateru/hello-world",
      },
    },
    ...nodeOverrides,
  };

  return {
    nodes: [node],
    node,
    execution: {
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
      ...execution,
    },
  };
}

export function createGitLabTriggerBuilders(defaults: {
  name: string;
  componentName: string;
  definitionName: string;
  label: string;
  eventType: string;
}) {
  return {
    buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
      return {
        id: "node-1",
        name: defaults.name,
        componentName: defaults.componentName,
        isCollapsed: false,
        configuration: {},
        metadata: {},
        ...overrides,
      };
    },
    buildDefinition(overrides?: Partial<ComponentDefinition>): ComponentDefinition {
      return {
        name: defaults.definitionName,
        label: defaults.label,
        description: "",
        icon: "gitlab",
        color: "orange",
        ...overrides,
      };
    },
    buildEvent(overrides?: Partial<NonNullable<EventInfo>>): EventInfo {
      return {
        id: "evt-1",
        createdAt: new Date().toISOString(),
        data: {},
        nodeId: "node-1",
        type: defaults.eventType,
        ...overrides,
      };
    },
  };
}
