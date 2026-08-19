import { describe, expect, it } from "vitest";

import { getIssueMapper } from "./get_issue";
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
    componentName: "linear.getIssue",
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
      name: "linear.getIssue",
      label: "Get Issue",
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

describe("getIssueMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getIssueMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when the default channel is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => getIssueMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("always includes Executed At first", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    const details = getIssueMapper.getExecutionDetails(ctx);
    expect(Object.keys(details)[0]).toBe("Executed At");
    expect(details["Executed At"]).not.toBe("-");
  });

  it("shows a dash for Executed At when createdAt is missing", () => {
    const ctx = buildDetailsCtx({ execution: { createdAt: undefined, outputs: undefined } });
    expect(getIssueMapper.getExecutionDetails(ctx)["Executed At"]).toBe("-");
  });

  it("extracts the issue fields that matter, including the link", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(issuePayload)] } } });
    const details = getIssueMapper.getExecutionDetails(ctx);

    expect(details["Issue"]).toBe("ENG-142");
    expect(details["Issue URL"]).toBe("https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry");
    expect(details["Title"]).toBe("Deploy pipeline fails on retry");
    expect(details["Status"]).toBe("In Progress");
    expect(details["Assignee"]).toBe("jane");
  });

  it("shows at most six details with the timestamp first", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(issuePayload)] } } });
    const details = getIssueMapper.getExecutionDetails(ctx);

    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
    expect(Object.keys(details)[0]).toBe("Executed At");
  });

  it("omits missing fields rather than padding with dashes", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ identifier: "ENG-1", title: "Only a title" })] } },
    });
    const details = getIssueMapper.getExecutionDetails(ctx);

    expect(details["Issue"]).toBe("ENG-1");
    expect(details["Title"]).toBe("Only a title");
    expect(details["Status"]).toBeUndefined();
    expect(details["Assignee"]).toBeUndefined();
    expect(details["Issue URL"]).toBeUndefined();
  });
});

describe("getIssueMapper.props", () => {
  it("renders the configured issue as metadata", () => {
    const props = getIssueMapper.props(buildComponentContext({ node: { configuration: { issue: "ENG-142" } } }));
    expect(props.metadata).toEqual([{ icon: "hash", label: "ENG-142" }]);
  });

  it("does not throw when configuration is undefined", () => {
    expect(() => getIssueMapper.props(buildComponentContext({ node: { configuration: undefined } }))).not.toThrow();
  });
});

describe("getIssueMapper.subtitle", () => {
  it("returns the issue label when the payload has an identifier and title", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: { default: [buildOutput(issuePayload)] } } });
    expect(getIssueMapper.subtitle(ctx)).toBe("ENG-142 · Deploy pipeline fails on retry");
  });

  it("falls back to a time-ago element when there is no payload", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: undefined } });
    expect(getIssueMapper.subtitle(ctx)).not.toBe("");
  });

  it("returns an empty string with neither payload nor createdAt", () => {
    const ctx = buildSubtitleCtx({ execution: { createdAt: undefined, outputs: undefined } });
    expect(getIssueMapper.subtitle(ctx)).toBe("");
  });
});

describe("eventStateRegistry.getIssue", () => {
  it("maps a finished success to retrieved", () => {
    expect(eventStateRegistry.getIssue.getState(buildExecution())).toBe("retrieved");
  });

  it("returns running while the execution is in progress", () => {
    const execution = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });

    expect(eventStateRegistry.getIssue.getState(execution)).toBe("running");
  });
});
