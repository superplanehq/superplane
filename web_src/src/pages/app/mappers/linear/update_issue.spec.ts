import { describe, expect, it } from "vitest";

import { updateIssueMapper } from "./update_issue";
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
    componentName: "linear.updateIssue",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "linear.issue",
    timestamp: new Date().toISOString(),
    data,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date("2026-03-26T19:29:35Z").toISOString(),
    updatedAt: new Date("2026-03-26T19:29:35Z").toISOString(),
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
      name: "linear.updateIssue",
      label: "Update Issue",
      description: "",
      icon: "linear",
      color: "indigo",
      ...overrides?.componentDefinition,
    },
    lastExecutions: overrides?.lastExecutions ?? [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

const issuePayload = {
  id: "2174add1",
  identifier: "ENG-142",
  title: "Deploy pipeline fails on retry",
  url: "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry",
  state: { id: "s1", name: "In Progress", type: "started" },
  team: { id: "t1", key: "ENG", name: "Engineering" },
  assignee: { id: "u1", name: "Jane Doe", displayName: "jane" },
};

describe("updateIssueMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => updateIssueMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts the issue fields that matter, including the link", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(issuePayload)] } } });
    const details = updateIssueMapper.getExecutionDetails(ctx);

    expect(details["Issue"]).toBe("ENG-142");
    expect(details["Issue URL"]).toBe("https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry");
    expect(details["Title"]).toBe("Deploy pipeline fails on retry");
    expect(details["Status"]).toBe("In Progress");
    expect(details["Assignee"]).toBe("jane");
  });

  it("shows at most six details with the timestamp first", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(issuePayload)] } } });
    const details = updateIssueMapper.getExecutionDetails(ctx);

    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
    expect(Object.keys(details)[0]).toBe("Executed At");
  });
});

describe("updateIssueMapper.props", () => {
  it("renders the team and the configured issue as metadata", () => {
    const props = updateIssueMapper.props(
      buildComponentContext({
        node: {
          configuration: { team: "t1", issue: "ENG-142" },
          metadata: { team: { id: "t1", key: "ENG", name: "Engineering" } },
        },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "users", label: "Engineering" },
      { icon: "hash", label: "ENG-142" },
    ]);
  });

  it("falls back to the configured team when metadata is empty", () => {
    const props = updateIssueMapper.props(
      buildComponentContext({ node: { configuration: { team: "t1", issue: "ENG-142" }, metadata: {} } }),
    );

    expect(props.metadata).toEqual([
      { icon: "users", label: "t1" },
      { icon: "hash", label: "ENG-142" },
    ]);
  });

  it("does not throw when metadata and configuration are undefined", () => {
    expect(() =>
      updateIssueMapper.props(buildComponentContext({ node: { configuration: undefined, metadata: undefined } })),
    ).not.toThrow();
  });
});

describe("updateIssueMapper.subtitle", () => {
  it("returns the issue label when the payload has an identifier and title", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: { default: [buildOutput(issuePayload)] } } });
    expect(updateIssueMapper.subtitle(ctx)).toBe("ENG-142 · Deploy pipeline fails on retry");
  });

  it("returns an empty string with neither payload nor createdAt", () => {
    const ctx = buildSubtitleCtx({ execution: { createdAt: undefined, outputs: undefined } });
    expect(updateIssueMapper.subtitle(ctx)).toBe("");
  });
});

describe("eventStateRegistry.updateIssue", () => {
  it("maps a finished success to updated", () => {
    expect(eventStateRegistry.updateIssue.getState(buildExecution())).toBe("updated");
  });

  it("returns running while the execution is in progress", () => {
    const execution = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });

    expect(eventStateRegistry.updateIssue.getState(execution)).toBe("running");
  });
});
