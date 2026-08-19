import { describe, expect, it } from "vitest";

import { removeIssueLabelMapper } from "./remove_issue_label";
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
    componentName: "linear.removeIssueLabel",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return { type: "linear.issue", timestamp: new Date().toISOString(), data };
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
  return { node: buildNode(overrides?.node), execution: buildExecution(overrides?.execution) };
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
      name: "linear.removeIssueLabel",
      label: "Remove Issue Label",
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
  id: "i1",
  identifier: "ENG-142",
  title: "Deploy pipeline fails on retry",
  url: "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry",
  labels: [{ id: "l2", name: "backend" }],
};

describe("removeIssueLabelMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => removeIssueLabelMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("reports the updated issue, timestamp first", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(issuePayload)] } } });
    const details = removeIssueLabelMapper.getExecutionDetails(ctx);

    expect(Object.keys(details)[0]).toBe("Executed At");
    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
  });
});

describe("removeIssueLabelMapper.subtitle", () => {
  it("does not throw without outputs", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: undefined } });
    expect(() => removeIssueLabelMapper.subtitle!(ctx)).not.toThrow();
  });
});

describe("removeIssueLabelMapper.props", () => {
  it("surfaces the team and the issue", () => {
    const props = removeIssueLabelMapper.props(
      buildComponentContext({
        node: {
          configuration: { team: "t1", issue: "ENG-142", labels: ["l1"] },
          metadata: { team: { id: "t1", key: "ENG", name: "Engineering" } },
        },
      }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "users", label: "Engineering" });
    expect(props.metadata?.[1]).toEqual({ icon: "hash", label: "ENG-142" });
  });

  it("does not throw when configuration is empty", () => {
    expect(() => removeIssueLabelMapper.props(buildComponentContext())).not.toThrow();
  });
});

describe("eventStateRegistry", () => {
  it("registers removeIssueLabel", () => {
    expect(eventStateRegistry.removeIssueLabel).toBeDefined();
  });
});
