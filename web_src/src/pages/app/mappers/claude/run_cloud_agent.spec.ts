import { describe, expect, it } from "vitest";

import { runCloudAgentMapper } from "./run_cloud_agent";
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
  name: "claude.runCloudAgent",
  label: "Run Claude Cloud Agent",
  description: "",
  icon: "bot",
  color: "#C9784D",
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "claude.runCloudAgent",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "claude.runCloudAgent",
    timestamp: new Date().toISOString(),
    data,
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
    currentUser: {
      id: "user-1",
      name: "Test User",
      email: "test@example.com",
      roles: [],
      groups: [],
    },
    actions: {
      invokeNodeExecutionHook: async () => {},
    },
    ...overrides,
  };
}

describe("runCloudAgentMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => runCloudAgentMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => runCloudAgentMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when output data and metadata are empty", () => {
    const ctx = buildDetailsCtx({
      execution: { metadata: {}, outputs: { default: [buildOutput({})] } },
    });
    expect(runCloudAgentMapper.getExecutionDetails(ctx)).toEqual({
      "Emitted At": expect.any(String),
    });
  });

  it("extracts repository and branch from execution metadata and status/message from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        metadata: { repository: "https://github.com/owner/repo.git", branch: "main" },
        outputs: {
          default: [
            buildOutput({
              status: "idle",
              sessionId: "sess_1",
              lastMessage: "All done.",
            }),
          ],
        },
      },
    });
    const details = runCloudAgentMapper.getExecutionDetails(ctx);
    expect(details["Repository"]).toBe("https://github.com/owner/repo.git");
    expect(details["Branch"]).toBe("main");
    expect(details["Status"]).toBe("idle");
    expect(details["Last Message"]).toBe("All done.");
  });

  it("falls back to session status from metadata when output has no status", () => {
    const ctx = buildDetailsCtx({
      execution: {
        metadata: { session: { id: "sess_1", status: "terminated" } },
        outputs: { default: [buildOutput({})] },
      },
    });
    expect(runCloudAgentMapper.getExecutionDetails(ctx)["Status"]).toBe("terminated");
  });

  it("omits repository and branch when absent", () => {
    const ctx = buildDetailsCtx({
      execution: { metadata: {}, outputs: { default: [buildOutput({ status: "idle" })] } },
    });
    const details = runCloudAgentMapper.getExecutionDetails(ctx);
    expect(details["Repository"]).toBeUndefined();
    expect(details["Branch"]).toBeUndefined();
  });
});

describe("runCloudAgentMapper.props", () => {
  it("prefers resolved agent and environment names from node metadata", () => {
    const props = runCloudAgentMapper.props(
      buildPropsContext({
        node: buildNode({
          metadata: {
            agentId: "agent_01",
            agentName: "My Agent",
            environmentId: "env_01",
            environmentName: "My Env",
          },
          configuration: { agent: "agent_01", environmentId: "env_01" },
        }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "bot", label: "My Agent" },
      { icon: "box", label: "My Env" },
    ]);
  });

  it("falls back to configured ids when name metadata is absent", () => {
    const props = runCloudAgentMapper.props(
      buildPropsContext({
        node: buildNode({
          metadata: {},
          configuration: { agent: "agent_01", environmentId: "env_01" },
        }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "bot", label: "agent_01" },
      { icon: "box", label: "env_01" },
    ]);
  });

  it("includes the repository chip from configuration", () => {
    const props = runCloudAgentMapper.props(
      buildPropsContext({
        node: buildNode({
          metadata: { agentName: "My Agent", environmentName: "My Env" },
          configuration: {
            agent: "agent_01",
            environmentId: "env_01",
            repository: "https://github.com/owner/repo.git",
          },
        }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "bot", label: "My Agent" },
      { icon: "box", label: "My Env" },
      { icon: "git-branch", label: "https://github.com/owner/repo.git" },
    ]);
  });

  it("returns no metadata chips when nothing is configured", () => {
    const props = runCloudAgentMapper.props(buildPropsContext({}));
    expect(props.metadata).toEqual([]);
    expect(props.title).toBeDefined();
    expect(props.includeEmptyState).toBe(true);
  });
});

describe("eventStateRegistry.runCloudAgent", () => {
  it("maps finished passed to completed", () => {
    expect(eventStateRegistry.runCloudAgent.getState(buildExecution())).toBe("completed");
  });
});
