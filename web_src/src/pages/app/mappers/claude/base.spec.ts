import { describe, expect, it } from "vitest";

import { baseMapper } from "./base";
import type { ComponentBaseContext, ComponentDefinition, NodeInfo } from "../types";

const definition: ComponentDefinition = {
  name: "claude.textPrompt",
  label: "Text Prompt",
  description: "",
  icon: "loader",
  color: "#C9784D",
};

describe("claude baseMapper.props", () => {
  it("shows the model from configuration", () => {
    const props = baseMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { model: "claude-opus-4-6" } }) }),
    );

    expect(props.metadata).toEqual([{ icon: "sparkles", label: "claude-opus-4-6" }]);
  });

  it("coerces an IntegrationResourceRef model object to its display name instead of crashing", () => {
    const props = baseMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { model: { id: "model-id", name: "claude-opus-4-6", type: "model" } } }),
      }),
    );

    expect(props.metadata).toEqual([{ icon: "sparkles", label: "claude-opus-4-6" }]);
  });

  it("shows structured output when an output schema is set", () => {
    const props = baseMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { model: "claude-opus-4-6", outputSchema: "{}" } }),
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "sparkles", label: "claude-opus-4-6" },
      { icon: "braces", label: "Structured output" },
    ]);
  });
});

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Text Prompt",
    componentName: "claude.textPrompt",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildPropsContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return {
    nodes: [],
    node: buildNode(),
    componentDefinition: definition,
    lastExecutions: [],
    currentUser: { id: "user-1", name: "Test User", email: "test@example.com", roles: [], groups: [] },
    actions: { invokeNodeExecutionHook: async () => {} },
    ...overrides,
  };
}
