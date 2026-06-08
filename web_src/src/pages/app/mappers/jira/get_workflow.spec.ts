import { describe, expect, it } from "vitest";

import { getWorkflowMapper } from "./get_workflow";
import { eventStateRegistry } from "./index";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

function node(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Get workflow",
    componentName: "jira.getWorkflow",
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
      name: "jira.getWorkflow",
      label: "Get Workflow",
      description: "",
      icon: "jira",
      color: "blue",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("getWorkflowMapper", () => {
  it("extracts workflow + current status + transitions from the payload", () => {
    const details = getWorkflowMapper.getExecutionDetails(
      detailsCtx({
        execution: {
          outputs: {
            default: [
              {
                type: "jira.workflow",
                timestamp: "2026-01-19T12:00:00Z",
                data: {
                  issueKey: "TEST-1",
                  issueType: "Task",
                  workflowName: "Software Simplified",
                  workflowSchemeName: "Default scheme",
                  currentStatus: "In Progress",
                  availableTransitions: [
                    { id: "21", name: "Stop progress", toStatus: "To Do" },
                    { id: "31", name: "Resolve", toStatus: "Done" },
                  ],
                },
              },
            ],
          },
        },
      }),
    );

    expect(details.Issue).toBe("TEST-1");
    expect(details["Issue Type"]).toBe("Task");
    expect(details.Workflow).toBe("Software Simplified");
    expect(details["Current Status"]).toBe("In Progress");
    expect(details["Available Transitions"]).toBe("To Do, Done");
  });

  it("falls back gracefully when no execution data is present", () => {
    const details = getWorkflowMapper.getExecutionDetails(detailsCtx());
    expect(details["Executed At"]).toBeDefined();
    expect(details.Issue).toBeUndefined();
  });

  it("renders project + issue key in metadata", () => {
    const props = getWorkflowMapper.props(
      componentCtx({
        node: {
          configuration: { project: "TEST", issueKey: "TEST-1" },
          metadata: { project: { key: "TEST", name: "Test Project" } },
        },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "folder", label: "Test Project" },
      { icon: "hash", label: "TEST-1" },
    ]);
  });

  it("maps finished success to retrieved", () => {
    expect(eventStateRegistry.getWorkflow.getState(execution())).toBe("retrieved");
  });
});
