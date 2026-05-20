import { describe, expect, it } from "vitest";

import { deleteIssueMapper } from "./delete_issue";
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
    componentName: "jira.deleteIssue",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "jira.issueDeleted",
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
      name: "jira.deleteIssue",
      label: "Delete Issue",
      description: "",
      icon: "jira",
      color: "red",
      ...overrides?.componentDefinition,
    },
    lastExecutions: overrides?.lastExecutions ?? [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("deleteIssueMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => deleteIssueMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("shows Status=Deleted when payload reports deleted true", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ key: "TEST-9", id: "10009", deleted: true })] } },
    });
    const details = deleteIssueMapper.getExecutionDetails(ctx);
    expect(details["Key"]).toBe("TEST-9");
    expect(details["ID"]).toBe("10009");
    expect(details["Status"]).toBe("Deleted");
  });

  it("omits Status when deleted is false or absent", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ key: "TEST-10", id: "10010", deleted: false })] } },
    });
    expect(deleteIssueMapper.getExecutionDetails(ctx)["Status"]).toBeUndefined();
  });

  it("always includes Executed At", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(deleteIssueMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

describe("deleteIssueMapper.props", () => {
  it("shows the subtask badge when deleteSubtasks is true", () => {
    const props = deleteIssueMapper.props(
      buildComponentContext({
        node: {
          configuration: { project: "TEST", issueKey: "TEST-1", deleteSubtasks: true },
          metadata: { project: { key: "TEST", name: "Test" } },
        },
      }),
    );
    const labels = (props.metadata || []).map((m) => m.label);
    expect(labels).toContain("Also subtasks");
  });

  it("omits the subtask badge when deleteSubtasks is false", () => {
    const props = deleteIssueMapper.props(
      buildComponentContext({
        node: {
          configuration: { project: "TEST", issueKey: "TEST-1", deleteSubtasks: false },
        },
      }),
    );
    const labels = (props.metadata || []).map((m) => m.label);
    expect(labels).not.toContain("Also subtasks");
  });

  it("falls back to configuration project when metadata is empty", () => {
    const props = deleteIssueMapper.props(
      buildComponentContext({
        node: { configuration: { project: "DEMO", issueKey: "DEMO-2" }, metadata: {} },
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "folder", label: "DEMO" });
  });
});

describe("deleteIssueMapper.subtitle", () => {
  it("formats deleted key into a sentence", () => {
    const ctx = buildSubtitleCtx({
      execution: { outputs: { default: [buildOutput({ key: "TEST-1", deleted: true })] } },
    });
    expect(deleteIssueMapper.subtitle(ctx)).toBe("TEST-1 deleted");
  });

  it("falls back to time-ago when no key in payload", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    expect(deleteIssueMapper.subtitle(ctx)).not.toBe("");
  });

  it("returns empty string when no createdAt and no payload key", () => {
    const ctx = buildSubtitleCtx({ execution: { createdAt: undefined, outputs: undefined } });
    expect(deleteIssueMapper.subtitle(ctx)).toBe("");
  });
});

describe("eventStateRegistry.deleteIssue", () => {
  it("maps finished success to deleted", () => {
    expect(eventStateRegistry.deleteIssue.getState(buildExecution())).toBe("deleted");
  });
});
