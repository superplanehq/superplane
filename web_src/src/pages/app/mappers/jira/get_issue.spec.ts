import { describe, expect, it } from "vitest";

import { eventStateRegistry } from "./index";
import { getIssueMapper } from "./get_issue";
import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "jira.getIssue",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "jira.issue",
    timestamp: new Date().toISOString(),
    data,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date("2026-01-15T12:34:56Z").toISOString(),
    updatedAt: new Date("2026-01-15T12:34:56Z").toISOString(),
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

function buildComponentContext(overrides?: {
  node?: Partial<NodeInfo>;
  lastExecutions?: ExecutionInfo[];
  componentDefinition?: Partial<ComponentDefinition>;
}): ComponentBaseContext {
  const node = buildNode(overrides?.node);
  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: "jira.getIssue",
      label: "Get Issue",
      description: "",
      icon: "jira",
      color: "blue",
      ...overrides?.componentDefinition,
    },
    lastExecutions: overrides?.lastExecutions ?? [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("getIssueMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getIssueMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("always includes Executed At", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(getIssueMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });

  it("extracts the 5 most relevant fields and caps at 6 rows", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              key: "TEST-7",
              self: "https://test.atlassian.net/rest/api/3/issue/10007",
              fields: {
                summary: "Login error",
                status: { name: "To Do" },
                assignee: { displayName: "Bob" },
                priority: { name: "High" },
                project: { key: "TEST", name: "Test" },
                labels: ["a", "b"],
              },
            }),
          ],
        },
      },
    });
    const details = getIssueMapper.getExecutionDetails(ctx);
    expect(details["Key"]).toBe("TEST-7");
    expect(details["Issue URL"]).toBe("https://test.atlassian.net/browse/TEST-7");
    expect(details["Summary"]).toBe("Login error");
    expect(details["Status"]).toBe("To Do");
    expect(details["Assignee"]).toBe("Bob");
    expect(details["Priority"]).toBe("High");
    expect(Object.keys(details)).toHaveLength(7);
    expect(details["Project"]).toBeUndefined();
    expect(details["Labels"]).toBeUndefined();
  });

  it("omits missing fields", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ key: "TEST-1" })] } },
    });
    const details = getIssueMapper.getExecutionDetails(ctx);
    expect(details["Key"]).toBe("TEST-1");
    expect(details["Summary"]).toBeUndefined();
    expect(details["Status"]).toBeUndefined();
  });
});

describe("getIssueMapper.props", () => {
  it("renders project + static issue key in metadata", () => {
    const props = getIssueMapper.props(
      buildComponentContext({
        node: {
          configuration: { project: "TEST", issueKey: "TEST-1" },
          metadata: { project: { key: "TEST", name: "Test" } },
        },
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "folder", label: "Test" },
      { icon: "hash", label: "TEST-1" },
    ]);
  });

  it("skips issueKey badge when it is an expression", () => {
    const props = getIssueMapper.props(
      buildComponentContext({
        node: {
          configuration: { project: "TEST", issueKey: "{{ trigger.key }}" },
          metadata: { project: { key: "TEST", name: "Test" } },
        },
      }),
    );
    const labels = (props.metadata || []).map((m) => m.label);
    expect(labels).not.toContain("{{ trigger.key }}");
  });

  it("falls back to configuration project when metadata is empty", () => {
    const props = getIssueMapper.props(
      buildComponentContext({
        node: { configuration: { project: "DEMO", issueKey: "DEMO-2" }, metadata: {} },
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "folder", label: "DEMO" });
  });
});

describe("eventStateRegistry.getIssue", () => {
  it("maps finished success to retrieved", () => {
    expect(eventStateRegistry.getIssue.getState(buildExecution())).toBe("retrieved");
  });
});
