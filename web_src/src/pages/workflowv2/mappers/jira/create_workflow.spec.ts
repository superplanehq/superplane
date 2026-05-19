import { describe, expect, it } from "vitest";

import { createWorkflowMapper } from "./create_workflow";
import { eventStateRegistry } from "./index";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo, SubtitleContext } from "../types";

function node(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Create workflow",
    componentName: "jira.createWorkflow",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function execution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: "2026-01-19T12:00:00Z",
    updatedAt: "2026-01-19T12:00:00Z",
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

function detailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const n = node(overrides?.node);
  return { nodes: [n], node: n, execution: execution(overrides?.execution) };
}

function componentCtx(overrides?: { node?: Partial<NodeInfo> }): ComponentBaseContext {
  const n = node(overrides?.node);
  return {
    nodes: [n],
    node: n,
    componentDefinition: {
      name: "jira.createWorkflow",
      label: "Create Workflow",
      description: "",
      icon: "jira",
      color: "blue",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("createWorkflowMapper", () => {
  it("extracts workflow output details", () => {
    const details = createWorkflowMapper.getExecutionDetails(
      detailsCtx({
        node: { configuration: { scope: "GLOBAL", statuses: [{ name: "To Do" }], transitions: [{ name: "Done" }] } },
        execution: {
          outputs: {
            default: [
              {
                type: "jira.workflow.created",
                timestamp: "2026-01-19T12:00:00Z",
                data: { id: "wf-1", name: "Support", version: { versionNumber: 1 } },
              },
            ],
          },
        },
      }),
    );

    expect(details["Workflow ID"]).toBe("wf-1");
    expect(details.Name).toBe("Support");
    expect(details.Version).toBe("1");
    expect(details.Statuses).toBe("1");
    expect(details.Transitions).toBe("1");
  });

  it("renders workflow metadata", () => {
    const props = createWorkflowMapper.props(
      componentCtx({
        node: {
          configuration: { name: "Support", scope: "PROJECT", project: "TEST", statuses: [{ name: "To Do" }] },
          metadata: { project: { key: "TEST", name: "Test Project" }, workflowName: "Support" },
        },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "workflow", label: "Support" },
      { icon: "globe", label: "Project scoped" },
      { icon: "folder", label: "Test Project" },
      { icon: "list", label: "1 statuses" },
    ]);
  });

  it("uses workflow name as subtitle", () => {
    const result = createWorkflowMapper.subtitle({
      node: node(),
      execution: execution({
        outputs: {
          default: [{ type: "jira.workflow.created", timestamp: "2026-01-19T12:00:00Z", data: { name: "Support" } }],
        },
      }),
    } as SubtitleContext);

    expect(result).toBe("Support");
  });

  it("maps finished success to created", () => {
    expect(eventStateRegistry.createWorkflow.getState(execution())).toBe("created");
  });
});
