import { describe, expect, it } from "vitest";

import { addIssueLabelMapper } from "./add_issue_label";
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
    componentName: "linear.addIssueLabel",
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
      name: "linear.addIssueLabel",
      label: "Add Issue Label",
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
  labels: [
    { id: "l1", name: "bug" },
    { id: "l2", name: "regression" },
  ],
};

describe("addIssueLabelMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => addIssueLabelMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("always includes Executed At first", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(issuePayload)] } } });
    const details = addIssueLabelMapper.getExecutionDetails(ctx);

    expect(details["Executed At"]).toBeDefined();
    expect(Object.keys(details)[0]).toBe("Executed At");
  });

  it("extracts the issue fields that matter, including the link", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(issuePayload)] } } });
    const details = addIssueLabelMapper.getExecutionDetails(ctx);

    expect(details["Issue"]).toBe("ENG-142");
    expect(details["Issue URL"]).toBe("https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry");
    expect(details["Status"]).toBe("In Progress");
  });

  it("shows at most six details", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(issuePayload)] } } });
    const details = addIssueLabelMapper.getExecutionDetails(ctx);

    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
  });

  it("omits missing fields rather than padding with dashes", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ identifier: "ENG-1" })] } },
    });
    const details = addIssueLabelMapper.getExecutionDetails(ctx);

    expect(details["Issue"]).toBe("ENG-1");
    expect(details["Issue URL"]).toBeUndefined();
    expect(details["Status"]).toBeUndefined();
  });
});

describe("addIssueLabelMapper.props", () => {
  it("renders the team and the configured issue", () => {
    const props = addIssueLabelMapper.props(
      buildComponentContext({
        node: {
          configuration: { team: "t1", issue: "ENG-142", labels: ["l1"] },
          metadata: { team: { id: "t1", key: "ENG", name: "Engineering" } },
        },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "users", label: "Engineering" },
      { icon: "hash", label: "ENG-142" },
    ]);
  });

  it("does not throw when metadata and configuration are undefined", () => {
    expect(() =>
      addIssueLabelMapper.props(buildComponentContext({ node: { configuration: undefined, metadata: undefined } })),
    ).not.toThrow();
  });
});

describe("addIssueLabelMapper.subtitle", () => {
  it("returns the issue label when the payload has an identifier and title", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: { default: [buildOutput(issuePayload)] } } });
    expect(addIssueLabelMapper.subtitle(ctx)).toBe("ENG-142 · Deploy pipeline fails on retry");
  });
});

describe("eventStateRegistry.addIssueLabel", () => {
  it("maps a finished success to labeled", () => {
    expect(eventStateRegistry.addIssueLabel.getState(buildExecution())).toBe("labeled");
  });

  it("returns running while the execution is in progress", () => {
    const execution = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });

    expect(eventStateRegistry.addIssueLabel.getState(execution)).toBe("running");
  });
});
