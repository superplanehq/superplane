import { describe, expect, it } from "vitest";

import type { ComponentBaseContext, ComponentDefinition, NodeInfo } from "../types";
import { addReactionMapper } from "./add_reaction";

const definition: ComponentDefinition = {
  name: "github.addReaction",
  label: "Add Reaction",
  description: "",
  icon: "github",
  color: "gray",
};

describe("addReactionMapper.props", () => {
  it("shows the repository and reaction from configuration", () => {
    const props = addReactionMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { repository: "superplane", content: "+1" } }) }),
    );

    expect(props.metadata).toEqual([
      { icon: "book", label: "superplane" },
      { icon: "smile", label: "Reaction: 👍" },
    ]);
  });

  it("coerces an IntegrationResourceRef repository object to its display name instead of crashing", () => {
    const props = addReactionMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { repository: { id: "repo-id", name: "superplane", type: "repository" }, content: "heart" },
        }),
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "book", label: "superplane" },
      { icon: "smile", label: "Reaction: ❤️" },
    ]);
  });

  it("falls back to node metadata when configuration has no repository", () => {
    const props = addReactionMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: {},
          metadata: { repository: { id: "123", name: "superplane", url: "https://github.com/x/superplane" } },
        }),
      }),
    );

    expect(props.metadata).toEqual([{ icon: "book", label: "superplane" }]);
  });
});

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Add Reaction",
    componentName: "github.addReaction",
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
