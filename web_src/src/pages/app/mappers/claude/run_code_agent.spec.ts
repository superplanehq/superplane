import { describe, expect, it } from "vitest";

import { runCodeAgentMapper } from "./run_code_agent";
import { eventStateRegistry } from "./index";
import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";

const defaultDefinition: ComponentDefinition = {
  name: "claude.runCodeAgent",
  label: "Run Code Agent",
  description: "",
  icon: "bot",
  color: "#C9784D",
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "claude.runCodeAgent",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
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

function buildOutput(data: unknown): OutputPayload {
  return { type: "claude.runCodeAgent", timestamp: new Date().toISOString(), data };
}

function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

function buildPropsContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return {
    nodes: [],
    node: buildNode(),
    componentDefinition: defaultDefinition,
    lastExecutions: [],
    currentUser: { id: "user-1", name: "Test User", email: "test@example.com", roles: [], groups: [] },
    actions: { invokeNodeExecutionHook: async () => {} },
    ...overrides,
  };
}

describe("runCodeAgentMapper.getExecutionDetails", () => {
  it("does not throw when outputs are missing", () => {
    expect(() =>
      runCodeAgentMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs: undefined } })),
    ).not.toThrow();
    expect(() =>
      runCodeAgentMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs: { default: [] } } })),
    ).not.toThrow();
  });

  it("surfaces executed-at first, then status, pull request, and branch", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              status: "idle",
              prUrl: "https://github.com/o/r/pull/7",
              branch: "claude/agent-abc",
              summary: "Did the work.",
            }),
          ],
        },
      },
    });
    const details = runCodeAgentMapper.getExecutionDetails(ctx);
    expect(Object.keys(details)[0]).toBe("Executed At");
    expect(details["Status"]).toBe("idle");
    expect(details["Pull Request"]).toBe("https://github.com/o/r/pull/7");
    expect(details["Branch"]).toBe("claude/agent-abc");
    expect(details["Summary"]).toBeUndefined();
    expect(details["Emitted At"]).toBeUndefined();
  });
});

describe("runCodeAgentMapper.props", () => {
  it("shows repository and base branch from node metadata (repository mode)", () => {
    const props = runCodeAgentMapper.props(
      buildPropsContext({
        node: buildNode({
          metadata: { sourceMode: "repository", repository: "acme/widgets", baseBranch: "main", model: "claude-x" },
        }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "git-branch", label: "acme/widgets" },
      { icon: "git-branch", label: "main" },
      { icon: "bot", label: "claude-x" },
    ]);
  });

  it("shows the pull request in PR mode", () => {
    const props = runCodeAgentMapper.props(
      buildPropsContext({
        node: buildNode({ metadata: { sourceMode: "pr", prUrl: "https://github.com/o/r/pull/9" } }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "git-pull-request", label: "https://github.com/o/r/pull/9" }]);
  });

  it("falls back to configuration (repository and base branch) when metadata is absent", () => {
    const props = runCodeAgentMapper.props(
      buildPropsContext({
        node: buildNode({
          metadata: {},
          configuration: { sourceMode: "repository", repository: "acme/widgets", baseBranch: "develop" },
        }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "git-branch", label: "acme/widgets" },
      { icon: "git-branch", label: "develop" },
    ]);
  });

  it("coerces IntegrationResourceRef repository and model objects to their display names", () => {
    const props = runCodeAgentMapper.props(
      buildPropsContext({
        node: buildNode({
          metadata: {
            sourceMode: "repository",
            repository: { id: "repo-id", name: "acme/widgets", type: "repository" },
            model: { id: "model-id", name: "claude-opus-4-6", type: "model" },
          },
        }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "git-branch", label: "acme/widgets" },
      { icon: "bot", label: "claude-opus-4-6" },
    ]);
  });
});

describe("eventStateRegistry.runCodeAgent", () => {
  it("maps finished passed to completed", () => {
    expect(eventStateRegistry.runCodeAgent.getState(buildExecution())).toBe("completed");
  });
});
