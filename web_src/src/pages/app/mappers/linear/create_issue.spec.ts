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
    componentName: "linear.createIssue",
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
      name: "linear.createIssue",
      label: "Create Issue",
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
  state: { id: "s1", name: "Todo", type: "unstarted" },
  team: { id: "t1", key: "ENG", name: "Engineering" },
  assignee: { id: "u1", name: "Jane Doe", displayName: "jane" },
};

describe("createIssueMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createIssueMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when the default channel is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => createIssueMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("always includes Executed At", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    const details = createIssueMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Executed At"]).not.toBe("-");
  });

  it("shows a dash for Executed At when createdAt is missing", () => {
    const ctx = buildDetailsCtx({ execution: { createdAt: undefined, outputs: undefined } });
    expect(createIssueMapper.getExecutionDetails(ctx)["Executed At"]).toBe("-");
  });

  it("extracts the issue fields that matter, including the link", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(issuePayload)] } } });
    const details = createIssueMapper.getExecutionDetails(ctx);

    expect(details["Issue"]).toBe("ENG-142");
    expect(details["Issue URL"]).toBe("https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry");
    expect(details["Title"]).toBe("Deploy pipeline fails on retry");
    expect(details["Status"]).toBe("Todo");
    expect(details["Assignee"]).toBe("jane");
  });

  it("shows at most six details", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(issuePayload)] } } });
    const details = createIssueMapper.getExecutionDetails(ctx);

    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
    expect(Object.keys(details)[0]).toBe("Executed At");
  });

  it("omits missing fields rather than padding with dashes", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ identifier: "ENG-1", title: "Only a title" })] } },
    });
    const details = createIssueMapper.getExecutionDetails(ctx);

    expect(details["Issue"]).toBe("ENG-1");
    expect(details["Title"]).toBe("Only a title");
    expect(details["Status"]).toBeUndefined();
    expect(details["Assignee"]).toBeUndefined();
    expect(details["Issue URL"]).toBeUndefined();
  });

  it("falls back to the assignee name when there is no display name", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { default: [buildOutput({ identifier: "ENG-1", assignee: { id: "u1", name: "Jane Doe" } })] },
      },
    });

    expect(createIssueMapper.getExecutionDetails(ctx)["Assignee"]).toBe("Jane Doe");
  });
});

describe("createIssueMapper.props", () => {
  it("renders the team from node metadata", () => {
    const props = createIssueMapper.props(
      buildComponentContext({
        node: {
          configuration: { team: "t1" },
          metadata: { team: { id: "t1", key: "ENG", name: "Engineering" } },
        },
      }),
    );

    expect(props.metadata).toEqual([{ icon: "users", label: "Engineering" }]);
  });

  it("falls back to the configured team when metadata is empty", () => {
    const props = createIssueMapper.props(
      buildComponentContext({ node: { configuration: { team: "t1" }, metadata: {} } }),
    );

    expect(props.metadata).toEqual([{ icon: "users", label: "t1" }]);
  });

  it("renders the priority label for the configured priority", () => {
    const props = createIssueMapper.props(
      buildComponentContext({
        node: {
          configuration: { team: "t1", priority: "1" },
          metadata: { team: { id: "t1", key: "ENG", name: "Engineering" } },
        },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "users", label: "Engineering" },
      { icon: "flag", label: "Urgent" },
    ]);
  });

  it("renders 'No priority' when priority is explicitly zero", () => {
    const props = createIssueMapper.props(
      buildComponentContext({ node: { configuration: { team: "t1", priority: "0" } } }),
    );

    expect((props.metadata || []).map((item) => item.label)).toContain("No priority");
  });

  it("omits the priority badge when no priority is configured", () => {
    const props = createIssueMapper.props(buildComponentContext({ node: { configuration: { team: "t1" } } }));

    expect(props.metadata).toEqual([{ icon: "users", label: "t1" }]);
  });

  it("does not throw when metadata and configuration are undefined", () => {
    expect(() =>
      createIssueMapper.props(buildComponentContext({ node: { configuration: undefined, metadata: undefined } })),
    ).not.toThrow();
  });
});

describe("createIssueMapper.subtitle", () => {
  it("returns the issue label when the payload has an identifier and title", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: { default: [buildOutput(issuePayload)] } } });
    expect(createIssueMapper.subtitle(ctx)).toBe("ENG-142 · Deploy pipeline fails on retry");
  });

  it("falls back to a time-ago element when there is no payload", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: undefined } });
    expect(createIssueMapper.subtitle(ctx)).not.toBe("");
  });

  it("returns an empty string with neither payload nor createdAt", () => {
    const ctx = buildSubtitleCtx({ execution: { createdAt: undefined, outputs: undefined } });
    expect(createIssueMapper.subtitle(ctx)).toBe("");
  });
});

describe("eventStateRegistry.createIssue", () => {
  it("maps a finished success to created", () => {
    expect(eventStateRegistry.createIssue.getState(buildExecution())).toBe("created");
  });

  it("returns running while the execution is in progress", () => {
    const execution = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });

    expect(eventStateRegistry.createIssue.getState(execution)).toBe("running");
  });
});
