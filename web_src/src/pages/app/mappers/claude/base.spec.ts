import { describe, expect, it } from "vitest";

import { baseMapper } from "./base";
import type { ComponentBaseContext, ComponentDefinition, NodeInfo } from "../types";

const defaultDefinition: ComponentDefinition = {
  name: "claude.textPrompt",
  label: "Text Prompt",
  description: "",
  icon: "message-square",
  color: "orange",
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "claude.textPrompt",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildContext(node: NodeInfo): ComponentBaseContext {
  return {
    nodes: [],
    node,
    componentDefinition: defaultDefinition,
    lastExecutions: [],
    currentUser: { id: "user-1", name: "Test User", email: "test@example.com", roles: [], groups: [] },
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("claude baseMapper node metadata", () => {
  it("shows a code execution badge when the toggle is on", () => {
    const props = baseMapper.props(
      buildContext(buildNode({ configuration: { model: "claude-x", codeExecution: true } })),
    );
    expect(props.metadata).toEqual([
      { icon: "sparkles", label: "claude-x" },
      { icon: "terminal", label: "Code execution" },
    ]);
  });

  it("omits the badge when the toggle is off", () => {
    const props = baseMapper.props(buildContext(buildNode({ configuration: { model: "claude-x" } })));
    expect(props.metadata).toEqual([{ icon: "sparkles", label: "claude-x" }]);
  });

  it("falls back to node metadata when the node has no configuration yet", () => {
    const props = baseMapper.props(
      buildContext(buildNode({ configuration: undefined, metadata: { model: "claude-x", codeExecution: true } })),
    );
    expect(props.metadata).toContainEqual({ icon: "terminal", label: "Code execution" });
  });

  it("prefers the live configuration over stale metadata", () => {
    // Autosave updates configuration only, so metadata can lag behind.
    const props = baseMapper.props(
      buildContext(buildNode({ configuration: { model: "claude-x" }, metadata: { codeExecution: true } })),
    );
    expect(props.metadata).toEqual([{ icon: "sparkles", label: "claude-x" }]);
  });

  it("shows structured output and code execution together", () => {
    const props = baseMapper.props(
      buildContext(buildNode({ configuration: { model: "claude-x", outputSchema: "{}", codeExecution: true } })),
    );
    expect(props.metadata).toEqual([
      { icon: "sparkles", label: "claude-x" },
      { icon: "braces", label: "Structured output" },
      { icon: "terminal", label: "Code execution" },
    ]);
  });
});
