import { describe, expect, it } from "vitest";

import { assignWorkflowToProjectMapper } from "./assign_workflow_to_project";
import { eventStateRegistry } from "./index";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

function node(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Assign scheme",
    componentName: "jira.assignWorkflowToProject",
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
      name: "jira.assignWorkflowToProject",
      label: "Assign Workflow To Project",
      description: "",
      icon: "jira",
      color: "blue",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("assignWorkflowToProjectMapper", () => {
  it("extracts assignment details", () => {
    const details = assignWorkflowToProjectMapper.getExecutionDetails(
      detailsCtx({
        execution: {
          outputs: {
            default: [
              {
                type: "jira.workflowScheme.assigned",
                timestamp: "2026-01-19T12:00:00Z",
                data: {
                  projectId: "10000",
                  workflowSchemeId: "101010",
                  draftCreated: false,
                  taskId: "task-1",
                  taskStatus: "ENQUEUED",
                },
              },
            ],
          },
        },
      }),
    );

    expect(details["Project ID"]).toBe("10000");
    expect(details["Workflow Scheme ID"]).toBe("101010");
    expect(details["Task ID"]).toBe("task-1");
    expect(details["Task Status"]).toBe("ENQUEUED");
  });

  it("renders project and scheme metadata", () => {
    const props = assignWorkflowToProjectMapper.props(
      componentCtx({
        node: {
          configuration: { project: "TEST", workflowScheme: "101010", dryRun: true },
          metadata: {
            project: { key: "TEST", name: "Test Project" },
            workflowScheme: { id: "101010", name: "Support scheme" },
          },
        },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "folder", label: "Test Project" },
      { icon: "workflow", label: "Support scheme" },
      { icon: "search-check", label: "Dry run" },
    ]);
  });

  it("maps finished success to assigned", () => {
    expect(eventStateRegistry.assignWorkflowToProject.getState(execution())).toBe("assigned");
  });
});
