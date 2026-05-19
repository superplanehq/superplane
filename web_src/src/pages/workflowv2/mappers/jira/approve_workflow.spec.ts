import { describe, expect, it } from "vitest";

import { approveWorkflowMapper } from "./approve_workflow";
import { eventStateRegistry } from "./index";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

function node(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Approve workflow",
    componentName: "jira.approveWorkflow",
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
      name: "jira.approveWorkflow",
      label: "Approve Workflow",
      description: "",
      icon: "jira",
      color: "green",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("approveWorkflowMapper", () => {
  it("extracts approval details", () => {
    const details = approveWorkflowMapper.getExecutionDetails(
      detailsCtx({
        node: { configuration: { issueKey: "ITSM-1", decision: "approve" } },
        execution: {
          outputs: {
            default: [
              {
                type: "jira.approval",
                timestamp: "2026-01-19T12:00:00Z",
                data: {
                  id: "2",
                  name: "Manager",
                  finalDecision: "approved",
                  approvers: [{ approver: { displayName: "Alice" } }],
                },
              },
            ],
          },
        },
      }),
    );

    expect(details["Approval ID"]).toBe("2");
    expect(details.Name).toBe("Manager");
    expect(details.Decision).toBe("approved");
    expect(details.Approvers).toBe("Alice");
    expect(details["Issue Key"]).toBe("ITSM-1");
  });

  it("renders issue, decision, and approval id metadata", () => {
    const props = approveWorkflowMapper.props(
      componentCtx({
        node: {
          configuration: { issueKey: "ITSM-1", decision: "decline", approvalSelector: "byId", approvalId: "2" },
        },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "hash", label: "ITSM-1" },
      { icon: "circle-x", label: "decline" },
      { icon: "badge-check", label: "2" },
    ]);
  });

  it("maps finished success to decided", () => {
    expect(eventStateRegistry.approveWorkflow.getState(execution())).toBe("decided");
  });
});
