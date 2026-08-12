import { describe, expect, it } from "vitest";

import { addReactionMapper } from "./add_reaction";
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
    componentName: "linear.addReaction",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return { type: "linear.reaction", timestamp: new Date().toISOString(), data };
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
      name: "linear.addReaction",
      label: "Add Reaction",
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

const reactionPayload = {
  id: "c9e1f7a3",
  emoji: "eyes",
  createdAt: "2026-03-27T16:42:11.123Z",
  user: { id: "u1", name: "Ada Lovelace", displayName: "ada" },
};

describe("addReactionMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => addReactionMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("surfaces the emoji, the reaction id and the author", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(reactionPayload)] } } });
    const details = addReactionMapper.getExecutionDetails(ctx);

    expect(Object.keys(details)[0]).toBe("Executed At");
    expect(details["Reaction"]).toBe("eyes");
    expect(details["Reaction ID"]).toBe("c9e1f7a3");
    expect(details["Author"]).toBe("ada");
    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
  });

  it("omits the id when a conflict meant no reaction came back", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({ emoji: "eyes" })] } } });
    const details = addReactionMapper.getExecutionDetails(ctx);

    expect(details["Reaction"]).toBe("eyes");
    expect(details["Reaction ID"]).toBeUndefined();
  });
});

describe("addReactionMapper.subtitle", () => {
  it("uses the emoji when present", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: { default: [buildOutput(reactionPayload)] } } });
    expect(addReactionMapper.subtitle!(ctx)).toBe("eyes");
  });

  it("does not throw without outputs", () => {
    const ctx = buildSubtitleCtx({ execution: { outputs: undefined } });
    expect(() => addReactionMapper.subtitle!(ctx)).not.toThrow();
  });
});

describe("addReactionMapper.props", () => {
  it("shows the issue for the issue target", () => {
    const props = addReactionMapper.props(
      buildComponentContext({ node: { configuration: { target: "issue", issue: "ENG-142", emoji: "eyes" } } }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "hash", label: "ENG-142" });
    expect(props.metadata?.[1]).toEqual({ icon: "smile", label: "eyes" });
  });

  it("shows the comment for the comment target", () => {
    const props = addReactionMapper.props(
      buildComponentContext({
        node: { configuration: { target: "comment", comment: "c1", issue: "ENG-142", emoji: "tada" } },
      }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "hash", label: "c1" });
  });

  it("does not throw when configuration is empty", () => {
    expect(() => addReactionMapper.props(buildComponentContext())).not.toThrow();
  });
});

describe("eventStateRegistry", () => {
  it("registers addReaction", () => {
    expect(eventStateRegistry.addReaction).toBeDefined();
  });
});
