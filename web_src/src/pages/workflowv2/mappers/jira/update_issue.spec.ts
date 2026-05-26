import { describe, expect, it } from "vitest";

import { eventStateRegistry } from "./index";
import { updateIssueMapper } from "./update_issue";
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
    componentName: "jira.updateIssue",
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
      name: "jira.updateIssue",
      label: "Update Issue",
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

describe("updateIssueMapper.getExecutionDetails", () => {
  it("lists which fields were updated based on configuration", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: {
          project: "TEST",
          issueKey: "TEST-1",
          summary: "new summary",
          priority: "High",
          labels: ["bug"],
        },
      },
      execution: {
        outputs: {
          default: [
            buildOutput({
              key: "TEST-1",
              self: "https://test.atlassian.net/rest/api/3/issue/10001",
              fields: { summary: "new summary", status: { name: "In Progress" } },
            }),
          ],
        },
      },
    });
    const details = updateIssueMapper.getExecutionDetails(ctx);
    expect(details["Key"]).toBe("TEST-1");
    expect(details["Issue URL"]).toBe("https://test.atlassian.net/browse/TEST-1");
    expect(details["Status"]).toBe("In Progress");
    expect(details["Fields Updated"]).toBe("Summary, Priority, Labels");
  });

  it("skips Fields Updated when no editable fields are set", () => {
    const ctx = buildDetailsCtx({
      node: { configuration: { project: "TEST", issueKey: "TEST-1", notifyUsers: true } },
      execution: { outputs: { default: [buildOutput({ key: "TEST-1" })] } },
    });
    expect(updateIssueMapper.getExecutionDetails(ctx)["Fields Updated"]).toBeUndefined();
  });

  it("always includes Executed At", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(updateIssueMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

describe("updateIssueMapper.props", () => {
  it("renders project, issue key, and updates summary", () => {
    const props = updateIssueMapper.props(
      buildComponentContext({
        node: {
          configuration: {
            project: "TEST",
            issueKey: "TEST-5",
            summary: "new",
            assignee: "acct-99",
          },
          metadata: { project: { key: "TEST", name: "Test" } },
        },
      }),
    );
    const labels = (props.metadata || []).map((m) => m.label).filter((l): l is string => typeof l === "string");
    expect(labels).toContain("Test");
    expect(labels).toContain("TEST-5");
    const updatesLabel = labels.find((l) => l.startsWith("Updates:"));
    expect(updatesLabel).toBeDefined();
    expect(updatesLabel).toContain("Summary");
    expect(updatesLabel).toContain("Assignee");
  });

  it("omits the updates badge when only ignored fields are present", () => {
    const props = updateIssueMapper.props(
      buildComponentContext({
        node: {
          configuration: { project: "TEST", issueKey: "TEST-5", notifyUsers: true },
          metadata: { project: { key: "TEST", name: "Test" } },
        },
      }),
    );
    const labels = (props.metadata || []).map((m) => m.label).filter((l): l is string => typeof l === "string");
    expect(labels.some((l) => l.startsWith("Updates:"))).toBe(false);
  });
});

describe("eventStateRegistry.updateIssue", () => {
  it("maps finished success to updated", () => {
    expect(eventStateRegistry.updateIssue.getState(buildExecution())).toBe("updated");
  });
});
