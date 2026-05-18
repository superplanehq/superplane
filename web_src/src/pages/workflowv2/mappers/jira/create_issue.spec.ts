import { describe, expect, it } from "vitest";

import { createIssueMapper } from "./create_issue";
import { eventStateRegistry } from "./index";
import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "jira.createIssue",
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

function buildSubtitleCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): SubtitleContext {
  return {
    node: buildNode(overrides?.node),
    execution: buildExecution(overrides?.execution),
  };
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
      name: "jira.createIssue",
      label: "Create Issue",
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

describe("createIssueMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createIssueMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => createIssueMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("always includes Executed At", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    const details = createIssueMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Executed At"]).not.toBe("-");
  });

  it("shows dash for Executed At when createdAt is missing", () => {
    const ctx = buildDetailsCtx({ execution: { createdAt: undefined, outputs: undefined } });
    expect(createIssueMapper.getExecutionDetails(ctx)["Executed At"]).toBe("-");
  });

  it("extracts the 5 most relevant fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              key: "TEST-42",
              self: "https://test.atlassian.net/rest/api/3/issue/10042",
              fields: {
                summary: "New bug",
                status: { name: "In Progress" },
                issuetype: { name: "Bug" },
                assignee: { displayName: "Alice" },
              },
            }),
          ],
        },
      },
    });
    const details = createIssueMapper.getExecutionDetails(ctx);
    expect(details["Key"]).toBe("TEST-42");
    expect(details["Issue URL"]).toBe("https://test.atlassian.net/browse/TEST-42");
    expect(details["Summary"]).toBe("New bug");
    expect(details["Status"]).toBe("In Progress");
    expect(details["Issue Type"]).toBe("Bug");
    expect(details["Assignee"]).toBe("Alice");
    // 1 timestamp + 6 fields = 7
    expect(Object.keys(details)).toHaveLength(7);
  });

  it("omits missing fields rather than padding with dashes", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ key: "TEST-1", fields: { summary: "Only summary" } })],
        },
      },
    });
    const details = createIssueMapper.getExecutionDetails(ctx);
    expect(details["Key"]).toBe("TEST-1");
    expect(details["Summary"]).toBe("Only summary");
    expect(details["Status"]).toBeUndefined();
    expect(details["Issue Type"]).toBeUndefined();
    expect(details["Assignee"]).toBeUndefined();
  });
});

describe("createIssueMapper.props", () => {
  it("renders project, issue type, and status from node metadata", () => {
    const props = createIssueMapper.props(
      buildComponentContext({
        node: {
          configuration: { project: "TEST", issueType: "Bug", status: "In Progress" },
          metadata: {
            project: { key: "TEST", name: "Test Project" },
            issueType: "Bug",
            status: "In Progress",
          },
        },
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "folder", label: "Test Project" },
      { icon: "tag", label: "Bug" },
      { icon: "flag", label: "In Progress" },
    ]);
  });

  it("falls back to configuration when metadata is empty", () => {
    const props = createIssueMapper.props(
      buildComponentContext({
        node: {
          configuration: { project: "DEMO", issueType: "Story" },
          metadata: {},
        },
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "folder", label: "DEMO" },
      { icon: "tag", label: "Story" },
    ]);
  });

  it("omits status badge when neither metadata nor config has status", () => {
    const props = createIssueMapper.props(
      buildComponentContext({
        node: { configuration: { project: "TEST", issueType: "Task" } },
      }),
    );
    const labels = (props.metadata || []).map((m) => m.label);
    expect(labels).not.toContain("In Progress");
  });

  it("does not throw when node metadata and configuration are undefined", () => {
    expect(() =>
      createIssueMapper.props(buildComponentContext({ node: { configuration: undefined, metadata: undefined } })),
    ).not.toThrow();
  });
});

describe("createIssueMapper.subtitle", () => {
  it("returns issue label when payload has key and summary", () => {
    const ctx = buildSubtitleCtx({
      execution: {
        outputs: {
          default: [buildOutput({ key: "TEST-1", fields: { summary: "Hi" } })],
        },
      },
    });
    expect(createIssueMapper.subtitle(ctx)).toBe("TEST-1 · Hi");
  });

  it("falls back to a time-ago element when no payload", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: undefined } });
    const result = createIssueMapper.subtitle(ctx);
    expect(result).not.toBe("");
  });

  it("returns empty string when no payload and no createdAt", () => {
    const ctx = buildSubtitleCtx({ execution: { createdAt: undefined, outputs: undefined } });
    expect(createIssueMapper.subtitle(ctx)).toBe("");
  });
});

describe("eventStateRegistry.createIssue", () => {
  it("maps finished success to created", () => {
    expect(eventStateRegistry.createIssue.getState(buildExecution())).toBe("created");
  });

  it("returns running when execution is in progress", () => {
    const execution = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.createIssue.getState(execution)).toBe("running");
  });
});
