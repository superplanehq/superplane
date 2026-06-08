import { describe, expect, it } from "vitest";

import { transitionIssueMapper } from "./transition_issue";
import { eventStateRegistry } from "./index";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

function node(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Transition issue",
    componentName: "jira.transitionIssue",
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
      name: "jira.transitionIssue",
      label: "Transition Issue",
      description: "",
      icon: "jira",
      color: "blue",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("transitionIssueMapper", () => {
  it("extracts transitioned issue details", () => {
    const details = transitionIssueMapper.getExecutionDetails(
      detailsCtx({
        node: { configuration: { targetStatus: "Done", resolution: "Done" } },
        execution: {
          outputs: {
            default: [
              {
                type: "jira.issue",
                timestamp: "2026-01-19T12:00:00Z",
                data: {
                  key: "TEST-1",
                  self: "https://test.atlassian.net/rest/api/3/issue/10001",
                  fields: { summary: "Ship", status: { name: "Done" } },
                },
              },
            ],
          },
        },
      }),
    );

    expect(details.Key).toBe("TEST-1");
    expect(details["Issue URL"]).toBe("https://test.atlassian.net/browse/TEST-1");
    expect(details.Status).toBe("Done");
    expect(details["Target Status"]).toBe("Done");
    expect(details.Resolution).toBe("Done");
  });

  it("renders project, key, status, and resolution metadata", () => {
    const props = transitionIssueMapper.props(
      componentCtx({
        node: {
          configuration: { project: "TEST", issueKey: "TEST-1", targetStatus: "Done", resolution: "Done" },
          metadata: { project: { key: "TEST", name: "Test Project" }, status: "Done" },
        },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "folder", label: "Test Project" },
      { icon: "hash", label: "TEST-1" },
      { icon: "flag", label: "Done" },
      { icon: "circle-check", label: "Done" },
    ]);
  });

  it("maps finished success to transitioned", () => {
    expect(eventStateRegistry.transitionIssue.getState(execution())).toBe("transitioned");
  });
});
