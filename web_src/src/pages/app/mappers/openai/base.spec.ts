import { describe, expect, it } from "vitest";

import { baseMapper } from "./base";
import type { ComponentBaseContext, ComponentDefinition, NodeInfo } from "../types";

const defaultDefinition: ComponentDefinition = {
  name: "openai.textPrompt",
  label: "Text Prompt",
  description: "",
  icon: "sparkles",
  color: "gray",
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "openai.textPrompt",
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

describe("openai baseMapper node metadata", () => {
  it("shows a code execution badge when the toggle is on", () => {
    const props = baseMapper.props(
      buildContext(buildNode({ configuration: { model: "gpt-5.2", codeInterpreter: true } })),
    );
    expect(props.metadata).toEqual([
      { icon: "sparkles", label: "gpt-5.2" },
      { icon: "terminal", label: "Code interpreter" },
    ]);
  });

  it("omits the badge when the toggle is off", () => {
    const props = baseMapper.props(buildContext(buildNode({ configuration: { model: "gpt-5.2" } })));
    expect(props.metadata).toEqual([{ icon: "sparkles", label: "gpt-5.2" }]);
  });

  it("falls back to node metadata when the node has no configuration yet", () => {
    const props = baseMapper.props(
      buildContext(buildNode({ configuration: undefined, metadata: { model: "gpt-5.2", codeInterpreter: true } })),
    );
    expect(props.metadata).toContainEqual({ icon: "terminal", label: "Code interpreter" });
  });

  it("prefers the live configuration over stale metadata", () => {
    // Autosave updates configuration only, so metadata can lag behind.
    const props = baseMapper.props(
      buildContext(buildNode({ configuration: { model: "gpt-5.2" }, metadata: { codeInterpreter: true } })),
    );
    expect(props.metadata).toEqual([{ icon: "sparkles", label: "gpt-5.2" }]);
  });

  it("shows structured output and code execution together", () => {
    const props = baseMapper.props(
      buildContext(buildNode({ configuration: { model: "gpt-5.2", outputSchema: "{}", codeInterpreter: true } })),
    );
    expect(props.metadata).toEqual([
      { icon: "sparkles", label: "gpt-5.2" },
      { icon: "braces", label: "Structured output" },
      { icon: "terminal", label: "Code interpreter" },
    ]);
  });
});
