import { describe, expect, it } from "vitest";

import { runAgentMapper } from "./run_agent";
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
  name: "claude.runAgent",
  label: "Run Managed Agent",
  description: "",
  icon: "bot",
  color: "#C9784D",
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "claude.runAgent",
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
  return { type: "claude.runAgent", timestamp: new Date().toISOString(), data };
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

describe("runAgentMapper.getExecutionDetails", () => {
  it("does not throw when outputs are missing", () => {
    expect(() =>
      runAgentMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs: undefined } })),
    ).not.toThrow();
    expect(() =>
      runAgentMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs: { default: [] } } })),
    ).not.toThrow();
  });

  it("surfaces executed-at first, then status, session id, and artifacts", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              status: "completed",
              sessionId: "session_011",
              lastMessage: "Done.",
              messages: [],
              artifacts: [{ fileId: "file_1", filename: "report.pdf" }, { fileId: "file_2" }],
            }),
          ],
        },
      },
    });
    const details = runAgentMapper.getExecutionDetails(ctx);
    expect(Object.keys(details)[0]).toBe("Executed At");
    expect(details["Status"]).toBe("completed");
    expect(details["Session ID"]).toBe("session_011");
    expect(details["Artifacts"]).toBe("report.pdf, file_2");
    expect(details["Last Message"]).toBeUndefined();
    expect(Object.keys(details)).toHaveLength(4);
  });

  it("omits the artifacts entry when the run produced none", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { default: [buildOutput({ status: "completed", sessionId: "session_011", artifacts: [] })] },
      },
    });
    const details = runAgentMapper.getExecutionDetails(ctx);
    expect(details["Artifacts"]).toBeUndefined();
  });
});

describe("runAgentMapper.props", () => {
  it("uses the definition icon with a bot fallback", () => {
    const props = runAgentMapper.props(buildPropsContext());
    expect(props.iconSlug).toBe("bot");
    expect(props.title).toBe("Test Node");
  });

  it("falls back to the definition label when the node has no name", () => {
    const props = runAgentMapper.props(buildPropsContext({ node: buildNode({ name: "" }) }));
    expect(props.title).toBe("Run Managed Agent");
  });
});

describe("eventStateRegistry.runAgent", () => {
  it("maps finished passed to completed", () => {
    expect(eventStateRegistry.runAgent.getState(buildExecution())).toBe("completed");
  });
});
