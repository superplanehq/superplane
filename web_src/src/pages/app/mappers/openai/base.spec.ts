import { describe, expect, it } from "vitest";

import { baseMapper } from "./base";
import type { ComponentBaseContext, ComponentDefinition, NodeInfo } from "../types";

const definition: ComponentDefinition = {
  name: "openai.response",
  label: "Response",
  description: "",
  icon: "sparkles",
  color: "#10A37F",
};

describe("openai baseMapper.props", () => {
  it("shows the model from configuration", () => {
    const props = baseMapper.props(buildPropsContext({ node: buildNode({ configuration: { model: "gpt-4o" } }) }));

    expect(props.metadata).toEqual([{ icon: "sparkles", label: "gpt-4o" }]);
  });

  it("coerces an IntegrationResourceRef model object to its display name instead of crashing", () => {
    const props = baseMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { model: { id: "model-id", name: "gpt-4o", type: "model" } } }),
      }),
    );

    expect(props.metadata).toEqual([{ icon: "sparkles", label: "gpt-4o" }]);
  });
});

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Response",
    componentName: "openai.response",
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
