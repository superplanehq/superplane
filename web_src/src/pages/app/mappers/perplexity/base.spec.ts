import { describe, expect, it } from "vitest";

import { baseMapper } from "./base";
import type { ComponentBaseContext, ComponentDefinition, NodeInfo } from "../types";

const definition: ComponentDefinition = {
  name: "perplexity.runAgent",
  label: "Run Agent",
  description: "",
  icon: "bot",
  color: "#20808D",
};

describe("perplexity baseMapper.props", () => {
  it("shows the model from configuration when no preset is set", () => {
    const props = baseMapper.props(buildPropsContext({ node: buildNode({ configuration: { model: "sonar-pro" } }) }));

    expect(props.metadata).toEqual([{ icon: "cpu", label: "sonar-pro" }]);
  });

  it("coerces an IntegrationResourceRef model object to its display name instead of crashing", () => {
    const props = baseMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { model: { id: "model-id", name: "sonar-pro", type: "model" } } }),
      }),
    );

    expect(props.metadata).toEqual([{ icon: "cpu", label: "sonar-pro" }]);
  });

  it("prefers the preset over the model", () => {
    const props = baseMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { preset: "Research", model: "sonar-pro" } }) }),
    );

    expect(props.metadata).toEqual([{ icon: "sparkles", label: "Research" }]);
  });
});

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Run Agent",
    componentName: "perplexity.runAgent",
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
